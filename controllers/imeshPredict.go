package controllers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var istio_annotation = "istio.packy.io/imesh"

type ServiceAnnotationPredicate struct {
}

func (r *ServiceAnnotationPredicate) Create(e event.CreateEvent) bool {
	return r.predict(e.Meta)
}

func (r *ServiceAnnotationPredicate) Update(e event.UpdateEvent) bool {
	return r.predict(e.MetaNew)
}

func (r *ServiceAnnotationPredicate) Delete(e event.DeleteEvent) bool {
	return r.predict(e.Meta)
}

func (r *ServiceAnnotationPredicate) Generic(e event.GenericEvent) bool {
	return r.predict(e.Meta)
}

func (r *ServiceAnnotationPredicate) predict(obj metav1.Object) bool {
	return containsAnnotation(obj, istio_annotation)
}
