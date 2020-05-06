package controllers

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type recontileRequest struct {
	reconcile.Request
	schema.GroupVersionKind
	Action string
}
