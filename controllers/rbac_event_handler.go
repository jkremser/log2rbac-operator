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
	"reflect"
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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.6.1"
	"go.opentelemetry.io/otel/trace"
)

// RbacEventHandler handles the CRUD event from CR
type RbacEventHandler struct {
	client.Client
	clientset *kubernetes.Clientset
	Recorder  record.EventRecorder
	Config    *internal.Config
	Tracer    trace.Tracer
}

// AppInfo bundles the application specific information including logs, service account and list of live pods
type AppInfo struct {
	serviceAccount string
	log            string
	livePods       []core.Pod
}

// Setup Add some initialization stuff in here
func (r *RbacEventHandler) Setup(ctx context.Context) {
	r.Tracer = otel.GetTracerProvider().Tracer(
		"github.com/jkremser/log2rbac-operator",
		trace.WithInstrumentationVersion(r.Config.App.Version),
		trace.WithSchemaURL(semconv.SchemaURL))
	// create dummy spam on start
	_, span := r.Tracer.Start(ctx, "setup")
	span.End()
}

func (r *RbacEventHandler) handleResource(ctx context.Context, resource *kremserv1.RbacNegotiation) ctrl.Result {
	// tracing
	newCtx, span := r.Tracer.Start(ctx, "handleResource")
	span.SetAttributes(attribute.String("resource.name", resource.Name))
	span.SetAttributes(attribute.String("resource.ns", resource.Namespace))
	defer span.End()

	appInfo, err := r.getAppInfo(newCtx, resource.Spec)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		if !strings.Contains(fmt.Sprint(err), "ContainerCreating") {
			UpdateStatus(r.Client, ctx, resource, true, false)
		}
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
		err := r.addMissingRbacEntry(newCtx, resource.Spec.For.Namespace, appInfo.serviceAccount, missingRbacEntry, resource.Spec.Role)
		if err != nil {
			log.Log.Error(err, "Unable to add missing rbac entry")
			UpdateStatus(r.Client, ctx, resource, true, false)

			return ctrl.Result{
				Requeue:      true,
				RequeueAfter: time.Duration(r.Config.Controller.SyncIntervalAfterErrorMinutes) * time.Minute,
			}
		}
		r.emitEvent(*resource, missingRbacEntry)
		tryAgainInSeconds := r.Config.Controller.SyncIntervalAfterPodRestartSeconds
		if r.Config.Controller.ShouldRestartAppPods {
			r.restartPods(newCtx, appInfo.livePods)
		} else {
			// pod is going to be restarted anyway, but the default exp backoff can make this quite a long process
			tryAgainInSeconds *= 4
		}
		UpdateStatus(r.Client, ctx, resource, false, true)
		return ctrl.Result{
			Requeue:      true,
			RequeueAfter: time.Duration(tryAgainInSeconds) * time.Second,
		}
	}
	retryMinutes := r.Config.Controller.SyncIntervalAfterNoRbacEntryMinutes
	log.Log.Info(fmt.Sprintf("No rbac related stuff has been found in the logs. Will try again in %d minutes..", retryMinutes))

	UpdateStatus(r.Client, ctx, resource, false, false)
	return ctrl.Result{
		Requeue:      true,
		RequeueAfter: time.Duration(retryMinutes) * time.Minute,
	}
}

func (r *RbacEventHandler) restartPods(ctx context.Context, pods []core.Pod) {
	_, span := r.Tracer.Start(ctx, "restartPods")
	defer span.End()
	for _, pod := range pods {
		if err := r.Client.Delete(ctx, &pod); err != nil {
			log.Log.Error(err, fmt.Sprintf("Unable to restart pod: %s", pod.GetName()))
		}
	}
}

func (r *RbacEventHandler) addMissingRbacEntry(ctx context.Context, ns string, sa string, entry *RbacEntry, role kremserv1.RoleSpec) error {
	var span trace.Span
	ctx, span = r.Tracer.Start(ctx, "addMissingRbacEntry")
	span.SetAttributes(attribute.String("entry.verb", entry.Verb))
	span.SetAttributes(attribute.String("entry.kind", entry.Object.Kind))
	defer span.End()

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
			rol.ObjectMeta.Annotations = map[string]string{internal.CreatedByAnnotationKey: internal.CreatedByAnnotationValue}
			c, clusterRoleSpan := r.Tracer.Start(ctx, "createClusterRole")
			if err := r.Client.Create(c, rol); err != nil {
				log.Log.Error(err, "Unable to create cluster role")
				return err
			}
			clusterRoleSpan.End()
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
			rb.ObjectMeta.Annotations = map[string]string{internal.CreatedByAnnotationKey: internal.CreatedByAnnotationValue}
			_, clusterRoleBSpan := r.Tracer.Start(c, "createClusterRoleBinding")
			if err := r.Client.Create(ctx, rb); err != nil && !errors.IsAlreadyExists(err) {
				log.Log.Error(err, "Unable to create cluster role binding")
				return err
			}
			clusterRoleBSpan.End()
			return nil
		}
		// todo: consolidate the rules
		crol.Rules = append(crol.Rules, rbacEntryToPolicyRule(entry))
		c, clusterRoleUpSpan := r.Tracer.Start(ctx, "updateClusterRole")
		if err := r.Client.Update(c, &crol); err != nil {
			log.Log.Error(err, "Unable to update cluster role")
		}
		clusterRoleUpSpan.End()
	} else {
		nrol := rbac.Role{}
		if err := r.Client.Get(ctx, client.ObjectKey{Name: role.Name, Namespace: ns}, &nrol); err != nil {
			if !role.CreateIfNotExist {
				log.Log.Error(err, "Unable to read role")
				return err
			}

			// create role
			rol := &rbac.Role{
				Rules: []rbac.PolicyRule{rbacEntryToPolicyRule(entry)},
			}
			rol.SetName(role.Name)
			rol.ObjectMeta.Annotations = map[string]string{internal.CreatedByAnnotationKey: internal.CreatedByAnnotationValue}
			rol.SetNamespace(ns)
			c, roleSpan := r.Tracer.Start(ctx, "createRole")
			if err := r.Client.Create(c, rol); err != nil {
				log.Log.Error(err, "Unable to create role")
				return err
			}
			roleSpan.End()
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
			rb.ObjectMeta.Annotations = map[string]string{internal.CreatedByAnnotationKey: internal.CreatedByAnnotationValue}
			rb.SetNamespace(ns)
			c, roleBSpan := r.Tracer.Start(c, "createClusterRoleBinding")
			if err := r.Client.Create(c, rb); err != nil && !errors.IsAlreadyExists(err) {
				log.Log.Error(err, "Unable to create role binding")
				return err
			}
			roleBSpan.End()
			return nil
		}
		nrol.Rules = append(nrol.Rules, rbacEntryToPolicyRule(entry))
		c, roleUpSpan := r.Tracer.Start(ctx, "updateClusterRole")
		if err := r.Client.Update(c, &nrol); err != nil {
			log.Log.Error(err, "Unable to update role")
		}
		roleUpSpan.End()
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
	// tracing
	_, span := r.Tracer.Start(ctx, "getAppInfo")
	defer span.End()

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
		k := strings.ToLower(forS.Kind)
		if k == "svc" || k == "service" {
			// unsupported, use-case. For svc we need to know the sa
			return nil, fmt.Errorf("cannot get service account from a service. You need to specify it explicitly in the custom resource")
		}
		sa = "default"
	} else {
		r.createSAIfNotExists(ctx, sa, forS.Namespace)
	}
	if selector == nil {
		return nil, fmt.Errorf("cannot get pod selector for resource %s / %s", resource.For.Kind, resource.For.Name)
	}
	log.Log.Info(fmt.Sprintf("selector: %+v", selector))
	var podList core.PodList
	if err := r.Client.List(ctx, &podList, client.InNamespace(forS.Namespace), client.MatchingLabels(selector)); err != nil {
		return nil, err
	}
	pods := podList.Items
	if len(pods) == 0 {
		return nil, fmt.Errorf("no pods found for %s called '%s'", forS.Kind, forS.Name)
	}
	pod := pods[0]
	podName := pod.GetName()
	containerName := getContainerName(pod)
	req := r.ClientSet().CoreV1().Pods(forS.Namespace).GetLogs(podName, &core.PodLogOptions{Container: containerName})
	podLogs, err := req.Stream(ctx)
	if err != nil {
		if strings.Contains(fmt.Sprint(err), "waiting to start: ContainerCreating") {
			log.Log.Info(fmt.Sprintf("Unable to get logs from underlying pod. Pod %s is still starting (ContainerCreating)", podName))
			return nil, fmt.Errorf("pod %s is still starting (ContainerCreating)", podName)
		}
		log.Log.V(1).Info("Check the ReplicaSet if the service account isn't missing.")
		log.Log.Error(err, "Unable to get logs from underlying pod.")
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

func (r *RbacEventHandler) createSAIfNotExists(ctx context.Context, saName string, ns string) {
	_, span := r.Tracer.Start(ctx, "createServiceAccount")
	defer span.End()
	sa := core.ServiceAccount{}
	if err := r.Client.Get(ctx, client.ObjectKey{Name: saName, Namespace: ns}, &sa); err != nil && errors.IsNotFound(err) {
		log.Log.Info(fmt.Sprintf("Service account '%s/%s' has not been found, creating one..", ns, saName))
		sa.Name = saName
		sa.ObjectMeta.Annotations = map[string]string{internal.CreatedByAnnotationKey: internal.CreatedByAnnotationValue}
		sa.Namespace = ns
		if err := r.Client.Create(ctx, &sa); err != nil {
			log.Log.Error(err, "Unable to create the service account")
		}
	}
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
		log.Log.Error(err, fmt.Sprintf("Cannot get %v resource with name '%v' ", reflect.TypeOf(obj), nsName))
		return nil, ""
	}
	switch casted := obj.(type) {
	case *apps.Deployment:
		return casted.Spec.Selector.MatchLabels, casted.Spec.Template.Spec.ServiceAccountName
	case *apps.ReplicaSet:
		return casted.Spec.Selector.MatchLabels, casted.Spec.Template.Spec.ServiceAccountName
	case *apps.DaemonSet:
		return casted.Spec.Selector.MatchLabels, casted.Spec.Template.Spec.ServiceAccountName
	case *apps.StatefulSet:
		return casted.Spec.Selector.MatchLabels, casted.Spec.Template.Spec.ServiceAccountName
	case *core.Service:
		return casted.Spec.Selector, ""
	}

	return nil, ""
}

func getContainerName(pod core.Pod) string {
	for k, v := range pod.GetAnnotations() {
		if k == "kubectl.kubernetes.io/default-container" {
			return v
		}
	}
	// if the annotation is not specified, use the first declared container
	return pod.Spec.Containers[0].Name
}

// ClientSet returns the k8s client
func (r *RbacEventHandler) ClientSet() *kubernetes.Clientset {
	if r.clientset == nil {
		r.clientset, _ = SetupK8sClient()
	}
	return r.clientset
}

func (r *RbacEventHandler) emitEvent(resource kremserv1.RbacNegotiation, entry *RbacEntry) {
	// todo: consider using AnnotatedEventf
	r.Recorder.Eventf(&resource, "Normal", "RbacEntryCreated", "New RBAC entry: "+
		"role=%s, verb=%s, resource=%s, group=%s", resource.Spec.Role.Name, entry.Verb, entry.Object.Kind, entry.Object.Group)
}
