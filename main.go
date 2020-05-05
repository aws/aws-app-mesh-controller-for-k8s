/*


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
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/references"
	"os"

	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"

	zapraw "go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/inject"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/mesh"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualnode"

	appmeshv1beta2 "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	appmeshcontroller "github.com/aws/aws-app-mesh-controller-for-k8s/controllers/appmesh"
	appmeshwebhook "github.com/aws/aws-app-mesh-controller-for-k8s/webhooks/appmesh"
	corewebhook "github.com/aws/aws-app-mesh-controller-for-k8s/webhooks/core"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = appmeshv1beta2.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	var injectConfig inject.Config
	injectConfig.BindFlags()
	flag.Parse()
	if err := injectConfig.Validate(); err != nil {
		setupLog.Error(err, "invalid flags")
		os.Exit(1)
	}

	// TODO: make level configurable
	lvl := zapraw.NewAtomicLevelAt(-2)
	ctrl.SetLogger(zap.New(zap.UseDevMode(true), zap.Level(&lvl)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "f4abe831.k8s.aws",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// TODO: organize component initialization below
	cloud, err := aws.NewCloud(aws.CloudConfig{}, metrics.Registry)
	if err != nil {
		setupLog.Error(err, "unable to initialize AWS cloud")
		os.Exit(1)
	}

	finalizerManager := k8s.NewDefaultFinalizerManager(mgr.GetClient(), ctrl.Log)
	meshMembersFinalizer := mesh.NewPendingMembersFinalizer(mgr.GetClient(), mgr.GetEventRecorderFor("mesh-members"), ctrl.Log)
	referencesResolver := references.NewDefaultResolver(mgr.GetClient(), ctrl.Log)
	meshResManager := mesh.NewDefaultResourceManager(mgr.GetClient(), cloud.AppMesh(), cloud.AccountID(), ctrl.Log)
	vnResManager := virtualnode.NewDefaultResourceManager(mgr.GetClient(), cloud.AppMesh(), referencesResolver, cloud.AccountID(), ctrl.Log)
	msReconciler := appmeshcontroller.NewMeshReconciler(mgr.GetClient(), finalizerManager, meshMembersFinalizer, meshResManager, ctrl.Log.WithName("controllers").WithName("Mesh"))
	vnReconciler := appmeshcontroller.NewVirtualNodeReconciler(mgr.GetClient(), finalizerManager, vnResManager, ctrl.Log.WithName("controllers").WithName("VirtualNode"))

	if err = msReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Mesh")
		os.Exit(1)
	}
	if err = (&appmeshcontroller.VirtualServiceReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("VirtualService"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VirtualService")
		os.Exit(1)
	}
	if err = vnReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VirtualNode")
		os.Exit(1)
	}
	if err = (&appmeshcontroller.VirtualRouterReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("VirtualRouter"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VirtualRouter")
		os.Exit(1)
	}

	meshMembershipDesignator := mesh.NewMembershipDesignator(mgr.GetClient())
	vnMembershipDesignator := virtualnode.NewMembershipDesignator(mgr.GetClient())
	sidecarInjector := inject.NewSidecarInjector(injectConfig, cloud.Region())
	appmeshwebhook.NewMeshMutator().SetupWithManager(mgr)
	appmeshwebhook.NewMeshValidator().SetupWithManager(mgr)
	appmeshwebhook.NewVirtualNodeMutator(meshMembershipDesignator).SetupWithManager(mgr)
	appmeshwebhook.NewVirtualNodeValidator().SetupWithManager(mgr)
	appmeshwebhook.NewVirtualServiceMutator(meshMembershipDesignator).SetupWithManager(mgr)
	appmeshwebhook.NewVirtualServiceValidator().SetupWithManager(mgr)
	corewebhook.NewPodMutator(referencesResolver, vnMembershipDesignator, sidecarInjector).SetupWithManager(mgr)

	if err = (&appmeshcontroller.VirtualGatewayReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("VirtualGateway"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VirtualGateway")
		os.Exit(1)
	}
	if err = (&appmeshcontroller.GatewayRouteReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("GatewayRoute"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "GatewayRoute")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
