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
// volumeclaims is forbidden: User "system:serviceaccount:development:spark-operator-development-spark" cannot list resource "persistentvolumeclaims" in API group "" in the namespace "development".

// python
//message":"pods is forbidden: User "system:serviceaccount:citicai:default" cannot list pods at the cluster scope","reason":"Forbidden","
//forbidden: User \"system:serviceaccount:default:default\" cannot get resource \"v1\" in API group \"\" at the cluster scope","reason":"Forbidden"
// HTTP response body: {"kind":"Status","apiVersion":"v1","metadata":{},"status":"Failure","message":"pods \"pony-job-gmji0i3o-gmji0i3owc\" is forbidden: User \"system:serviceaccount:spark-dev:spark\" cannot get pods/status in the namespace \"spark-dev\"",

// rust

// other
//User "system:serviceaccount:mycomp-services-process:default" cannot get services in the namespace "mycomp-services-process"

// cache-client
//k8gb-55985fb855-82zh4 k8gb E0120 15:14:03.625397       1 reflector.go:138] k8s.io/client-go@v0.22.2/tools/cache/reflector.go:167: Failed to watch *v1.Endpoints: unknown (get endpoints)
//k8gb-55985fb855-82zh4 k8gb E0120 15:14:15.718384       1 reflector.go:138] k8s.io/client-go@v0.22.2/tools/cache/reflector.go:167: Failed to watch *v1beta1.Gslb: unknown (get gslbs.k8gb.absa.oss)
//k8gb-55985fb855-82zh4 k8gb E0120 15:14:17.352672       1 reflector.go:138] k8s.io/client-go@v0.22.2/tools/cache/reflector.go:167: Failed to watch *endpoint.DNSEndpoint: unknown (get dnsendpoints.externaldns.k8s.io)
//k8gb-55985fb855-82zh4 k8gb E0120 15:14:26.758227       1 reflector.go:138] k8s.io/client-go@v0.22.2/tools/cache/reflector.go:167: Failed to watch *v1beta1.Ingress: unknown (get ingresses.networking.k8s.io)


func TestGoClientLog1(t *testing.T) {
	log := "Yada yada \n yada forbidden: User \"system:serviceaccount:log2rbac-operator-system:log2rbac-operator-controller-manager\"" +
		" cannot list resource \"nodes\" in API group \"\" at the cluster scope. yada yada\n yada"
	want := &RbacEntry{
		Verb: "list",
		Object: RbacResource{
			Group: "",
			Kind:  "nodes",
		},
		ObjectNS: "",
	}
	got := FindRbacEntry(log, "log2rbac-operator-system", "log2rbac-operator-controller-manager")
	require.Equal(t, want, got)
}

func TestGoClientLog2(t *testing.T) {
	log := "forbidden: User \"system:serviceaccount:log2rbac-operator-system:log2rbac-operator-controller-manager\"" +
		" cannot list resource \"configmaps\" in API group \"\" at the cluster scope. yada yada\n yada"
	want := &RbacEntry{
		Verb: "list",
		Object: RbacResource{
			Group: "",
			Kind:  "configmaps",
		},
		ObjectNS: "",
	}
	got := FindRbacEntry(log, "log2rbac-operator-system", "log2rbac-operator-controller-manager")
	require.Equal(t, want, got)
}

func TestGoClientLog3(t *testing.T) {
	log := "Yada yada \n yada forbidden: User" +
		" \"system:serviceaccount:log2rbac-operator-system:log2rbac-operator-controller-manager\"" +
		" cannot list resource \"clusterrolebindings\" in API group \"rbac.authorization.k8s.io\" at the cluster scope\n yada"
	want := &RbacEntry{
		Verb: "list",
		Object: RbacResource{
			Group: "rbac.authorization.k8s.io",
			Kind:  "clusterrolebindings",
		},
		ObjectNS: "",
	}
	got := FindRbacEntry(log, "log2rbac-operator-system", "log2rbac-operator-controller-manager")
	require.Equal(t, want, got)
}

func TestJavascriptClientLog1(t *testing.T) {
	log := "Yada yada \n yada forbidden: User \"system:serviceaccount:foo:bar\" cannot list resource" +
		" \"jobs\" in API group \"batch\" in the namespace \"default\" yada\n yada"
	want := &RbacEntry{
		Verb: "list",
		Object: RbacResource{
			Group: "batch",
			Kind:  "jobs",
		},
		ObjectNS: "default",
	}
	got := FindRbacEntry(log, "foo", "bar")
	require.Equal(t, want, got)
}

func TestJavaClientLog1(t *testing.T) {
	log := "Yada yada \n yada forbidden: User forbidden: User \"system:serviceaccount:staging:default\" cannot get pods in the namespace \"staging\""
	want := &RbacEntry{
		Verb: "get",
		Object: RbacResource{
			Group: "",
			Kind:  "pods",
		},
		ObjectNS: "staging",
	}
	got := FindRbacEntry(log, "staging", "default")
	require.Equal(t, want, got)
}

func TestJavaClientLog2(t *testing.T) {
	log := "dsfforbidden: User \"system:serviceaccount:default:my-svc-account\" cannot get deployments.extensions in the namespace \"default\"ff"
	want := &RbacEntry{
		Verb: "get",
		Object: RbacResource{
			Group: "",
			Kind:  "deployments.extensions",
		},
		ObjectNS: "default",
	}
	got := FindRbacEntry(log, "default", "my-svc-account")
	require.Equal(t, want, got)
}

func TestJavaClientLog3(t *testing.T) {
	log := "dsfforbidden: User \"system:serviceaccount:ba:ba\" cannot list resource \"namespaces\" in API group \"\" at the cluster scopeff"
	want := &RbacEntry{
		Verb: "list",
		Object: RbacResource{
			Group: "",
			Kind:  "namespaces",
		},
		ObjectNS: "",
	}
	got := FindRbacEntry(log, "ba", "ba")
	require.Equal(t, want, got)
}

func TestJavaClientLog4(t *testing.T) {
	log := "volumeclaims is forbidden: User \"system:serviceaccount:development:spark-operator-development-spark\" cannot list resource \"persistentvolumeclaims\" in API group \"\" in the namespace \"development\"."
	want := &RbacEntry{
		Verb: "list",
		Object: RbacResource{
			Group: "",
			Kind:  "persistentvolumeclaims",
		},
		ObjectNS: "development",
	}
	got := FindRbacEntry(log, "development", "spark-operator-development-spark")
	require.Equal(t, want, got)
}

func TestJavaClientLogNegative1(t *testing.T) {
	log := "dsfforbidden: User \"system:serviceaccount:default:my-svc-account\" cannot get deployments.extensions in the namespace \"default\"sf"
	var want *RbacEntry = nil
	got := FindRbacEntry(log, "not-there", "non-existent")
	require.Equal(t, want, got)
}

func TestJavaClientLog5Negative2(t *testing.T) {
	log := "dsfforbidden: User \"system:serviceaccount:default:my-svc-account\" cannot yada get deployments.extensions in the namespace \"default\"sf"
	var want *RbacEntry = nil
	got := FindRbacEntry(log, "default", "my-svc-account")
	require.Equal(t, want, got)
}

func TestPythonClientLog1(t *testing.T) {
	log := "message\":\"pods is forbidden: User \"system:serviceaccount:citicai:default\" cannot list pods at the cluster scope\",\"reason\":\"Forbidden\",\""
	want := &RbacEntry{
		Verb: "list",
		Object: RbacResource{
			Group: "",
			Kind:  "pods",
		},
		ObjectNS: "",
	}
	got := FindRbacEntry(log, "citicai", "default")
	require.Equal(t, want, got)
}

func TestPythonClientLog2(t *testing.T) {
	log := "forbidden: User \\\"system:serviceaccount:default:default\\\" cannot get resource \\\"namespace\\\" in API group \\\"\\\" at the cluster scope\"," +
		"\"reason\":\"Forbidden\""
	want := &RbacEntry{
		Verb: "get",
		Object: RbacResource{
			Group: "",
			Kind:  "namespace",
		},
		ObjectNS: "",
	}
	got := FindRbacEntry(log, "default", "default")
	require.Equal(t, want, got)
}

func TestPythonClientLog3(t *testing.T) {
	log := "HTTP response body: {\"kind\":\"Status\",\"apiVersion\":\"v1\",\"metadata\":{},\"status\":\"Failure\"," +
		"\"message\":\"pods \\\"pony-job-gmji0i3o-gmji0i3owc\\\" is forbidden: User \\\"system:serviceaccount:foo:spark-dev:spark\\\"" +
		" cannot get pods/status in the namespace \\\"spark-dev\\\"\","
	want := &RbacEntry{
		Verb: "get",
		Object: RbacResource{
			Group: "",
			Kind:  "pods/status",
		},
		ObjectNS: "spark-dev",
	}
	got := FindRbacEntry(log, "foo", "spark-dev:spark")
	require.Equal(t, want, got)
}

func TestOtherLog1(t *testing.T) {
	log := "uuuu User \"system:serviceaccount:mycomp-services-process:default\" cannot get services in the namespace \"mycomp-services-process\""
	want := &RbacEntry{
		Verb: "get",
		Object: RbacResource{
			Group: "",
			Kind:  "services",
		},
		ObjectNS: "mycomp-services-process",
	}
	got := FindRbacEntry(log, "mycomp-services-process", "default")
	require.Equal(t, want, got)
}
