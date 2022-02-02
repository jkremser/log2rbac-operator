/*
Copyright 2022.

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
	"fmt"
	"jkremser/log2rbac-operator/internal"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kremserv1 "jkremser/log2rbac-operator/api/v1"
)

// RbacNegotiationReconciler reconciles a RbacNegotiation object
type RbacNegotiationReconciler struct {
	client.Client
	handler  *RbacEventHandler
	Scheme   *runtime.Scheme
	Config   *internal.Config
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=kremser.dev,resources=rbacnegotiations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kremser.dev,resources=rbacnegotiations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kremser.dev,resources=rbacnegotiations/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac,resources={serviceaccounts,roles,clusterroles,rolebindings,clusterrolebindings},verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *RbacNegotiationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	log.Log.Info("fetching RbacNegotiation resource")
	rbacNeg := kremserv1.RbacNegotiation{}
	if err := r.Client.Get(ctx, req.NamespacedName, &rbacNeg); err != nil {
		log.Log.Error(err, "failed to get RbacNegotiation resource")
		// Ignore NotFound errors as they will be retried automatically if the
		// resource is created in the future.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Log.Info(fmt.Sprintf("New rbac negotiation event: for %s '%s'", strings.ToLower(rbacNeg.Spec.For.Kind), rbacNeg.Spec.For.Name))
	result := r.handler.handleResource(ctx, rbacNeg)
	return result, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RbacNegotiationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.handler = &RbacEventHandler{
		Client:    r.Client,
		clientset: SetupK8sClient(),
		Recorder:  r.Recorder,
		Config: r.Config,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&kremserv1.RbacNegotiation{}).
		Complete(r)
}
