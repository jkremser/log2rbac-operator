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
	"bytes"
	"context"
	"fmt"
	"io"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
	"strings"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kremserv1 "jkremser/log2rbac-operator/api/v1"
)

// RbacNegotiationReconciler reconciles a RbacNegotiation object
type RbacNegotiationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=kremser.dev,resources=rbacnegotiations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kremser.dev,resources=rbacnegotiations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kremser.dev,resources=rbacnegotiations/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac,resources={serviceaccounts,roles,clusterroles,rolebindings,clusterrolebindings},verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RbacNegotiation object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
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
	log.Log.Info(fmt.Sprintf("New rbac negotiation event: for kind=%s and name=%s", rbacNeg.Spec.For.Kind, rbacNeg.Spec.For.Name))
	r.handleResource(ctx, rbacNeg.Spec.For)
	return ctrl.Result{}, nil
}

func (r *RbacNegotiationReconciler) handleResource(ctx context.Context, resource kremserv1.ForSpec) {
	logs, err := r.logsFromFirstPod(ctx, resource)
	if err != nil {
		log.Log.Error(err, "Unable to get logs from underlying pod")
	}
	log.Log.Info(logs)
	//if logs.contain that string edit the role to contain that exception and restart the pod
}

func (r *RbacNegotiationReconciler) logsFromFirstPod(ctx context.Context, resource kremserv1.ForSpec) (string, error) {
	var selector map[string]string
	switch strings.ToLower(resource.Kind) {
	case "deploy", "deployment", "deployments":
		nsName := client.ObjectKey{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		}
		dep := apps.Deployment{}
		if err := r.Client.Get(ctx, nsName, &dep); err != nil {
			return "", err
		}
		selector = dep.Spec.Selector.MatchLabels
	default:
		return "", fmt.Errorf("unrecognized kind: '%s'", resource.Kind)
	}

	var podList core.PodList
	if err := r.Client.List(ctx, &podList, client.InNamespace(resource.Namespace), client.MatchingLabels(selector)); err != nil {
		return "", err
	}
	pods := podList.Items
	if len(pods) == 0 {
		return "", fmt.Errorf("no pods found for %s called '%s'", resource.Kind, resource.Name)
	}
	podName := pods[0].GetName()
	req := r.K8sClient().CoreV1().Pods(resource.Namespace).GetLogs(podName, &core.PodLogOptions{})
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", err
	}
	str := buf.String()

	return str, nil
}

func (r *RbacNegotiationReconciler) K8sClient() *kubernetes.Clientset {
	var config *rest.Config
	var err error
	_, inCluster := os.LookupEnv("KUBERNETES_SERVICE_HOST")
	if !inCluster {
		var kubeconfig string
		if home := homedir.HomeDir(); home != "" {
			log.Log.Info("Using kubeconfig from:" + filepath.Join(home, ".kube", "config"))
			kubeconfig = filepath.Join(home, ".kube", "config")
		}

		// use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err)
		}
	} else {
		// creates the in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err)
		}
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	return clientset
}

// SetupWithManager sets up the controller with the Manager.
func (r *RbacNegotiationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kremserv1.RbacNegotiation{}).
		Complete(r)
}
