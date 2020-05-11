package controllers

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DemoReconciler reconciles a Demo object
type IMeshReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups="",resources=service,verbs=get;list;watch;create;update;patch;delete
func (r *IMeshReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	//ctx := context.Background()
	logs := r.Log.WithValues("service", req.NamespacedName)
	logs.Info("---GET Request for " + req.String())
	return ctrl.Result{}, nil
}

func (r *IMeshReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		WithEventFilter(&ServiceAnnotationPredicate{}).
		Complete(r)
}
