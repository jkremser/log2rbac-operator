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
	"github.com/jkremser/log2rbac-operator/internal"
	"io"
	apps "k8s.io/api/apps/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"
	"time"

	core "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kremserv1 "github.com/jkremser/log2rbac-operator/api/v1"
)

// RbacEventHandler handles the CRUD event from CR
type RbacEventHandler struct {
	client.Client
	clientset *kubernetes.Clientset
	Recorder  record.EventRecorder
	Config    *internal.Config
}

// AppInfo bundles the application specific information including logs, service account and list of live pods
type AppInfo struct {
	serviceAccount string
	log            string
	livePods       []core.Pod
}

func (r *RbacEventHandler) handleResource(ctx context.Context, resource kremserv1.RbacNegotiation) ctrl.Result {
	appInfo, err := r.getAppInfo(ctx, resource.Spec)
	if err != nil {
		log.Log.Error(err, "Unable to get logs from underlying pod. Check the ReplicaSet (k describe) if the service account isn't missing.")
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: time.Duration(r.Config.Controller.SyncIntervalAfterNoLogsSeconds) * time.Second,
		}
	}

	//if logs.contain that string edit the role to contain that exception and restart the pod
	missingRbacEntry := FindRbacEntry(appInfo.log, resource.Spec.For.Namespace, appInfo.serviceAccount)
	// todo: update status sub-resource on the cr
	if missingRbacEntry != nil {
		log.Log.Info(fmt.Sprintf("Rbac entry: %#v", missingRbacEntry))
		err := r.addMissingRbacEntry(ctx, resource.Spec.For.Namespace, appInfo.serviceAccount, missingRbacEntry, resource.Spec.Role)
		if err != nil {
			log.Log.Error(err, "Unable to add missing rbac entry")
			return ctrl.Result{
				Requeue:      true,
				RequeueAfter: time.Duration(r.Config.Controller.SyncIntervalAfterErrorMinutes) * time.Minute,
			}
		}
		r.emitEvent(ctx, resource, missingRbacEntry)
		tryAgainInSeconds := r.Config.Controller.SyncIntervalAfterPodRestartSeconds
		if r.Config.Controller.ShouldRestartAppPods {
			r.restartPods(ctx, appInfo.livePods)
		} else {
			// pod is going to be restarted anyway, but exp backoff can make this quite a long process
			tryAgainInSeconds *= 4
		}
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: time.Duration(r.Config.Controller.SyncIntervalAfterPodRestartSeconds) * time.Second,
		}
	}
	retryMinutes := r.Config.Controller.SyncIntervalAfterNoRbacEntryMinutes
	log.Log.Info(fmt.Sprintf("No rbac related stuff has been found in the logs. Will try again in %d minutes..", retryMinutes))
	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: time.Duration(retryMinutes) * time.Minute,
	}
}

func (r *RbacEventHandler) restartPods(ctx context.Context, pods []core.Pod) {
	for _, pod := range pods {
		if err := r.Client.Delete(ctx, &pod); err != nil {
			log.Log.Error(err, fmt.Sprintf("Unable to restart pod: %s", pod.GetName()))
		}
	}
}

func (r *RbacEventHandler) addMissingRbacEntry(ctx context.Context, ns string, sa string, entry *RbacEntry, role kremserv1.RoleSpec) error {
	if role.IsClusterRole {
		crol := rbac.ClusterRole{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: role.Name}, &crol); err != nil {
			if !role.CreateIfNotExist {
				log.Log.Error(err, "Unable to read cluster role")
				return err
			}

			// create cluster role
			rol := &rbac.ClusterRole{
				Rules: []rbac.PolicyRule{rbacEntryToPolicyRule(entry)},
			}
			rol.SetName(role.Name)
			if err := r.Client.Create(ctx, rol); err != nil {
				log.Log.Error(err, "Unable to create cluster role")
				return err
			}
			rb := &rbac.ClusterRoleBinding{
				Subjects: []rbac.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      sa,
						Namespace: ns,
					},
				},
				RoleRef: rbac.RoleRef{
					Kind: "ClusterRole",
					Name: role.Name,
				},
			}
			rb.SetName(role.Name + "-binding")
			if err := r.Client.Create(ctx, rb); err != nil && !errors.IsAlreadyExists(err) {
				log.Log.Error(err, "Unable to create cluster role binding")
				return err
			}
			return nil
		}
		// todo: consolidate the rules
		crol.Rules = append(crol.Rules, rbacEntryToPolicyRule(entry))
		if err := r.Client.Update(ctx, &crol); err != nil {
			log.Log.Error(err, "Unable to update cluster role")
		}
	} else {
		nrol := rbac.Role{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: role.Name}, &nrol); err != nil {
			if !role.CreateIfNotExist {
				log.Log.Error(err, "Unable to read role")
				return err
			}

			// create role
			rol := &rbac.Role{
				Rules: []rbac.PolicyRule{rbacEntryToPolicyRule(entry)},
			}
			rol.SetName(role.Name)
			rol.SetNamespace(ns)
			if err := r.Client.Create(ctx, rol); err != nil {
				log.Log.Error(err, "Unable to create role")
				return err
			}
			rb := &rbac.RoleBinding{
				Subjects: []rbac.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      sa,
						Namespace: ns,
					},
				},
				RoleRef: rbac.RoleRef{
					Kind: "Role",
					Name: role.Name,
				},
			}
			rb.SetName(role.Name + "-binding")
			rb.SetNamespace(ns)
			if err := r.Client.Create(ctx, rb); err != nil && !errors.IsAlreadyExists(err) {
				log.Log.Error(err, "Unable to create role binding")
				return err
			}
			return nil
		}
		nrol.Rules = append(nrol.Rules, rbacEntryToPolicyRule(entry))
		if err := r.Client.Update(ctx, &nrol); err != nil {
			log.Log.Error(err, "Unable to update role")
		}
	}
	return nil
}

func rbacEntryToPolicyRule(entry *RbacEntry) rbac.PolicyRule {
	return rbac.PolicyRule{
		APIGroups: []string{entry.Object.Group},
		Verbs:     []string{entry.Verb},
		Resources: []string{entry.Object.Kind},
	}
}

func (r *RbacEventHandler) getAppInfo(ctx context.Context, resource kremserv1.RbacNegotiationSpec) (*AppInfo, error) {
	forS := resource.For
	var selector map[string]string
	var sa string
	if len(resource.ServiceAccountName) == 0 || len(forS.PodSelector) == 0 {
		selector, sa = r.getSelectorAndSA(ctx, forS)
	}
	if len(resource.ServiceAccountName) > 0 {
		sa = resource.ServiceAccountName
	}
	if len(forS.PodSelector) > 0 {
		selector = forS.PodSelector
	}
	if len(sa) == 0 {
		sa = "default"
	}
	if selector == nil {
		return nil, fmt.Errorf("cannot get pod selector for resource %v", resource)
	}
	var podList core.PodList
	if err := r.Client.List(ctx, &podList, client.InNamespace(forS.Namespace), client.MatchingLabels(selector)); err != nil {
		return nil, err
	}
	pods := podList.Items
	if len(pods) == 0 {
		return nil, fmt.Errorf("no pods found for %s called '%s'", forS.Kind, forS.Name)
	}
	podName := pods[0].GetName()
	req := r.ClientSet().CoreV1().Pods(forS.Namespace).GetLogs(podName, &core.PodLogOptions{})
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return nil, err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return nil, err
	}
	str := buf.String()

	return &AppInfo{log: str, serviceAccount: sa, livePods: pods}, nil
}

func (r *RbacEventHandler) getSelectorAndSA(ctx context.Context, resource kremserv1.ForSpec) (map[string]string, string) {
	nsName := client.ObjectKey{
		Namespace: resource.Namespace,
		Name:      resource.Name,
	}
	switch strings.ToLower(resource.Kind) {
	case "deploy", "deployment":
		dep := apps.Deployment{}
		return r.getObject(ctx, &dep, nsName)
	case "rs", "replicaset":
		rs := apps.ReplicaSet{}
		return r.getObject(ctx, &rs, nsName)
	case "ds", "daemonset":
		ds := apps.DaemonSet{}
		return r.getObject(ctx, &ds, nsName)
	case "ss", "statefulset":
		ss := apps.StatefulSet{}
		return r.getObject(ctx, &ss, nsName)
	case "svc", "service":
		svc := core.Service{}
		return r.getObject(ctx, &svc, nsName)
	default:
		log.Log.Error(fmt.Errorf("unrecognized kind: '%s'", resource.Kind), "")
		return nil, ""
	}
}

func (r *RbacEventHandler) getObject(ctx context.Context, obj client.Object, nsName client.ObjectKey) (map[string]string, string) {
	if err := r.Client.Get(ctx, nsName, obj); err != nil {
		log.Log.Error(err, "Cannot get k8s object %s with name %v ", obj.GetObjectKind(), nsName)
		return nil, ""
	}
	switch obj.(type) {
	case *apps.Deployment:
		return (obj.(*apps.Deployment)).Spec.Selector.MatchLabels, (obj.(*apps.Deployment)).Spec.Template.Spec.ServiceAccountName
	case *apps.ReplicaSet:
		return (obj.(*apps.ReplicaSet)).Spec.Selector.MatchLabels, (obj.(*apps.ReplicaSet)).Spec.Template.Spec.ServiceAccountName
	case *apps.DaemonSet:
		return (obj.(*apps.DaemonSet)).Spec.Selector.MatchLabels, (obj.(*apps.DaemonSet)).Spec.Template.Spec.ServiceAccountName
	case *apps.StatefulSet:
		return (obj.(*apps.StatefulSet)).Spec.Selector.MatchLabels, (obj.(*apps.StatefulSet)).Spec.Template.Spec.ServiceAccountName
	case *core.Service:
		return (obj.(*core.Service)).Spec.Selector, ""
	}

	return nil, ""
}

// ClientSet returns the k8s client
func (r *RbacEventHandler) ClientSet() *kubernetes.Clientset {
	if r.clientset == nil {
		r.clientset = SetupK8sClient()
	}
	return r.clientset
}

func (r *RbacEventHandler) emitEvent(ctx context.Context, resource kremserv1.RbacNegotiation, entry *RbacEntry) {
	r.Recorder.Eventf(&resource, "Normal", "RbacEntryCreated", "New RBAC entry: "+
		"role=%s, verb=%s, resource=%s, group=%s", resource.Spec.Role.Name, entry.Verb, entry.Object.Kind, entry.Object.Group)
}
