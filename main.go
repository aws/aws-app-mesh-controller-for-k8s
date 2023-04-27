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
	"context"
	"github.com/aws/aws-sdk-go/service/eks"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws/throttle"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/cloudmap"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/references"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/version"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualrouter"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualservice"
	sdkgoaws "github.com/aws/aws-sdk-go/aws"
	"github.com/spf13/pflag"

	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/conversions"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/k8s"

	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws"
	zapraw "go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/gatewayroute"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/inject"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/mesh"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualgateway"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/virtualnode"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appmeshv1beta2 "github.com/aws/aws-app-mesh-controller-for-k8s/apis/appmesh/v1beta2"
	appmeshcontroller "github.com/aws/aws-app-mesh-controller-for-k8s/controllers/appmesh"
	appmeshwebhook "github.com/aws/aws-app-mesh-controller-for-k8s/webhooks/appmesh"
	corewebhook "github.com/aws/aws-app-mesh-controller-for-k8s/webhooks/core"
	// +kubebuilder:scaffold:imports
)

const (
	flagHealthProbePort    = "health-probe-port"
	defaultHealthProbePort = 61779
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
	var syncPeriod time.Duration
	var metricsAddr string
	var enableLeaderElection bool
	var enableCustomHealthCheck bool
	var logLevel string
	var listPageLimit int64
	var healthProbePort int
	var ipFamily string
	awsCloudConfig := aws.CloudConfig{ThrottleConfig: throttle.NewDefaultServiceOperationsThrottleConfig()}
	injectConfig := inject.Config{}
	cloudMapConfig := cloudmap.Config{}
	fs := pflag.NewFlagSet("", pflag.ExitOnError)
	fs.DurationVar(&syncPeriod, "sync-period", 10*time.Hour, "SyncPeriod determines the minimum frequency at which watched resources are reconciled.")
	fs.StringVar(&metricsAddr, "metrics-addr", "0.0.0.0:8080", "The address the metric endpoint binds to.")
	fs.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller. "+
			"Enabling this will ensure there is only one active controller.")
	fs.BoolVar(&enableCustomHealthCheck, "enable-custom-health-check", false,
		"Enable custom healthCheck when using cloudMap serviceDiscovery")
	fs.StringVar(&logLevel, "log-level", "info", "Set the controller log level - info(default), debug")
	fs.Int64Var(&listPageLimit, "page-limit", 100,
		"The page size limiting the number of response for list operation to API Server")
	fs.IntVar(&healthProbePort, flagHealthProbePort, defaultHealthProbePort,
		"The port the health probes binds to.")

	awsCloudConfig.BindFlags(fs)
	injectConfig.BindFlags(fs)
	cloudMapConfig.BindFlags(fs)
	if err := fs.Parse(os.Args); err != nil {
		setupLog.Error(err, "invalid flags")
		os.Exit(1)
	}

	if err := injectConfig.Validate(); err != nil {
		setupLog.Error(err, "invalid flags")
		os.Exit(1)
	}

	lvl := zapraw.NewAtomicLevelAt(0)
	if logLevel == "debug" {
		lvl = zapraw.NewAtomicLevelAt(-1)
	}
	ctrl.SetLogger(zap.New(zap.UseDevMode(false), zap.Level(&lvl)))
	setupLog.Info("version",
		"GitVersion", version.GitVersion,
		"GitCommit", version.GitCommit,
		"BuildDate", version.BuildDate,
	)

	awsCloudConfig.HandleAccountID(setupLog)
	parsedPort := strconv.Itoa(healthProbePort)
	healthProbeBindAddress := ":" + parsedPort
	setupLog.Info("Health endpoint", "HealthProbeBindAddress", healthProbeBindAddress)

	eventNotificationChan := make(chan k8s.GenericEvent)

	kubeConfig := ctrl.GetConfigOrDie()

	clientSet, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		setupLog.Error(err, "unable to generate clientSet from the kubeConfig")
		os.Exit(1)
	}

	k8sVersion := k8s.ServerVersion(clientSet.Discovery())

	mgr, err := ctrl.NewManager(kubeConfig, ctrl.Options{
		Scheme:                     scheme,
		SyncPeriod:                 &syncPeriod,
		MetricsBindAddress:         metricsAddr,
		Port:                       9443,
		LeaderElection:             enableLeaderElection,
		LeaderElectionID:           "appmesh-controller-leader-election",
		LeaderElectionResourceLock: resourcelock.ConfigMapsLeasesResourceLock,
		HealthProbeBindAddress:     healthProbeBindAddress,
	})

	customController := k8s.NewCustomController(
		clientSet,
		listPageLimit,
		metav1.NamespaceAll,
		conversions.NewPodConverter(),
		syncPeriod,
		false,
		eventNotificationChan,
		setupLog.WithName("pod custom controller"))

	if err != nil {
		setupLog.Error(err, "unable to start app mesh controller")
		os.Exit(1)
	}
	cloud, err := aws.NewCloud(awsCloudConfig, metrics.Registry)
	if err != nil {
		setupLog.Error(err, "unable to initialize AWS cloud")
		os.Exit(1)
	}

	setupLog.Info("Cluster Name", "ClusterName", injectConfig.ClusterName)

	if injectConfig.ClusterName != "" {
		input := &eks.DescribeClusterInput{
			Name: sdkgoaws.String(injectConfig.ClusterName),
		}
		result, err := cloud.EKS().DescribeCluster(input)
		if err != nil {
			setupLog.Info(err.Error(), "unable to get cluster info")
		} else {
			ipFamily = *result.Cluster.KubernetesNetworkConfig.IpFamily
			setupLog.Info("IpFamily of the cluster", "IpFamily", ipFamily)
		}
	} else {
		setupLog.Info("please provide a cluster-name using --set clusterName=name-of-your-cluster")
	}

	podsRepository := k8s.NewPodsRepository(customController)

	ctx := ctrl.SetupSignalHandler()
	referencesIndexer := references.NewDefaultObjectReferenceIndexer(mgr.GetCache(), mgr.GetFieldIndexer())
	finalizerManager := k8s.NewDefaultFinalizerManager(mgr.GetClient(), ctrl.Log)
	meshMembersFinalizer := mesh.NewPendingMembersFinalizer(mgr.GetClient(), mgr.GetEventRecorderFor("mesh-members"), ctrl.Log)
	vgMembersFinalizer := virtualgateway.NewPendingMembersFinalizer(mgr.GetClient(), mgr.GetEventRecorderFor("virtualgateway-members"), ctrl.Log)
	referencesResolver := references.NewDefaultResolver(mgr.GetClient(), ctrl.Log)
	virtualNodeEndpointResolver := cloudmap.NewDefaultVirtualNodeEndpointResolver(podsRepository, ctrl.Log)
	cloudMapInstancesReconciler := cloudmap.NewDefaultInstancesReconciler(mgr.GetClient(), cloud.CloudMap(), ctrl.Log, ctx.Done(), ipFamily)
	meshResManager := mesh.NewDefaultResourceManager(mgr.GetClient(), cloud.AppMesh(), cloud.AccountID(), ctrl.Log)
	vgResManager := virtualgateway.NewDefaultResourceManager(mgr.GetClient(), cloud.AppMesh(), referencesResolver, cloud.AccountID(), ctrl.Log)
	grResManager := gatewayroute.NewDefaultResourceManager(mgr.GetClient(), cloud.AppMesh(), referencesResolver, cloud.AccountID(), ctrl.Log)
	vnResManager := virtualnode.NewDefaultResourceManager(mgr.GetClient(), cloud.AppMesh(), referencesResolver, cloud.AccountID(), ctrl.Log, injectConfig.EnableBackendGroups)
	vsResManager := virtualservice.NewDefaultResourceManager(mgr.GetClient(), cloud.AppMesh(), referencesResolver, cloud.AccountID(), ctrl.Log)
	vrResManager := virtualrouter.NewDefaultResourceManager(mgr.GetClient(), cloud.AppMesh(), referencesResolver, cloud.AccountID(), ctrl.Log)
	cloudMapResManager := cloudmap.NewDefaultResourceManager(mgr.GetClient(), cloud.CloudMap(), referencesResolver, virtualNodeEndpointResolver, cloudMapInstancesReconciler, enableCustomHealthCheck, ctrl.Log, cloudMapConfig, ipFamily)
	msReconciler := appmeshcontroller.NewMeshReconciler(mgr.GetClient(), finalizerManager, meshMembersFinalizer, meshResManager, ctrl.Log.WithName("controllers").WithName("Mesh"), mgr.GetEventRecorderFor("Mesh"))
	vgReconciler := appmeshcontroller.NewVirtualGatewayReconciler(mgr.GetClient(), finalizerManager, vgMembersFinalizer, vgResManager, ctrl.Log.WithName("controllers").WithName("VirtualGateway"), mgr.GetEventRecorderFor("VirtualGateway"))
	grReconciler := appmeshcontroller.NewGatewayRouteReconciler(mgr.GetClient(), finalizerManager, grResManager, ctrl.Log.WithName("controllers").WithName("GatewayRoute"), mgr.GetEventRecorderFor("GatewayRoute"))
	vnReconciler := appmeshcontroller.NewVirtualNodeReconciler(mgr.GetClient(), finalizerManager, vnResManager, ctrl.Log.WithName("controllers").WithName("VirtualNode"), mgr.GetEventRecorderFor("VirtualNode"), injectConfig.EnableBackendGroups)

	cloudMapReconciler := appmeshcontroller.NewCloudMapReconciler(
		mgr.GetClient(),
		finalizerManager,
		cloudMapResManager,
		eventNotificationChan,
		ctrl.Log.WithName("controllers").WithName("CloudMap"),
		mgr.GetEventRecorderFor("CloudMap"))

	vsReconciler := appmeshcontroller.NewVirtualServiceReconciler(mgr.GetClient(), finalizerManager, referencesIndexer, vsResManager, ctrl.Log.WithName("controllers").WithName("VirtualService"), mgr.GetEventRecorderFor("VirtualService"))
	vrReconciler := appmeshcontroller.NewVirtualRouterReconciler(mgr.GetClient(), finalizerManager, referencesIndexer, vrResManager, ctrl.Log.WithName("controllers").WithName("VirtualRouter"), mgr.GetEventRecorderFor("VirtualRouter"))
	if err = msReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Mesh")
		os.Exit(1)
	}
	if err = vsReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VirtualService")
		os.Exit(1)
	}

	if err = vgReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VirtualGateway")
		os.Exit(1)
	}
	if err = grReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "GatewayRoute")
		os.Exit(1)
	}

	if err = vnReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VirtualNode")
		os.Exit(1)
	}
	if err = vrReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VirtualRouter")
		os.Exit(1)
	}
	if err = cloudMapReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CloudMap")
		os.Exit(1)
	}

	meshMembershipDesignator := mesh.NewMembershipDesignator(mgr.GetClient())
	vgMembershipDesignator := virtualgateway.NewMembershipDesignator(mgr.GetClient())
	vnMembershipDesignator := virtualnode.NewMembershipDesignator(mgr.GetClient())
	sidecarInjector := inject.NewSidecarInjector(injectConfig, cloud.AccountID(), cloud.Region(), version.GitVersion, k8sVersion, mgr.GetClient(), referencesResolver, vnMembershipDesignator, vgMembershipDesignator)
	appmeshwebhook.NewMeshMutator(ipFamily).SetupWithManager(mgr)
	appmeshwebhook.NewMeshValidator(ipFamily).SetupWithManager(mgr)
	appmeshwebhook.NewVirtualGatewayMutator(meshMembershipDesignator).SetupWithManager(mgr)
	appmeshwebhook.NewVirtualGatewayValidator().SetupWithManager(mgr)
	appmeshwebhook.NewGatewayRouteMutator(meshMembershipDesignator, vgMembershipDesignator).SetupWithManager(mgr)
	appmeshwebhook.NewGatewayRouteValidator().SetupWithManager(mgr)
	appmeshwebhook.NewVirtualNodeMutator(meshMembershipDesignator).SetupWithManager(mgr)
	appmeshwebhook.NewVirtualNodeValidator().SetupWithManager(mgr)
	appmeshwebhook.NewVirtualServiceMutator(meshMembershipDesignator).SetupWithManager(mgr)
	appmeshwebhook.NewVirtualServiceValidator().SetupWithManager(mgr)
	appmeshwebhook.NewVirtualRouterMutator(meshMembershipDesignator).SetupWithManager(mgr)
	appmeshwebhook.NewVirtualRouterValidator().SetupWithManager(mgr)
	appmeshwebhook.NewBackendGroupMutator(meshMembershipDesignator).SetupWithManager(mgr)
	appmeshwebhook.NewBackendGroupValidator().SetupWithManager(mgr)
	corewebhook.NewPodMutator(sidecarInjector).SetupWithManager(mgr)

	// Add liveness probe
	err = mgr.AddHealthzCheck("health-ping", healthz.Ping)
	setupLog.Info("adding health check for controller")
	if err != nil {
		setupLog.Error(err, "unable add a health check")
		os.Exit(1)
	}

	// Only start the controller when the leader election is won
	mgr.Add(manager.RunnableFunc(func(ctx context.Context) error {
		setupLog.Info("starting custom controller")

		// Start the custom controller
		customController.StartController(ctx.Done())
		// If the manager is stopped, signal the controller to stop as well.
		<-ctx.Done()

		setupLog.Info("stopping the controller")

		return nil
	}))

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting controller")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running controller")
		os.Exit(1)
	}
}
