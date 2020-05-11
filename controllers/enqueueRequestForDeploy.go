package controllers

import (
	"fmt"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type EnqueueRequestForDP struct{}

// Create implements EventHandler
func (e *EnqueueRequestForDP) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	if evt.Meta == nil {
		enqueueLog.Error(nil, "CreateEvent received with no metadata", "event", evt)
		return
	}
	fmt.Printf("Create Event for object: %s\n", evt.Meta.GetName())
	deploy := evt.Object.(*v1.Deployment)
	for _, owner := range deploy.OwnerReferences {
		if owner.Kind == "Demo" {
			q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      owner.Name,
				Namespace: evt.Meta.GetNamespace(),
			}})
		}
	}
}

// Update implements EventHandler
func (e *EnqueueRequestForDP) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	fmt.Printf("Update Event for object: %s\n", evt.MetaOld.GetName())
	if evt.MetaNew != nil {
		deploy := evt.ObjectNew.(*v1.Deployment)
		for _, owner := range deploy.OwnerReferences {
			if owner.Kind == "Demo" {
				q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
					Name:      owner.Name,
					Namespace: evt.MetaNew.GetNamespace(),
				}})
			}
		}
	} else {
		enqueueLog.Error(nil, "UpdateEvent received with no new metadata", "event", evt)
	}
}

// Delete implements EventHandler
func (e *EnqueueRequestForDP) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	fmt.Printf("Delete Event for object: %s\n", evt.Meta.GetName())
	if evt.Meta == nil {
		enqueueLog.Error(nil, "DeleteEvent received with no metadata", "event", evt)
		return
	}
	deploy := evt.Object.(*v1.Deployment)
	for _, owner := range deploy.OwnerReferences {
		if owner.Kind == "Demo" {
			q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      owner.Name,
				Namespace: evt.Meta.GetNamespace(),
			}})
		}
	}
}

// Generic implements EventHandler
func (e *EnqueueRequestForDP) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	fmt.Printf("Generic Event for object: %s\n", evt.Meta.GetName())
	if evt.Meta == nil {
		enqueueLog.Error(nil, "GenericEvent received with no metadata", "event", evt)
		return
	}
	deploy := evt.Object.(*v1.Deployment)
	for _, owner := range deploy.OwnerReferences {
		if owner.Kind == "Demo" {
			q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      owner.Name,
				Namespace: evt.Meta.GetNamespace(),
			}})
		}
	}
}
