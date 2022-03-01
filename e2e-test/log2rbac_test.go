package test

import (
	"context"
	. "github.com/franela/goblin"
	"github.com/gruntwork-io/terratest/modules/shell"
	kremserv1 "github.com/jkremser/log2rbac-operator/api/v1"
	operator "github.com/jkremser/log2rbac-operator/controllers"
	crd "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"time"

	"testing"
)

const (
	// operator specific names
	operatorNs         = "log2rbac"
	operatorDeployment = operatorNs
	crdName            = "rbacnegotiations.kremser.dev"
	crdKindName        = "RbacNegotiation"
	svcName            = operatorNs + "-metrics-service"
	saName             = operatorNs
	roleName           = operatorNs + "-role"
	roleBindingName    = operatorNs + "-rolebinding"

	// test app specific names (k8gb)
	appK8gbNs              = "k8gb"
	appK8gbRoleName        = "new-" + appK8gbNs + "-role"
	appK8gbRoleName2       = "custom-" + appK8gbNs + "-role"
	appK8gbRoleBIndingName = appK8gbRoleName + "-binding"
	appK8gbDeploymentName  = appK8gbNs
	saAppK8gbName          = appK8gbNs
	appK8gbRnName1         = "for-" + appK8gbNs + "-using-deployment"
	appK8gbRnName2         = "for-" + appK8gbNs + "-using-selector"
	appK8gbRnNs            = appK8gbNs

	// test app specific names (prometheus)
	appPromNs              = "monitoring"
	appPromRoleName        = "custom-" + appPromNs + "-role"
)

func TestDeployment(t *testing.T) {

	g := Goblin(t)
	k8sCl, crdCl, _ := getAndTestClients(g)
	g.Describe("After log2rbac deployment", func() {
		// deployment
		g.It("k8s should contain the deployment with 1 replica in ready state", func() {
			dep, err := k8sCl.AppsV1().Deployments(operatorNs).Get(context.Background(), operatorDeployment, metav1.GetOptions{})
			callWasOk(g, err, dep)
			g.Assert(dep.Status.ReadyReplicas).Equal(int32(1))
		})

		// crd
		g.It("k8s should contain the CRD definition", func() {
			c, err := crdCl.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), crdName, metav1.GetOptions{})
			callWasOk(g, err, c)
			g.Assert(c.Spec.Names.Kind).Equal(crdKindName)
		})

		// svc
		g.It("k8s should contain the service for metrics", func() {
			svc, err := k8sCl.CoreV1().Services(operatorNs).Get(context.Background(), svcName, metav1.GetOptions{})
			callWasOk(g, err, svc)
			g.Assert(svc.Spec.ClusterIP).IsNotNil()
		})

		// rbac
		g.Describe("k8s should contain following RBAC resources:", func() {
			g.It("service account", func() {
				sa, err := k8sCl.CoreV1().ServiceAccounts(operatorNs).Get(context.Background(), saName, metav1.GetOptions{})
				callWasOk(g, err, sa)
			})
			g.It("cluster role", func() {
				r, err := k8sCl.RbacV1().ClusterRoles().Get(context.Background(), roleName, metav1.GetOptions{})
				callWasOk(g, err, r)
			})
			g.It("cluster role binding", func() {
				rb, err := k8sCl.RbacV1().ClusterRoleBindings().Get(context.Background(), roleBindingName, metav1.GetOptions{})
				callWasOk(g, err, rb)
			})
		})
	})
}

func assertK8gbNotThere(g *G, k8sCl *kubernetes.Clientset) {
	g.Describe("In vanilla deployment", func() {
		g.It("there is no cluster role called "+appK8gbRoleName, func() {
			_, err := k8sCl.RbacV1().ClusterRoles().Get(context.Background(), appK8gbRoleName, metav1.GetOptions{})
			wasNotFound(g, err)
		})
		g.It("there is no sample app", func() {
			_, err := k8sCl.AppsV1().Deployments(appK8gbNs).Get(context.Background(), appK8gbDeploymentName, metav1.GetOptions{})
			wasNotFound(g, err)
		})
	})
}

func TestReconciliationForDeployment(t *testing.T) {
	// assure it's empty
	makeClean(t, appK8gbNs)

	g := Goblin(t)
	k8sCl, _, crdRest := getAndTestClients(g)

	// pre-requisites: it's empty
	assertK8gbNotThere(g, k8sCl)

	// deploy test application that fails to start because of insufficient rights
	deploySampleApp1(t)

	g.Describe("When sample app got deployed", func() {
		g.It("the deployment is present", func() {
			appDep, err := k8sCl.AppsV1().Deployments(appK8gbNs).Get(context.Background(), appK8gbDeploymentName, metav1.GetOptions{})
			callWasOk(g, err, appDep)
			g.Assert(appDep.Status.ReadyReplicas).Equal(int32(0), "No replica should be available because it's failing on rbac")
		})
		g.It("there is still no role called "+appK8gbRoleName, func() {
			_, err := k8sCl.RbacV1().ClusterRoles().Get(context.Background(), appK8gbRoleName, metav1.GetOptions{})
			wasNotFound(g, err)
		})
		g.It("there is no rbacnegotiation CR", func() {
			rns, err := getRNs(crdRest)
			callWasOk(g, err, rns)
			g.Assert(len(rns)).IsZero()
		})
	})

	// create the RBACNegotiation custom resource that will trigger the operator
	createCr(t)

	// verify that operator is doing its job
	g.Describe("After rbacnegotiation was requested", func() {
		g.It("the CR was created", func() {
			rns, err := getRNs(crdRest)
			callWasOk(g, err, rns)
			g.Assert(rns).IsNotZero()
			g.Assert(rns[0].Name).Equal(appK8gbRnName1)
			g.Assert(rns[0].Namespace).Equal(appK8gbRnNs)
		})
		g.It("there is a new event", func() {
			g.Timeout(130 * time.Second)
			var checkEvent func(attempts int32)
			checkEvent = func(attempts int32) {
				// wait a bit
				time.Sleep(10 * time.Second)

				evList, er := k8sCl.EventsV1().Events(appK8gbRnNs).List(context.Background(), metav1.ListOptions{})
				callWasOk(g, er, evList)
				found := false
				for _, e := range evList.Items {
					if e.Reason == "RbacEntryCreated" {
						found = true
					}
				}
				if found {
					return
				}
				if attempts == 0 {
					g.Failf("New event with reason = 'RbacEntryCreated' was not found, events: %+v", evList.Items)
				}
				checkEvent(attempts - 1)
			}
			checkEvent(12)
		})
		g.It("the cluster role got created", func() {
			r, err := k8sCl.RbacV1().ClusterRoles().Get(context.Background(), appK8gbRoleName, metav1.GetOptions{})
			callWasOk(g, err, r)
		})
		g.It("the cluster role is bound to the associated service account", func() {
			rb, err := k8sCl.RbacV1().ClusterRoleBindings().Get(context.Background(), appK8gbRoleBIndingName, metav1.GetOptions{})
			callWasOk(g, err, rb)
			g.Assert(rb.Subjects).IsNotZero()
			g.Assert(rb.Subjects[0]).IsNotZero()
			g.Assert(rb.Subjects[0].Name).Equal(saAppK8gbName)
		})
		g.It("after some time, new rights are populated on the role", func() {
			// wait a bit
			g.Timeout(10 * time.Second)
			time.Sleep(5 * time.Second)

			r, err := k8sCl.RbacV1().ClusterRoles().Get(context.Background(), appK8gbRoleName, metav1.GetOptions{})
			callWasOk(g, err, r)
			g.Assert(len(r.Rules)).IsNotZero()
		})
	})
	makeClean(t, appK8gbNs)
}

func TestReconciliationForCustomSelector(t *testing.T) {
	g := Goblin(t)
	k8sCl, _, crdRest := getAndTestClients(g)

	// pre-requisites: it's empty
	assertK8gbNotThere(g, k8sCl)

	// deploy test app again
	deploySampleApp1(t)

	// apply the CR
	applyYaml(t, "./yaml/k8gb-selector-rn.yaml")

	g.Describe("After rbacnegotiation was requested (using selector)", func() {
		var rulesNumber int
		var role *rbac.Role
		g.It("there is role called "+appK8gbRoleName, func() {
			var err error
			role, err = k8sCl.RbacV1().Roles(appK8gbNs).Get(context.Background(), appK8gbRoleName2, metav1.GetOptions{})
			callWasOk(g, err, role)
		})
		g.It("but the role is empty", func() {
			rulesNumber = len(role.Rules)
			g.Assert(rulesNumber <= 1).IsTrue() // or there is just one item if the operator was fast enough
		})
		g.It("the CR was created", func() {
			rns, err := getRNs(crdRest)
			callWasOk(g, err, rns)
			g.Assert(rns).IsNotZero()
			g.Assert(rns[0].Name).Equal(appK8gbRnName2)
			g.Assert(rns[0].Namespace).Equal(appK8gbRnNs)
		})
		g.It("after some time, new rights are populated on the role", func() {
			// wait a bit
			g.Timeout(130 * time.Second)
			var checkRole func(attempts int32)
			checkRole = func(attempts int32) {
				// wait a bit
				time.Sleep(10 * time.Second)

				r, err := k8sCl.RbacV1().Roles(appK8gbNs).Get(context.Background(), appK8gbRoleName2, metav1.GetOptions{})
				callWasOk(g, err, r)
				newRightsFound := rulesNumber < len(r.Rules)
				if newRightsFound {
					return // ok
				}
				if attempts == 0 {
					g.Failf("No new rules were populated on role %s. Rules: %+v", appK8gbRoleName2, r.Rules)
				}
				checkRole(attempts - 1)
			}
			checkRole(12)
		})
	})

	//makeClean(t, appK8gbNs)
}

//func TestReconciliationForPrometheusService(t *testing.T) {
//	// assure it's empty
//	makeClean(t, appPromNs)

//	applyYaml(t, "./yaml/prom-svc-rn.yaml")
//	deploySampleApp2(t)
//}
//
//func TestReconciliationForPrometheusDeployment(t *testing.T) {
//	makeClean(t, appPromNs)
//	applyYaml(t, "./yaml/prom-svc-rn.yaml")
//}


func getAndTestClients(g *G) (*kubernetes.Clientset, *crd.Clientset, *rest.RESTClient) {
	var k8sCl *kubernetes.Clientset
	var crdCl *crd.Clientset
	var crdRest *rest.RESTClient
	g.Describe("Connection to k8s works", func() {
		c1, cfg := operator.SetupK8sClient()
		c2, e := crd.NewForConfig(cfg)
		callWasOk(g, e, c1, c2)
		cfg.ContentConfig.GroupVersion = &kremserv1.GroupVersion
		cfg.APIPath = "/apis"
		cfg.NegotiatedSerializer = serializer.NewCodecFactory(scheme.Scheme)
		cfg.UserAgent = rest.DefaultKubernetesUserAgent()
		kremserv1.AddToScheme(scheme.Scheme)
		c3, er := rest.UnversionedRESTClientFor(cfg)
		callWasOk(g, er, c3)
		k8sCl = c1
		crdCl = c2
		crdRest = c3
	})
	return k8sCl, crdCl, crdRest
}

func kubectl(t *testing.T, args []string) {
	cmd := shell.Command{
		Command: "kubectl",
		Args:    args,
	}
	shell.RunCommand(t, cmd)
}

func applyYaml(t *testing.T, path string) {
	kubectl(t, []string{"apply", "-f", path})
}

func deploySampleApp1(t *testing.T) {
	applyYaml(t, "./yaml/k8gb.yaml")
}

func deploySampleApp2(t *testing.T) {
	applyYaml(t, "https://github.com/prometheus-operator/kube-prometheus/raw/v0.10.0/manifests/prometheusOperator-deployment.yaml")
}

func createCr(t *testing.T /*, crdCl *crd.Clientset*/) {
	applyYaml(t, "./yaml/k8gb-deploy-rn.yaml")
}

func isNotErr(g *G, err error) {
	if err != nil {
		g.Fail(err)
	}
}

func callWasOk(g *G, err error, obj ...interface{}) {
	isNotErr(g, err)
	for _, o := range obj {
		g.Assert(o).IsNotNil()
	}
}

func wasNotFound(g *G, err error) {
	if !errors.IsNotFound(err) {
		g.Failf("expected not found error, but got: %v", err)
	}
}

func getRNs(c *rest.RESTClient) ([]kremserv1.RbacNegotiation, error) {
	result := kremserv1.RbacNegotiationList{}
	e := c.Get().Resource("rbacnegotiations").Do(context.Background()).Into(&result)
	if e != nil {
		return nil, e
	}
	return result.Items, nil
}

func makeClean(t *testing.T, ns string) {
	// delete all RNs
	kubectl(t, []string{"delete", "rbacnegotiations", "--all", "-A"})

	// delete namespace
	kubectl(t, []string{"delete", "ns", ns, "--ignore-not-found"})

	// delete cluster roles
	kubectl(t, []string{"delete", "clusterrole", appK8gbRoleName, "--ignore-not-found"})
	kubectl(t, []string{"delete", "clusterrole", appPromRoleName, "--ignore-not-found"})
	time.Sleep(15 * time.Second)
}
