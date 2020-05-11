package controllers

import (
	"fmt"
	demov1 "github.com/CRDSample/api/v1"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var enqueueLog = ctrl.Log.WithName("CRD Enqueue Request")

type EnqueueRequestForCRD struct{}

func (e *EnqueueRequestForCRD) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	if evt.Meta == nil {
		enqueueLog.Error(nil, "CreateEvent received with no metadata", "event", evt)
		return
	}
	request := reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      evt.Meta.GetName(),
		Namespace: evt.Meta.GetNamespace(),
	}}
	kind, err := judgeKind(evt.Object)
	if err != nil {
		enqueueLog.Error(err, "Create Event judge object error")
		return
	}
	//crdRequest := CrdRequest{
	//	Request:          request,
	//	Kind:  			  kind,
	//	Action:           "Create",
	//}
	enqueueLog.Info("Create Event for kind: " + kind.String())
	q.Add(request)
}

func (e *EnqueueRequestForCRD) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	if evt.MetaNew == nil {
		enqueueLog.Error(nil, "CreateEvent received with no metadata", "event", evt)
		return
	}
	request := reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      evt.MetaNew.GetName(),
		Namespace: evt.MetaNew.GetNamespace(),
	}}
	kind, err := judgeKind(evt.ObjectNew)
	if err != nil {
		enqueueLog.Error(err, "Update Event judge object error")
		return
	}
	//crdRequest := CrdRequest{
	//	Request:          request,
	//	Kind:  			  kind,
	//	Action:           "Update",
	//}
	enqueueLog.Info("Update Event for kind " + kind.String())
	q.Add(request)
}

func (e *EnqueueRequestForCRD) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	if evt.Meta == nil {
		enqueueLog.Error(nil, "CreateEvent received with no metadata", "event", evt)
		return
	}
	request := reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      evt.Meta.GetName(),
		Namespace: evt.Meta.GetNamespace(),
	}}
	kind, err := judgeKind(evt.Object)
	if err != nil {
		enqueueLog.Error(err, "Delete Event judge object error")
		return
	}
	//crdRequest := CrdRequest{
	//	Request:          request,
	//	Kind:  			  kind,
	//	Action:           "Delete",
	//}
	enqueueLog.Info("Delete Event for kind " + kind.String())
	q.Add(request)
}

func (e *EnqueueRequestForCRD) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	enqueueLog.Info("Generic Event ")
	if evt.Meta == nil {
		enqueueLog.Error(nil, "CreateEvent received with no metadata", "event", evt)
		return
	}
	request := reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      evt.Meta.GetName(),
		Namespace: evt.Meta.GetNamespace(),
	}}
	//kind, err := judgeKind(evt.Object)
	//if err != nil {
	//	enqueueLog.Error(err, "Generic Event judge object error")
	//	return
	//}

	q.Add(request)
}

func judgeKind(obj runtime.Object) (source.Kind, error) {
	_, ok := obj.(*v1.Deployment)
	if ok {
		return source.Kind{Type: &v1.Deployment{}}, nil
	}
	_, ok = obj.(*v1.StatefulSet)
	if ok {
		return source.Kind{Type: &v1.StatefulSet{}}, nil
	}
	_, ok = obj.(*demov1.Demo)
	if ok {
		return source.Kind{Type: &demov1.Demo{}}, nil
	}
	return source.Kind{Type: nil}, fmt.Errorf("Cannot judge object type: %s", obj.GetObjectKind().GroupVersionKind().String())
}
