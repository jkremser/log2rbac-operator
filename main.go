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

package main

import (
	"context"
	"flag"
	"os"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	kremserv1 "github.com/jkremser/log2rbac-operator/api/v1"
	"github.com/jkremser/log2rbac-operator/controllers"
	"github.com/jkremser/log2rbac-operator/internal"
	"github.com/sethvargo/go-envconfig"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
	version  = "dev"
	gitSha   = "unknown"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(kremserv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	ctx := context.Background()
	var cfg internal.Config
	err := envconfig.Process(ctx, &cfg)
	internal.SetupLog(cfg.Log)
	cfg.App = &internal.AppConfig{
		Version: version,
		GitSha:  gitSha,
	}
	if err != nil {
		setupLog.Error(err, "unable to load config")
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "924cc958.dev",
	})
	if err != nil {
		setupLog.Error(err, "unable to start log2rbac manager")
		os.Exit(1)
	}

	if err = (&controllers.RbacNegotiationReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("log2rbac"),
		Config:   &cfg,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RbacNegotiation")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	internal.PrintBanner(cfg.Log)
	setupLog.Info("is starting..")
	internal.PrintInfo(setupLog, version)
	setupLog.Info("gitsha", "sha", gitSha)

	// tracing
	cleanup := internal.SetupTracing(cfg, ctx, setupLog)
	defer cleanup()

	// simple http server listening on /
	if err := internal.ServeRoot(mgr, *cfg.App); err != nil {
		setupLog.Error(err, "unable to set up http handler for root /")
		os.Exit(1)
	}

	// todo: check here if the CRD is there and if not, create it

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running log2rbac")
		os.Exit(1)
	}
}
