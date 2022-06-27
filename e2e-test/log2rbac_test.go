package test

import (
	"context"
	"fmt"
	. "github.com/franela/goblin"
	"github.com/gruntwork-io/terratest/modules/shell"
	kremserv1 "github.com/jkremser/log2rbac-operator/api/v1"
	operator "github.com/jkremser/log2rbac-operator/controllers"
	rbac "k8s.io/api/rbac/v1"
	crd "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"time"
	"crypto/rand"

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
	appK8gbRoleBindingName = appK8gbRoleName + "-binding"
	appK8gbDeploymentName  = appK8gbNs
	saAppK8gbName          = appK8gbNs
	appK8gbRnName1         = "for-" + appK8gbNs + "-using-deployment"
	appK8gbRnName2         = "for-" + appK8gbNs + "-using-selector"
	appK8gbRnNs            = appK8gbNs

	// test app specific names (prometheus)
	appPromNs        = "monitoring"
	appPromRoleName1 = "custom-" + appPromNs + "-role1"
	appPromRoleName2 = "custom-" + appPromNs + "-role2"
	appPromRnName1   = "for-prom-svc"
	//appPromRnName2         = "for-prom-deploy"
)

type TestContext struct {
	t       *testing.T
	g       *G
	k8sCl   *kubernetes.Clientset
	crdCl   *crd.Clientset
	crdRest *rest.RESTClient
}

// entry point
func TestAll(t *testing.T) {
	g := Goblin(t)
	k8sCl, crdCl, crdRest := getAndTestClients(g)
	c := TestContext{t: t, g: g, k8sCl: k8sCl, crdCl: crdCl, crdRest: crdRest}

	// tests
	g.Describe("log2rbac:", func() {
		GobTestDeployment(&c)
		GobTestReconciliationForDeployment(&c) // k8gb
		GobTestReconciliationForCustomSelector(&c) // k8gb
		GobTestReconciliationForPrometheusService(&c) // prometheus-operator
		GobTestReconciliationForPrometheusDeployment(&c) // prometheus-operator
	})
}

func GobTestDeployment(c *TestContext) {
	g := c.g
	g.Describe("After log2rbac deployment", func() {
		// deployment
		g.It("k8s should contain the deployment with 1 replica in ready state", func() {
			dep, err := c.k8sCl.AppsV1().Deployments(operatorNs).Get(context.Background(), operatorDeployment, metav1.GetOptions{})
			callWasOk(g, err, dep)
			g.Assert(dep.Status.ReadyReplicas).Equal(int32(1))
		})

		// crd
		g.It("k8s should contain the CRD definition", func() {
			crd, err := c.crdCl.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), crdName, metav1.GetOptions{})
			callWasOk(g, err, crd)
			g.Assert(crd.Spec.Names.Kind).Equal(crdKindName)
		})

		// svc
		g.It("k8s should contain the service for metrics", func() {
			svc, err := c.k8sCl.CoreV1().Services(operatorNs).Get(context.Background(), svcName, metav1.GetOptions{})
			callWasOk(g, err, svc)
			g.Assert(svc.Spec.ClusterIP).IsNotNil()
		})

		// rbac
		g.Describe("k8s should contain following RBAC resources:", func() {
			g.It("service account", func() {
				sa, err := c.k8sCl.CoreV1().ServiceAccounts(operatorNs).Get(context.Background(), saName, metav1.GetOptions{})
				callWasOk(g, err, sa)
			})
			g.It("cluster role", func() {
				r, err := c.k8sCl.RbacV1().ClusterRoles().Get(context.Background(), roleName, metav1.GetOptions{})
				callWasOk(g, err, r)
			})
			g.It("cluster role binding", func() {
				rb, err := c.k8sCl.RbacV1().ClusterRoleBindings().Get(context.Background(), roleBindingName, metav1.GetOptions{})
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
		//g.It("there is no sample app", func() {
		//	_, err := k8sCl.AppsV1().Deployments(ns).Get(context.Background(), appK8gbDeploymentName, metav1.GetOptions{})
		//	wasNotFound(g, err)
		//})
	})
}

func GobTestReconciliationForDeployment(c *TestContext) {
	g := c.g
	var ns string

	// pre-requisites: it's empty
	assertK8gbNotThere(g, c.k8sCl)

	g.Describe("When sample app got deployed", func() {
		g.Before(func() {
			ns = createRandomNs(c.t)

			// deploy test application that fails to start because of insufficient rights
			deploySampleApp1(c.t, ns)
		})
		g.It("the deployment is present", func() {
			appDep, err := c.k8sCl.AppsV1().Deployments(ns).Get(context.Background(), appK8gbDeploymentName, metav1.GetOptions{})
			callWasOk(g, err, appDep)
			g.Assert(appDep.Status.ReadyReplicas).Equal(int32(0), "No replica should be available because it's failing on rbac")
		})
		g.It("there is still no role called "+appK8gbRoleName, func() {
			_, err := c.k8sCl.RbacV1().ClusterRoles().Get(context.Background(), appK8gbRoleName, metav1.GetOptions{})
			wasNotFound(g, err)
		})
		g.It("there is no rbacnegotiation CR", func() {
			rns, err := getRNs(c.crdRest, ns)
			callWasOk(g, err, rns)
			g.Assert(len(rns)).IsZero()
		})
	})

	// verify that operator is doing its job
	g.Describe("After rbacnegotiation was requested", func() {
		g.After(func() {
			makeClean(c.t, ns)
		})
		g.Before(func() {
			// create the RBACNegotiation custom resource that will trigger the operator
			createCr(c.t, ns)
		})
		g.It("the CR was created", func() {rns, err := getRNs(c.crdRest, ns)
			callWasOk(g, err, rns)
			g.Assert(len(rns)).IsNotZero()
			g.Assert(rns[0].Name).Equal(appK8gbRnName1)
			g.Assert(rns[0].Namespace).Equal(ns)
		})
		g.It("there is a new event", func() {
			g.Timeout(130 * time.Second)
			var checkEvent func(attempts int32)
			checkEvent = func(attempts int32) {
				// wait a bit
				time.Sleep(10 * time.Second)

				evList, er := c.k8sCl.EventsV1().Events(ns).List(context.Background(), metav1.ListOptions{})
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
			r, err := c.k8sCl.RbacV1().ClusterRoles().Get(context.Background(), appK8gbRoleName, metav1.GetOptions{})
			callWasOk(g, err, r)
		})
		g.It("the cluster role is bound to the associated service account", func() {
			rb, err := c.k8sCl.RbacV1().ClusterRoleBindings().Get(context.Background(), appK8gbRoleBindingName, metav1.GetOptions{})
			callWasOk(g, err, rb)
			g.Assert(len(rb.Subjects)).IsNotZero()
			g.Assert(rb.Subjects[0]).IsNotZero()
			g.Assert(rb.Subjects[0].Name).Equal(saAppK8gbName)
		})
		g.It("after some time, new rights are populated on the role", func() {
			// wait a bit
			g.Timeout(10 * time.Second)
			time.Sleep(5 * time.Second)

			r, err := c.k8sCl.RbacV1().ClusterRoles().Get(context.Background(), appK8gbRoleName, metav1.GetOptions{})
			callWasOk(g, err, r)
			g.Assert(len(r.Rules)).IsNotZero()
		})
	})
}

func GobTestReconciliationForCustomSelector(c *TestContext) {
	g := c.g
	var ns string
	// pre-requisites: it's empty
	assertK8gbNotThere(g, c.k8sCl)
	g.Describe("After rbacnegotiation was requested (using selector)", func() {
		g.After(func() {
			makeClean(c.t, ns)
		})
		g.Before(func() {
			ns = createRandomNs(c.t)
			// deploy test app again
			deploySampleApp1(c.t, ns)
			// apply the CR
			applyYaml(c.t, "./yaml/k8gb-selector-rn.yaml", ns)
		})

		var rulesNumber int
		var role *rbac.Role
		g.It("there is role called "+appK8gbRoleName, func() {
			var err error
			role, err = c.k8sCl.RbacV1().Roles(ns).Get(context.Background(), appK8gbRoleName2, metav1.GetOptions{})
			callWasOk(g, err, role)
		})
		g.It("but the role is empty", func() {
			rulesNumber = len(role.Rules)
			g.Assert(rulesNumber <= 1).IsTrue() // or there is just one item if the operator was fast enough
		})
		g.It("the CR was created", func() {
			rns, err := getRNs(c.crdRest, ns)
			callWasOk(g, err, rns)
			g.Assert(rns).IsNotZero()
			g.Assert(rns[0].Name).Equal(appK8gbRnName2)
			g.Assert(rns[0].Namespace).Equal(ns)
		})
		g.It("after some time, new rights are populated on the role", func() {
			// wait a bit
			g.Timeout(130 * time.Second)
			var checkRole func(attempts int32)
			checkRole = func(attempts int32) {
				// wait a bit
				time.Sleep(10 * time.Second)

				r, err := c.k8sCl.RbacV1().Roles(ns).Get(context.Background(), appK8gbRoleName2, metav1.GetOptions{})
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
}

func GobTestReconciliationForPrometheusService(c *TestContext) {
	g := c.g
	var ns string
	g.Describe("For prometheus-operator (using svc)", func() {
		g.It("there is no cluster role called "+appPromRoleName1, func() {
			_, err := c.k8sCl.RbacV1().ClusterRoles().Get(context.Background(), appPromRoleName1, metav1.GetOptions{})
			wasNotFound(g, err)
		})
	})

	g.Describe("After was deployed prometheus-operator and RN requested (using svc)", func() {
		g.After(func() {
			makeClean(c.t, ns)
		})
		g.Before(func() {
			ns = createRandomNs(c.t)
			applyYaml(c.t, "./yaml/prom-svc-rn.yaml", ns)
			deploySampleApp2(c.t, ns)
		})

		var rulesNumber int
		var newRole *rbac.ClusterRole
		g.It("the CR was created", func() {
			rns, err := getRNs(c.crdRest, ns)
			callWasOk(g, err, rns)
			g.Assert(len(rns)).IsNotZero()
			g.Assert(rns[0].Name).Equal(appPromRnName1)
			g.Assert(rns[0].Namespace).Equal(ns)
		})
		g.It("there is eventually cluster role called "+appPromRoleName1, func() {
			g.Timeout(65 * time.Second)
			var checkRole func(attempts int32)
			checkRole = func(attempts int32) {
				// wait a bit
				time.Sleep(10 * time.Second)
				role, err := c.k8sCl.RbacV1().ClusterRoles().Get(context.Background(), appPromRoleName1, metav1.GetOptions{})
				if errors.IsNotFound(err) {
					if attempts == 0 {
						g.Failf("ClusterRole %s was not created by the operator.", appPromRoleName1)
					}
					checkRole(attempts - 1)
				}
				g.Assert(role).IsNotNil()
				newRole = role
				return
			}
			checkRole(5)
		})
		g.It("but the cluster role is empty", func() {
			rulesNumber = len(newRole.Rules)
			g.Assert(rulesNumber <= 1).IsTrue() // or there is just one item if the operator was fast enough
		})
		g.It("after some time, new rights are populated on the cluster role", func() {
			// wait a bit
			g.Timeout(130 * time.Second)
			var checkRole func(attempts int32)
			checkRole = func(attempts int32) {
				// wait a bit
				time.Sleep(10 * time.Second)

				r, err := c.k8sCl.RbacV1().ClusterRoles().Get(context.Background(), appPromRoleName1, metav1.GetOptions{})
				callWasOk(g, err, r)
				newRightsFound := rulesNumber < len(r.Rules)
				if newRightsFound {
					return // ok
				}
				if attempts == 0 {
					g.Failf("No new rules were populated on cluster role %s. Rules: %+v", appPromRoleName1, r.Rules)
				}
				checkRole(attempts - 1)
			}
			checkRole(12)
		})
	})
}

func GobTestReconciliationForPrometheusDeployment(c *TestContext) {
	g := c.g
	//makeClean(t, appPromNs)
	//applyYaml(t, "./yaml/prom-deploy-rn.yaml")
	g.It("Test also prometheus operator using rbac negotiation for deployment")
}

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

func applyYaml(t *testing.T, path string, ns string) {
	kubectl(t, []string{"apply", "-n", ns, "-f", path})
}

func createRandomNs(t *testing.T) string {
	b := make([]byte, 2)
	_, err := rand.Read(b)
	if err != nil {
		fmt.Println("Error: ", err)
		return ""
	}
	ns := fmt.Sprintf("test-%x", b[0:2])
	kubectl(t, []string{"create", "ns", ns})
	return ns
}

func deploySampleApp1(t *testing.T, ns string) {
	applyYaml(t, "./yaml/k8gb.yaml", ns)
}

func deploySampleApp2(t *testing.T, ns string) {
	//applyYaml(t, "https://github.com/prometheus-operator/kube-prometheus/raw/v0.10.0/manifests/prometheusOperator-deployment.yaml", ns)
	applyYaml(t, "./yaml/prometheusOperator-deployment.yaml", ns)
}

func createCr(t *testing.T, ns string) {
	applyYaml(t, "./yaml/k8gb-deploy-rn.yaml", ns)
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

func getRNs(c *rest.RESTClient, ns string) ([]kremserv1.RbacNegotiation, error) {
	result := kremserv1.RbacNegotiationList{}
	e := c.Get().Resource("rbacnegotiations").Namespace(ns).Do(context.Background()).Into(&result)
	if e != nil {
		return nil, e
	}
	return result.Items, nil
}

func makeClean(t *testing.T, ns string) {
	// delete all RNs
	kubectl(t, []string{"delete", "rbacnegotiations", "--all", "-n", ns})

	// delete namespace
	kubectl(t, []string{"delete", "ns", ns, "--ignore-not-found"})

	// delete cluster roles
	kubectl(t, []string{"delete", "clusterroles", appK8gbRoleName, appPromRoleName1, appPromRoleName2, "--ignore-not-found"})
	time.Sleep(20 * time.Second)
}
