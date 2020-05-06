package controllers

import (
	"fmt"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var enqueueLog = ctrl.Log.WithName("crdEventHandler")

type CrdEventHandler struct {
	// groupKind is the cached Group and Kind from OwnerType
	groupKind schema.GroupKind

	// mapper maps GroupVersionKinds to Resources
	mapper meta.RESTMapper
}

func (e *CrdEventHandler) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	if evt.Meta == nil {
		enqueueLog.Error(nil, "CreateEvent received with no metadata", "event", evt)
		return
	}
	request := e.getOwnerReconcileRequest(evt.Meta)
	customRequest := recontileRequest{
		Request:          request[0],
		GroupVersionKind: evt.Object.GetObjectKind().GroupVersionKind(),
		Action:           "Create",
	}
	q.Add(customRequest)
	enqueueLog.Info("Add queue", "name", customRequest.Request.Name, "Kind", customRequest.Kind, "Action", customRequest.Action)
}

func (e *CrdEventHandler) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	if evt.MetaNew != nil {
		request := e.getOwnerReconcileRequest(evt.MetaNew)
		customRequest := recontileRequest{Request: request[0],
			GroupVersionKind: evt.ObjectOld.GetObjectKind().GroupVersionKind(),
			Action:           "Update"}
		q.Add(customRequest)
		enqueueLog.Info("Add queue", "name", customRequest.Request.Name, "Kind", customRequest.Kind, "Action", customRequest.Action)
	} else {
		enqueueLog.Error(nil, "UpdateEvent received with no new metadata", "event", evt)
	}

}

func (e *CrdEventHandler) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	if evt.Meta == nil {
		enqueueLog.Error(nil, "CreateEvent received with no metadata", "event", evt)
		return
	}
	request := e.getOwnerReconcileRequest(evt.Meta)
	customRequest := recontileRequest{Request: request[0],
		GroupVersionKind: evt.Object.GetObjectKind().GroupVersionKind(),
		Action:           "Delete"}
	q.Add(customRequest)
	enqueueLog.Info("Add queue", "name", customRequest.Request.Name, "Kind", customRequest.Kind, "Action", customRequest.Action)

}

func (e *CrdEventHandler) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	if evt.Meta == nil {
		enqueueLog.Error(nil, "GenericEvent received with no metadata", "event", evt)
		return
	}
	request := e.getOwnerReconcileRequest(evt.Meta)
	customRequest := recontileRequest{
		Request:          request[0],
		GroupVersionKind: evt.Object.GetObjectKind().GroupVersionKind(),
		Action:           "Generic",
	}
	q.Add(customRequest)
	enqueueLog.Info("Add queue", "name", customRequest.Request.Name, "Kind", customRequest.Kind, "Action", customRequest.Action)
}

// getOwnerReconcileRequest looks at object and returns a slice of reconcile.Request to reconcile
// owners of object that match e.OwnerType.
func (e *CrdEventHandler) getOwnerReconcileRequest(object metav1.Object) []reconcile.Request {
	// Iterate through the OwnerReferences looking for a match on Group and Kind against what was requested
	// by the user
	var result []reconcile.Request
	for _, ref := range e.getOwnersReferences(object) {
		// Parse the Group out of the OwnerReference to compare it to what was parsed out of the requested OwnerType
		refGV, err := schema.ParseGroupVersion(ref.APIVersion)
		if err != nil {
			enqueueLog.Error(err, "Could not parse OwnerReference APIVersion",
				"api version", ref.APIVersion)
			return nil
		}

		// Compare the OwnerReference Group and Kind against the OwnerType Group and Kind specified by the user.
		// If the two match, create a Request for the objected referred to by
		// the OwnerReference.  Use the Name from the OwnerReference and the Namespace from the
		// object in the event.
		if ref.Kind == e.groupKind.Kind && refGV.Group == e.groupKind.Group {
			// Match found - add a Request for the object referred to in the OwnerReference
			request := reconcile.Request{NamespacedName: types.NamespacedName{
				Name: ref.Name,
			}}

			// if owner is not namespaced then we should set the namespace to the empty
			mapping, err := e.mapper.RESTMapping(e.groupKind, refGV.Version)
			if err != nil {
				enqueueLog.Error(err, "Could not retrieve rest mapping", "kind", e.groupKind)
				return nil
			}
			if mapping.Scope.Name() != meta.RESTScopeNameRoot {
				request.Namespace = object.GetNamespace()
			}

			result = append(result, request)
		}
	}

	// Return the matches
	return result
}

// getOwnersReferences returns the OwnerReferences for an object as specified by the EnqueueRequestForOwner
// - if IsController is true: only take the Controller OwnerReference (if found)
// - if IsController is false: take all OwnerReferences
func (e *CrdEventHandler) getOwnersReferences(object metav1.Object) []metav1.OwnerReference {
	if object == nil {
		return nil
	}

	// If filtered to a Controller, only take the Controller OwnerReference
	if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
		return []metav1.OwnerReference{*ownerRef}
	}
	// No Controller OwnerReference found
	return nil
}
