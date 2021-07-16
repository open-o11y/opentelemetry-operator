// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package controllers contains the main controller, where the reconciliation starts.
package controllers

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/open-telemetry/opentelemetry-operator/api/v1alpha1"
	"github.com/open-telemetry/opentelemetry-operator/internal/config"
	"github.com/open-telemetry/opentelemetry-operator/pkg/targetallocator/reconcile"
)

// OpenTelemetryTargetAllocatorReconciler reconciles a OpenTelemetryTargetAllocator object.
type OpenTelemetryTargetAllocatorReconciler struct {
	client.Client
	log    logr.Logger
	scheme *runtime.Scheme
	config config.Config
	tasks  []TgAlTask
}

// TgAlTask represents a reconciliation task to be executed by the reconciler.
type TgAlTask struct {
	Name        string
	Do          func(context.Context, reconcile.Params) error
	BailOnError bool
}

// TgAlParams is the set of options to build a new OpenTelemetryTargetAllocatorReconciler.
type TgAlParams struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Config config.Config
	Tasks  []TgAlTask
}

// NewTgAlReconciler creates a new reconciler for OpenTelemetryTargetAllocator objects.
func NewTgAlReconciler(p TgAlParams) *OpenTelemetryTargetAllocatorReconciler {
	if len(p.Tasks) == 0 {
		p.Tasks = []TgAlTask{
			{
				"config maps",
				reconcile.ConfigMaps,
				true,
			},
			{
				"deployments",
				reconcile.Deployments,
				true,
			},
			{
				"services",
				reconcile.Services,
				true,
			},
		}
	}

	return &OpenTelemetryTargetAllocatorReconciler{
		Client: p.Client,
		log:    p.Log,
		scheme: p.Scheme,
		config: p.Config,
		tasks:  p.Tasks,
	}
}

// Reconcile the current state of an OpenTelemetry target allocation resource with the desired state.
func (r *OpenTelemetryTargetAllocatorReconciler) Reconcile(_ context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.log.WithValues("opentelemtrytargetallocator", req.NamespacedName)

	var instance v1alpha1.OpenTelemetryCollector
	if err := r.Get(ctx, req.NamespacedName, &instance); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "unable to fetch OpenTelemetryTargetAllocator")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	params := reconcile.Params{
		Config:   r.config,
		Client:   r.Client,
		Instance: instance,
		Log:      log,
		Scheme:   r.scheme,
	}

	if err := r.RunTasks(ctx, params); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// RunTasks runs all the tasks associated with this reconciler.
func (r *OpenTelemetryTargetAllocatorReconciler) RunTasks(ctx context.Context, params reconcile.Params) error {
	for _, task := range r.tasks {
		if err := task.Do(ctx, params); err != nil {
			r.log.Error(err, fmt.Sprintf("failed to reconcile %s", task.Name))
			if task.BailOnError {
				return err
			}
		}
	}
	return nil
}

// SetupWithManager tells the manager what our controller is interested in.
func (r *OpenTelemetryTargetAllocatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.OpenTelemetryCollector{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}