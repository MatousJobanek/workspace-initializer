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
	"fmt"
	"os"

	"github.com/codeready-toolchain/workspace-initializer/controllers"
	tenancyv1alpha1 "github.com/kcp-dev/kcp/pkg/apis/tenancy/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/kcp"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	// +kubebuilder:scaffold:imports

	apisv1alpha1 "github.com/kcp-dev/kcp/pkg/apis/apis/v1alpha1"
)

var (
	scheme            = runtime.NewScheme()
	setupLog          = ctrl.Log.WithName("setup")
	kubeconfigContext string
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apisv1alpha1.AddToScheme(scheme))
	utilruntime.Must(tenancyv1alpha1.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme
	flag.StringVar(&kubeconfigContext, "context", "", "kubeconfig context")
}

//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="",resources=secrets/finalizers,verbs=update

// +kubebuilder:rbac:groups="tenancy.kcp.dev",resources=clusterworkspacetypes,verbs=get;list;watch;create;update;patch;delete;initialize
// +kubebuilder:rbac:groups="tenancy.kcp.dev",resources=clusterworkspacetypes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="tenancy.kcp.dev",resources=clusterworkspacetypes/finalizers,verbs=update

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", true,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	klog.InitFlags(flag.CommandLine)

	flag.Parse()
	flag.Lookup("v").Value.Set("3")

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	ctx := ctrl.SetupSignalHandler()

	restConfig := ctrl.GetConfigOrDie()
	setupLog.Info("Looking up virtual workspace URL")
	cfg, err := restConfigForCWT(ctx, restConfig, "appstudio")
	if err != nil {
		setupLog.Error(err, "error looking up virtual workspace URL")
	}

	setupLog.Info("Using virtual workspace URL", "url", cfg.Host)

	mgr, err := kcp.NewClusterAwareManager(cfg, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "f208085d.appstudio.kcp.com",
		LeaderElectionConfig:   restConfig,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.WorkspaceInitializer{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		Config: cfg,
		//APIExportPath: path,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Widget")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")

	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// restConfigForCWT returns a *rest.Config properly configured to communicate with the endpoint for the
// CWT's virtual workspace.
func restConfigForCWT(ctx context.Context, cfg *rest.Config, apiExportName string) (*rest.Config, error) {
	scheme := runtime.NewScheme()
	if err := tenancyv1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("error adding tenancy.kcp.dev/v1alpha1 to scheme: %w", err)
	}

	cwtClient, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("error creating CWT client: %w", err)
	}

	var cwt tenancyv1alpha1.ClusterWorkspaceType

	if err := cwtClient.Get(ctx, types.NamespacedName{Name: apiExportName}, &cwt); err != nil {
		return nil, fmt.Errorf("error getting CWT %q: %w", apiExportName, err)
	}

	if len(cwt.Status.VirtualWorkspaces) < 1 {
		return nil, fmt.Errorf("APIExport %q status.virtualWorkspaces is empty", apiExportName)
	}

	cfg = rest.CopyConfig(cfg)
	// TODO(ncdc): sharding support
	cfg.Host = cwt.Status.VirtualWorkspaces[0].URL

	return cfg, nil
}
