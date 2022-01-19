package controllers

import (
	"github.com/stretchr/testify/require"
	"testing"
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

// other
//User "system:serviceaccount:mycomp-services-process:default" cannot get services in the namespace "mycomp-services-process"

func TestGoClientLog1(t *testing.T) {
	log := "Yada yada \n yada forbidden: User \"system:serviceaccount:log2rbac-operator-system:log2rbac-operator-controller-manager\"" +
		" cannot list resource \"nodes\" in API group \"\" at the cluster scope. yada yada\n yada"
	want := RbacEntry{
		Verb: "list",
		Object: RbacResource{
			Group: "",
			Kind: "nodes",
		},
		ObjectNS: "",
	}
	got := FindRecord(log, "log2rbac-operator-system", "log2rbac-operator-controller-manager")
	require.Equal(t, want, got)
}

func TestGoClientLog2(t *testing.T) {
	log := "forbidden: User \"system:serviceaccount:log2rbac-operator-system:log2rbac-operator-controller-manager\"" +
		" cannot list resource \"configmaps\" in API group \"\" at the cluster scope. yada yada\n yada"
	want := RbacEntry{
		Verb: "list",
		Object: RbacResource{
			Group: "",
			Kind: "configmaps",
		},
		ObjectNS: "",
	}
	got := FindRecord(log, "log2rbac-operator-system", "log2rbac-operator-controller-manager")
	require.Equal(t, want, got)
}

func TestGoClientLog3(t *testing.T) {
	log := "Yada yada \n yada forbidden: User" +
		" \"system:serviceaccount:log2rbac-operator-system:log2rbac-operator-controller-manager\"" +
		" cannot list resource \"clusterrolebindings\" in API group \"rbac.authorization.k8s.io\" at the cluster scope\n yada"
	want := RbacEntry{
		Verb: "list",
		Object: RbacResource{
			Group: "rbac.authorization.k8s.io",
			Kind: "clusterrolebindings",
		},
		ObjectNS: "",
	}
	got := FindRecord(log, "log2rbac-operator-system", "log2rbac-operator-controller-manager")
	require.Equal(t, want, got)
}

func TestJavascriptClientLog1(t *testing.T) {
	log := "Yada yada \n yada forbidden: User \"system:serviceaccount:foo:bar\" cannot list resource" +
		" \"jobs\" in API group \"batch\" in the namespace \"default\" yada\n yada"
	want := RbacEntry{
		Verb: "list",
		Object: RbacResource{
			Group: "batch",
			Kind: "jobs",
		},
		ObjectNS: "default",
	}
	got := FindRecord(log,"foo", "bar")
	require.Equal(t, want, got)
}

func TestJavaClientLog1(t *testing.T) {
	log := "Yada yada \n yada forbidden: User forbidden: User \"system:serviceaccount:staging:default\" cannot get pods in the namespace \"staging\""
	want := RbacEntry{
		Verb: "get",
		Object: RbacResource{
			Group: "",
			Kind: "pods",
		},
		ObjectNS: "staging",
	}
	got := FindRecord(log,"staging", "default")
	require.Equal(t, want, got)
}

func TestJavaClientLog2(t *testing.T) {
	log := "dsfforbidden: User \"system:serviceaccount:default:my-svc-account\" cannot get deployments.extensions in the namespace \"default\"ff"
	want := RbacEntry{
		Verb: "get",
		Object: RbacResource{
			Group: "",
			Kind: "deployments.extensions",
		},
		ObjectNS: "default",
	}
	got := FindRecord(log,"default", "my-svc-account")
	require.Equal(t, want, got)
}

func TestJavaClientLog3(t *testing.T) {
	log := "dsfforbidden: User \"system:serviceaccount:ba:ba\" cannot list resource \"namespaces\" in API group \"\" at the cluster scopeff"
	want := RbacEntry{
		Verb: "list",
		Object: RbacResource{
			Group: "",
			Kind: "namespaces",
		},
		ObjectNS: "",
	}
	got := FindRecord(log,"ba", "ba")
	require.Equal(t, want, got)
}

func TestJavaClientLog4(t *testing.T) {
	log := "dsfforbidden: User \"system:serviceaccount:default:my-svc-account\" cannot get deployments.extensions in the namespace \"default\"sf"
	want := RbacEntry{}
	got := FindRecord(log,"not-there", "non-existent")
	require.Equal(t, want, got)
}

func TestJavaClientLog5(t *testing.T) {
	log := "dsfforbidden: User \"system:serviceaccount:default:my-svc-account\" cannot yada get deployments.extensions in the namespace \"default\"sf"
	want := RbacEntry{}
	got := FindRecord(log,"default", "my-svc-account")
	require.Equal(t, want, got)
}

func TestOtherLog1(t *testing.T) {
	log := "uuuu User \"system:serviceaccount:mycomp-services-process:default\" cannot get services in the namespace \"mycomp-services-process\""
	want := RbacEntry{
		Verb: "get",
		Object: RbacResource{
			Group: "",
			Kind: "services",
		},
		ObjectNS: "mycomp-services-process",
	}
	got := FindRecord(log,"mycomp-services-process", "default")
	require.Equal(t, want, got)
}
