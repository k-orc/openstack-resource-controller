/*
Copyright 2024.

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
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/k-orc/openstack-resource-controller/internal/controllers/export"
	internalmanager "github.com/k-orc/openstack-resource-controller/internal/manager"
	"github.com/k-orc/openstack-resource-controller/internal/scheme"
	orccontrollers "github.com/k-orc/openstack-resource-controller/pkg/controllers"
	// +kubebuilder:scaffold:imports
)

func main() {
	setupLog := ctrl.Log.WithName("setup")

	orcOpts := internalmanager.Options{}
	flag.StringVar(&orcOpts.MetricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	flag.StringVar(&orcOpts.ProbeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&orcOpts.EnableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&orcOpts.SecureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	flag.BoolVar(&orcOpts.EnableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	flag.IntVar(&orcOpts.ScopeCacheMaxSize, "scope-cache-max-size", 10,
		"The maximum credentials count the operator should keep in cache. "+
			"Setting this value to 0 means no cache.")

	zapOpts := zap.Options{
		Development: true,
	}
	zapOpts.BindFlags(flag.CommandLine)
	flag.Parse()

	log := zap.New(zap.UseFlagOptions(&zapOpts))
	ctrl.SetLogger(log)

	// Setup the context that's going to be used in controllers and for the manager.
	ctx := ctrl.SetupSignalHandler()

	// TODO: Implement custom caCerts
	var caCerts []byte
	scopeFactory := orccontrollers.NewScopeFactory(orcOpts.ScopeCacheMaxSize, caCerts)

	controllers := []export.Controller{
		orccontrollers.ImageController(scopeFactory),
		orccontrollers.NetworkController(scopeFactory),
		orccontrollers.SubnetController(scopeFactory),
		orccontrollers.RouterController(scopeFactory),
		orccontrollers.RouterInterfaceController(scopeFactory),
		orccontrollers.PortController(scopeFactory),
		orccontrollers.FlavorController(scopeFactory),
		orccontrollers.SecurityGroupController(scopeFactory),
		orccontrollers.ServerController(scopeFactory),
	}

	restConfig := ctrl.GetConfigOrDie()
	err := internalmanager.Run(ctx, &orcOpts, restConfig, scheme.New(), setupLog, log, controllers)
	if err != nil {
		setupLog.Error(err, "Error starting manager")
		os.Exit(1)
	}
}
