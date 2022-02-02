package controllers

import (
	"fmt"
	"regexp"
	"strings"
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

// RbacResource identifies the object of the RBAC triplet (example: apps/Deployment)
type RbacResource struct {
	Group string
	Kind  string
}

// RbacEntry holds the information about identified rbac entries that has been found in the logs
type RbacEntry struct {
	Verb     string
	Object   RbacResource
	ObjectNS string
}

// clients
const regexpTemplate1 = "User \\\\?\"system:serviceaccount:%s:%s\\\\?\" cannot (?P<Verb>\\S+) (resource )?\\\\?\"?(?P<Kind>[^\"\\s\\\\]+)\\\\?\"?" +
	" (in API group \\\\?\"(?P<ApiGroup>[^\"\\s\\\\]*)\\\\?\" )?(at the cluster scope|in the namespace \\\\?\"?(?P<Namespace>[^\"\\s\\\\]*)\\\\?\"?)"

// cache/reflector.go
const regexpTemplate2 = " Failed to (?P<Verb>\\S+) \\*[^:]+: (\\S+) \\(get (?P<Kind>[^)]+)\\)"

// FindRbacEntry returns the RbacEntry if it was found in the log for given subject and namespace or nil otherwise
func FindRbacEntry(log string, subjectNS string, subject string) *RbacEntry {
	re := fmt.Sprintf(regexpTemplate1, subjectNS, subject)
	r, err := regexp.Compile(re)
	if err != nil {
		return nil
	}
	match := r.FindStringSubmatch(log)
	if len(match) < 8 {
		return findRbacEntryFallback(log)
	}
	verb := match[r.SubexpIndex("Verb")]
	kind := match[r.SubexpIndex("Kind")]
	apiGr := match[r.SubexpIndex("ApiGroup")]
	ns := match[r.SubexpIndex("Namespace")]

	return &RbacEntry{
		Verb: verb,
		Object: RbacResource{
			Group: apiGr,
			Kind:  kind,
		},
		ObjectNS: ns,
	}
}

func findRbacEntryFallback(log string) *RbacEntry {
	r, err := regexp.Compile(regexpTemplate2)
	if err != nil {
		return nil
	}
	match := r.FindStringSubmatch(log)
	if len(match) < 3 {
		return nil
	}
	verb := match[r.SubexpIndex("Verb")]
	kind := match[r.SubexpIndex("Kind")]
	apiGr := ""
	if strings.Contains(kind, ".") {
		chunks := strings.SplitN(kind, ".", 2)
		kind = chunks[0]
		apiGr = chunks[1]
	}

	return &RbacEntry{
		Verb: verb,
		Object: RbacResource{
			Group: apiGr,
			Kind:  kind,
		},
		ObjectNS: "",
	}
}
