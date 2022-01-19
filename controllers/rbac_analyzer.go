package controllers

import (
	"fmt"
	"regexp"
)

//examples:

//go-client:
//forbidden: User "system:serviceaccount:log2rbac-operator-system:log2rbac-operator-controller-manager" cannot list resource "nodes" in API group "" at the cluster scope
//forbidden: User "system:serviceaccount:log2rbac-operator-system:log2rbac-operator-controller-manager" cannot list resource "configmaps" in API group "" at the cluster scope
//forbidden: User "system:serviceaccount:log2rbac-operator-system:log2rbac-operator-controller-manager" cannot list resource "clusterrolebindings" in API group "rbac.authorization.k8s.io" at the cluster scope

// javascript
//forbidden: User "system:serviceaccount:default:default" cannot list resource "jobs" in API group "batch" in the namespace "default"

// java
//fabric8
//forbidden: User "system:serviceaccount:staging:default" cannot get pods in the namespace "staging"
//forbidden: User "system:serviceaccount:default:my-svc-account" cannot get deployments.extensions in the namespace "default"
//forbidden: User "system:serviceaccount:badefault" cannot list resource "namespaces" in API group "" at the cluster scope

// python

// rust

type RbacResource struct {
	Group string
	Kind  string
}

type RbacEntry struct {
	Verb     string
	Object   RbacResource
	ObjectNS string
}

const regexpTemplate = "User \"system:serviceaccount:%s:%s\" cannot (?P<Verb>\\S+) (resource )?\"?(?P<Kind>[^\"\\s]+)\"?" +
	" (in API group \"(?P<ApiGroup>[^\"\\s]*)\" )?(at the cluster scope|in the namespace \"?(?P<Namespace>[^\"\\s]*)\"?)"

func FindRbacEntry(log string, subjectNS string, subject string) RbacEntry {
	re := fmt.Sprintf(regexpTemplate, subjectNS, subject)
	r, err := regexp.Compile(re)
	if err != nil {
		return RbacEntry{}
	}
	match := r.FindStringSubmatch(log)
	if len(match) < 8 {
		return RbacEntry{}
	}
	verb := match[r.SubexpIndex("Verb")]
	kind := match[r.SubexpIndex("Kind")]
	apiGr := match[r.SubexpIndex("ApiGroup")]
	ns := match[r.SubexpIndex("Namespace")]

	return RbacEntry{
		Verb: verb,
		Object: RbacResource{
			Group: apiGr,
			Kind:  kind,
		},
		ObjectNS: ns,
	}
}
