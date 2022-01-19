package controllers

import (
	"context"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func SetupK8sClient() *kubernetes.Clientset {
	_ = log.FromContext(context.Background())
	var config *rest.Config
	var err error
	_, inCluster := os.LookupEnv("KUBERNETES_SERVICE_HOST")
	if !inCluster {
		var kubeconfig string
		if home := homedir.HomeDir(); home != "" {
			log.Log.Info("Using kubeconfig from:" + filepath.Join(home, ".kube", "config"))
			kubeconfig = filepath.Join(home, ".kube", "config")
		}

		// use the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			panic(err)
		}
	} else {
		// creates the in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err)
		}
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	return clientset
}
