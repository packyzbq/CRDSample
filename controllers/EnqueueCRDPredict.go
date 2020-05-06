package controllers

import (
	demov1 "github.com/CRDSample/api/v1"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var gray_annotation = "k8s.packy.io/gray"

type ResourceAnnotationPredicate struct {
}

func (r *ResourceAnnotationPredicate) Create(e event.CreateEvent) bool {
	return predict(e.Meta)
}

func (r *ResourceAnnotationPredicate) Update(e event.UpdateEvent) bool {
	return predict(e.MetaNew)
}

func (r *ResourceAnnotationPredicate) Delete(e event.DeleteEvent) bool {
	return predict(e.Meta)
}

func (r *ResourceAnnotationPredicate) Generic(e event.GenericEvent) bool {
	return predict(e.Meta)
}

func predict(obj metav1.Object) bool {
	_, ok := obj.(*demov1.Demo)
	if ok {
		return true
	}
	_, ok = obj.(*v1.Deployment)
	if ok {
		return containsAnnotation(obj, gray_annotation)
	}
	return false
}

func containsAnnotation(obj metav1.Object, annotation string) bool {
	annotations := obj.GetAnnotations()
	_, ok := annotations[annotation]
	if ok {
		return true
	} else {
		return false
	}
}
