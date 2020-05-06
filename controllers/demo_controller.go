/*
Copyright 2020 packyzbq.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"github.com/prometheus/common/log"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/source"

	demov1 "github.com/CRDSample/api/v1"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	deployOwnerKey = ".metadata.controller"
	apiGV          = demov1.GroupVersion
)

// DemoReconciler reconciles a Demo object
type DemoReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=demo.packy.io,resources=demoes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=demo.packy.io,resources=demoes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=demo.packy.io,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

func (r *DemoReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	logs := r.Log.WithValues("demo", req.NamespacedName)
	fmt.Println("--------------------------------")
	demo := &demov1.Demo{}
	if err := r.Get(ctx, req.NamespacedName, demo); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	myFinalizerName := "demo.finalizers.packy.io"

	if demo.ObjectMeta.DeletionTimestamp.IsZero() {
		if !containsString(demo.ObjectMeta.Finalizers, myFinalizerName) {
			demo.ObjectMeta.Finalizers = append(demo.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(context.Background(), demo); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if containsString(demo.ObjectMeta.Finalizers, myFinalizerName) {
			if err := r.cleanupOwneredDeployment(ctx, logs, demo); err != nil {
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			demo.ObjectMeta.Finalizers = removeString(demo.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.Update(context.Background(), demo); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}
	if demo.Status.Status == "" {
		demo.Status.Status = "Created"
		if err := r.Status().Update(ctx, demo); err != nil {
			log.Error(err, ":unable update status")
		}
		r.Recorder.Event(demo, "Normal", "Created", "example something")
		// Create deployment
		deployment := &v1.Deployment{}
		err := r.Get(ctx, client.ObjectKey{Name: demo.Name + "-deploy", Namespace: demo.Namespace}, deployment)
		if apierrors.IsNotFound(err) {
			logs.Info("cannot find deployment, create one...")
			deployment = buildDeployment(*demo)
			if err := r.Client.Create(ctx, deployment); err != nil {
				logs.Error(err, "failed to create Deployment")
				r.Recorder.Event(demo, "Error", "Deploy", err.Error())
				return ctrl.Result{}, err
			}

			r.Recorder.Eventf(demo, corev1.EventTypeNormal, "Created", "Created deployment %q", deployment.Name)
			logs.Info("create deploy for demo")
			return ctrl.Result{}, nil
		}
		logs.Info("deploy already exists, checking replica count")
		expectedReplicas := demo.Spec.Replicas
		if deployment.Spec.Replicas != &expectedReplicas {
			logs.Info("updating replica count", "old_count", *deployment.Spec.Replicas, "new_count", expectedReplicas)
			deployment.Spec.Replicas = &expectedReplicas
			if err := r.Client.Update(ctx, deployment); err != nil {
				log.Error(err, "failed to Deployment update replica count")
				return ctrl.Result{}, err
			}

			r.Recorder.Eventf(demo, corev1.EventTypeNormal, "Scaled", "Scaled deployment %q to %d replicas", deployment.Name, expectedReplicas)
		}

	} else {
		deployment := &v1.Deployment{}
		err := r.Get(ctx, client.ObjectKey{Name: demo.Name + "-deploy", Namespace: demo.Namespace}, deployment)
		if err != nil {
			logs.Error(err, "cannot get deployment")
			return ctrl.Result{}, err
		}
		if deployment.Status.UnavailableReplicas >= 1 {
			r.Recorder.Eventf(demo, corev1.EventTypeWarning, "Deployment Error", "Deployment unavalible >= 1")
		}
	}

	return ctrl.Result{}, nil
}

func (r *DemoReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(&v1.Deployment{}, deployOwnerKey, func(rawObj runtime.Object) []string {
		// grab the Deployment object, extract the owner...
		depl := rawObj.(*v1.Deployment)
		owner := metav1.GetControllerOf(depl)
		if owner == nil {
			return nil
		}
		// ...make sure it's a MyKind...
		if owner.APIVersion != apiGV.String() || owner.Kind != "MyKind" {
			return nil
		}

		// ...and if so, return it
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&demov1.Demo{}).
		Watches(&source.Kind{Type: &demov1.Demo{}}, &CrdEventHandler{}).
		Complete(r)
}

func (r *DemoReconciler) cleanupOwneredDeployment(ctx context.Context, log logr.Logger, demo *demov1.Demo) error {
	log.Info("finding existing Deployments for MyKind resource")

	// List all deployment resources owned by this MyKind
	var deployments v1.DeploymentList
	if err := r.List(ctx, &deployments, client.InNamespace(demo.Namespace), client.MatchingField(deployOwnerKey, demo.Name)); err != nil {
		return err
	}

	deleted := 0
	for _, depl := range deployments.Items {
		if depl.Name == demo.Name+"-deploy" {
			// If this deployment's name matches the one on the MyKind resource
			// then do not delete it.
			continue
		}

		if err := r.Client.Delete(ctx, &depl); err != nil {
			log.Error(err, "failed to delete Deployment resource")
			return err
		}

		r.Recorder.Eventf(demo, corev1.EventTypeNormal, "Deleted", "Deleted deployment %q", depl.Name)
		deleted++
	}

	log.Info("finished cleaning up old Deployment resources", "number_deleted", deleted)

	return nil
}

func buildDeployment(demo demov1.Demo) *v1.Deployment {
	deployment := v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:            demo.Name + "-deploy",
			Namespace:       demo.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(&demo, apiGV.WithKind("Demo"))},
		},
		Spec: v1.DeploymentSpec{
			Replicas: &demo.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "demo-deploy",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "demo-deploy",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: demo.Spec.Image,
						},
					},
				},
			},
		},
	}
	return &deployment
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
