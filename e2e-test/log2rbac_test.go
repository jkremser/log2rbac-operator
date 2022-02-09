package test

import (
	"context"
	. "github.com/franela/goblin"
	operator "github.com/jkremser/log2rbac-operator/controllers"
	crd "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"testing"
)

const (
	operatorNs = "log2rbac"
	operatorDeployment = "log2rbac"
	crdName = "rbacnegotiations.kremser.dev"
	crdKindName = "RbacNegotiation"
	svcName = "log2rbac-metrics-service"
	saName = "log2rbac"
	roleName = "log2rbac-role"
	roleBindingName = "log2rbac-rolebinding"
)

func TestDeployment(t *testing.T) {
	g := Goblin(t)
	var k8sCl *kubernetes.Clientset
	var crdCl *crd.Clientset
	g.Describe("Connection to k8s works", func() {
		c1, cfg := operator.SetupK8sClient()
		c2, e := crd.NewForConfig(cfg)
		callWasOk(g, e, c1, c2)
		k8sCl = c1
		crdCl = c2
	})
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
		g.It("k8s should contain the service for metrics", func() {
			svc, err := k8sCl.CoreV1().Services(operatorNs).Get(context.Background(), svcName, metav1.GetOptions{})
			callWasOk(g, err, svc)
			g.Assert(svc.Spec.ClusterIP).IsNotNil()
		})
		g.Describe("k8s should contain the RBAC:", func() {
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

func TestReconciliation(t *testing.T) {
	// todo:
	// there is no role called foo
	// deploy sample app
	// create the CR for it
	// the role is there
	// after some time the role was populated with a verb and resource
}

func callWasOk(g *G, err error, obj... interface{}) {
	if err != nil {
		g.Fail(err)
	}
	for _, o := range obj {
		g.Assert(o).IsNotNil()
	}
}
