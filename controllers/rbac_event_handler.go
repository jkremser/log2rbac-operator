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
	"strings"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kremserv1 "jkremser/log2rbac-operator/api/v1"
)

// RbacEventHandler handles the CRUD event from CR
type RbacEventHandler struct {
	client.Client
	clientset *kubernetes.Clientset
}

func (r *RbacEventHandler) handleResource(ctx context.Context, resource kremserv1.ForSpec) {
	logs, sa, err := r.logsFromFirstPod(ctx, resource)
	if err != nil {
		log.Log.Error(err, "Unable to get logs from underlying pod")
	}
	//if logs.contain that string edit the role to contain that exception and restart the pod
	missingRbacEntry := FindRbacEntry(logs, resource.Namespace, sa)
	r.addMissingRbacEntry(sa, missingRbacEntry)

	// make this optional
	//restartPods()
	log.Log.Info(fmt.Sprintf("%#v", missingRbacEntry))
}

func (r *RbacEventHandler) addMissingRbacEntry(sa string, entry RbacEntry) {

}

func (r *RbacEventHandler) logsFromFirstPod(ctx context.Context, resource kremserv1.ForSpec) (string, string, error) {
	var selector map[string]string
	var sa string
	switch strings.ToLower(resource.Kind) {
	case "deploy", "deployment", "deployments":
		deploymentNsName := client.ObjectKey{
			Namespace: resource.Namespace,
			Name:      resource.Name,
		}
		dep := apps.Deployment{}
		if err := r.Client.Get(ctx, deploymentNsName, &dep); err != nil {
			return "", "", err
		}
		selector = dep.Spec.Selector.MatchLabels
		sa = dep.Spec.Template.Spec.ServiceAccountName
	default:
		return "", "", fmt.Errorf("unrecognized kind: '%s'", resource.Kind)
	}

	var podList core.PodList
	if err := r.Client.List(ctx, &podList, client.InNamespace(resource.Namespace), client.MatchingLabels(selector)); err != nil {
		return "", "", err
	}
	pods := podList.Items
	if len(pods) == 0 {
		return "", "", fmt.Errorf("no pods found for %s called '%s'", resource.Kind, resource.Name)
	}
	podName := pods[0].GetName()
	req := r.ClientSet().CoreV1().Pods(resource.Namespace).GetLogs(podName, &core.PodLogOptions{})
	podLogs, err := req.Stream(ctx)
	if err != nil {
		return "", "", err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "", "", err
	}
	str := buf.String()

	return str, sa, nil
}

func (r *RbacEventHandler) ClientSet() *kubernetes.Clientset {
	if r.clientset == nil {
		r.clientset = SetupK8sClient()
	}
	return r.clientset
}
