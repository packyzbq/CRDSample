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
	demov1 "github.com/CRDSample/api/v1"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/source"
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
	logs.Info("---GET Request for " + req.String())
	demo := &demov1.Demo{}

	//err := r.finalize(ctx, demo); if err != nil {
	//	logs.Error(err,"set Finalize error")
	//	return ctrl.Result{}, nil
	//}
	err := r.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, demo)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			logs.Error(err, "Get demo error", "demo", req.NamespacedName)
			return ctrl.Result{}, err
		} else {
			return ctrl.Result{}, nil
		}
	}

	if demo.Status.Phase == "" {
		demo.Status.Phase = demov1.PhasePending
	}

	switch demo.Status.Phase {
	case demov1.PhasePending:
		err := r.processPending(ctx, demo)
		if err != nil {
			logs.Error(err, "error")
		}
	case demov1.PhaseRunning:
		err := r.processRunning(ctx, demo)
		if err != nil {
			logs.Error(err, "error")
		}
	case demov1.PhaseError:
		return ctrl.Result{}, nil
	case demov1.PhaseDone:
		return ctrl.Result{}, nil
	default:
		logs.Info("NOP")
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *DemoReconciler) SetupWithManager(mgr ctrl.Manager) error {

	return ctrl.NewControllerManagedBy(mgr).
		For(&demov1.Demo{}).
		Watches(&source.Kind{Type: &v1.Deployment{}}, &EnqueueRequestForDP{}).
		WithEventFilter(&ResourceAnnotationPredicate{}).
		//Build(r)
		Complete(r)
}

func (c *DemoReconciler) getLog(demo *demov1.Demo) logr.Logger {
	return c.Log.WithValues("Name", demo.Name, "NameSpace", demo.Namespace)
}

func (r *DemoReconciler) processDeploy(ctx context.Context, demo *demov1.Demo, deployment *v1.Deployment) error {
	r.getLog(demo).Info("Process deploy event" + deployment.Name)
	//r.finalizeDeploy(deployment, demo)
	//if err := r.Update(ctx, deployment); err != nil {
	//	r.getLog(demo).Error(err,"update deployment error")
	//	return err
	//}
	if deployment.Status.UnavailableReplicas >= 1 {
		r.Recorder.Eventf(demo, corev1.EventTypeWarning, "Error", "unavailabelReplicas >= 1")
	}
	return nil
}

func (c *DemoReconciler) processPending(ctx context.Context, demo *demov1.Demo) error {
	logs := c.getLog(demo)
	// Create deployment
	deployment := &v1.Deployment{}
	err := c.Get(ctx, client.ObjectKey{Name: demo.Spec.DeployNew.Name, Namespace: demo.Namespace}, deployment)
	if apierrors.IsNotFound(err) {
		logs.Info("cannot find deployment, create one. Name=" + demo.Spec.DeployNew.Name + " Namespace=" + demo.Namespace)
		deployment = c.buildDeployment(demo)
		if err = c.Client.Create(ctx, deployment); err != nil {
			logs.Error(err, "failed to create Deployment")
			c.Recorder.Event(demo, "Error", "Deploy", err.Error())
			return err
		}

		c.Recorder.Eventf(demo, corev1.EventTypeNormal, "Created", "Created deployment %q", deployment.Name)
		demo.Status.Phase = demov1.PhaseRunning
	} else {
		return err
	}
	if err := c.Status().Update(ctx, demo); err != nil {
		logs.Error(err, "Update demo status error")
	}
	return nil
}

func (r *DemoReconciler) processRunning(ctx context.Context, demo *demov1.Demo) error {
	logs := r.getLog(demo)
	logs.Info("deploy already exists, checking replica count")
	deployment := &v1.Deployment{}
	expectedReplicas := demo.Spec.DeployNew.Replicas
	err := r.Get(ctx, client.ObjectKey{Name: demo.Spec.DeployNew.Name, Namespace: demo.Namespace}, deployment)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.Recorder.Eventf(demo, corev1.EventTypeWarning, "WARN", "cannot found deploy %q", demo.Spec.DeployNew.Name)
			return nil
		} else {
			logs.Error(err, "processRunning get deploy error")
			return err
		}
	}
	if deployment.Status.UnavailableReplicas >= 1 {
		//demo.Status.Phase = demov1.PhaseDone
		r.Recorder.Eventf(demo, corev1.EventTypeWarning, "WARN", "deployment %q unavaliable replicas > 1", deployment.Name)
		//if err := r.Status().Update(ctx, demo); err != nil {
		//	logs.Error(err, "Update demo status error")
		//}
	} else if *deployment.Spec.Replicas != expectedReplicas {
		logs.Info("updating replica count", "old_count", *deployment.Spec.Replicas, "new_count", expectedReplicas)
		deployment.Spec.Replicas = &expectedReplicas
		if err := r.Client.Update(ctx, deployment); err != nil {
			logs.Error(err, "failed to Deployment update replica count")
			return err
		}

		r.Recorder.Eventf(demo, corev1.EventTypeNormal, "Scaled", "Scaled deployment %q to %d replicas", deployment.Name, expectedReplicas)
	}
	return nil
}

func (c *DemoReconciler) finalize(ctx context.Context, demo *demov1.Demo) error {
	logs := c.getLog(demo)
	myFinalizerName := "demo.finalizers.packy.io"

	if demo.ObjectMeta.DeletionTimestamp.IsZero() {
		if !containsString(demo.ObjectMeta.Finalizers, myFinalizerName) {
			demo.ObjectMeta.Finalizers = append(demo.ObjectMeta.Finalizers, myFinalizerName)
		}
	} else {
		if containsString(demo.ObjectMeta.Finalizers, myFinalizerName) {
			if err := c.cleanupOwneredDeployment(ctx, logs, demo); err != nil {
				return err
			}

			// remove our finalizer from the list and update it.
			demo.ObjectMeta.Finalizers = removeString(demo.ObjectMeta.Finalizers, myFinalizerName)
		}

		// Stop reconciliation as the item is being deleted
		return nil
	}
	return nil
}

func (c *DemoReconciler) cleanupOwneredDeployment(ctx context.Context, log logr.Logger, demo *demov1.Demo) error {
	log.Info("finding existing Deployments for MyKind resource")

	// List all deployment resources owned by this MyKind
	var depitem demov1.DeploySepc
	if demo.Spec.DeployOld.Final {
		depitem = demo.Spec.DeployOld
	} else {
		depitem = demo.Spec.DeployNew
	}

	dep := &v1.Deployment{}
	err := c.Get(ctx, client.ObjectKey{Name: depitem.Name, Namespace: demo.Namespace}, dep)
	if err != nil {
		log.Error(err, "Do not find deployment "+depitem.Name)
		c.Recorder.Eventf(demo, corev1.EventTypeWarning, "Finalize", "Deleted deployment %q error", depitem.Name)
		return err
	}
	if err := c.Client.Delete(ctx, dep); err != nil {
		log.Error(err, "failed to delete Deployment resource")
		return err
	}

	c.Recorder.Eventf(demo, corev1.EventTypeNormal, "Deleted", "Deleted deployment %q", depitem.Name)

	return nil
}

func (r *DemoReconciler) buildDeployment(demo *demov1.Demo) *v1.Deployment {
	annotations := make(map[string]string)
	annotations[gray_annotation] = ""
	deployment := v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        demo.Spec.DeployNew.Name,
			Namespace:   demo.Namespace,
			Annotations: annotations,
		},
		Spec: v1.DeploymentSpec{
			Replicas: &demo.Spec.DeployNew.Replicas,
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
							Image: "busybox-fda",
						},
					},
				},
			},
		},
	}
	_ = controllerutil.SetControllerReference(demo, &deployment, r.Scheme)
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
