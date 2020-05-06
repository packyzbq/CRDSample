package controllers

import (
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type CrdRequest struct {
	reconcile.Request
	Kind   source.Kind
	Action string
}
