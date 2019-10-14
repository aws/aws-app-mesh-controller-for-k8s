package main

import (
	goflag "flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	//Tools
	_ "k8s.io/code-generator"
	_ "github.com/vektra/mockery"

	// TODO(nic) Don't depend on k8s.io/kubernetes, just duplicate the logic in this package -- it will be a
	// smaller headache.
	//_ "k8s.io/kubernetes/pkg/client/metrics/prometheus" // for client metric registration
	//_ "k8s.io/kubernetes/pkg/util/reflector/prometheus" // for reflector metric registration
	//_ "k8s.io/kubernetes/pkg/util/workqueue/prometheus" // for workqueue metric registration

	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/aws"
	meshclientset "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/client/clientset/versioned"
	meshinformers "github.com/aws/aws-app-mesh-controller-for-k8s/pkg/client/informers/externalversions"
	"github.com/aws/aws-app-mesh-controller-for-k8s/pkg/controller"
)

var (
	cfgFile     string
	master      string
	kubeconfig  string
	region      string
	threadiness int
)

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.Flags().StringVarP(&cfgFile, "config", "c", "", "Load configuration from `filename`")
	rootCmd.Flags().StringVar(&master, "master", "", "Master address")
	rootCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to your kubeconfig")
	rootCmd.Flags().StringVar(&region, "aws-region", "", "AWS Region")
	rootCmd.Flags().IntVar(&threadiness, "threadiness", controller.DefaultThreadiness, "Worker concurrency.")

	viper.BindPFlag("master", rootCmd.Flags().Lookup("master"))
	viper.BindPFlag("kubeconfig", rootCmd.Flags().Lookup("kubeconfig"))
	viper.BindPFlag("aws-region", rootCmd.Flags().Lookup("aws-region"))
}

func main() {
	flag.CommandLine.Parse([]string{})

	fs := rootCmd.Flags()
	addKlogFlags(fs)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func initConfig() {
	if cfgFile == "" {
		return
	}
	viper.SetConfigFile(cfgFile)
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Can't read configuration file %q: %v\n", cfgFile, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:  "app-mesh-controller",
	Long: `app-mesh-controller is responsible for creating resources in AWS App Mesh.`,
	Run: func(cmd *cobra.Command, args []string) {
		version := NewVersion()
		fmt.Printf("%s\n", version.String())

		cmd.Flags().VisitAll(func(flag *flag.Flag) {
			klog.V(1).Infof("FLAG: --%s=%q", flag.Name, flag.Value)
		})

		var stopCh chan struct{}

		cfg, err := getConfig()
		if err != nil {
			klog.Fatal(err)
		}

		cloud, err := aws.NewCloud(cfg.aws)
		if err != nil {
			klog.Fatal(err)
		}

		// creates the connection
		config, err := clientcmd.BuildConfigFromFlags(cfg.client.Master, cfg.client.Kubeconfig)
		if err != nil {
			klog.Fatal(err)
		}

		// creates kube clientset
		kubeclientset := kubernetes.NewForConfigOrDie(config)

		// creates clientset for our custom resources
		meshclientset := meshclientset.NewForConfigOrDie(config)

		kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeclientset, time.Second*30)
		meshInformerFactory := meshinformers.NewSharedInformerFactory(meshclientset, time.Second*30)

		c, err := controller.NewController(
			cloud,
			kubeclientset,
			meshclientset,
			kubeInformerFactory.Core().V1().Pods(),
			meshInformerFactory.Appmesh().V1beta1().Meshes(),
			meshInformerFactory.Appmesh().V1beta1().VirtualNodes(),
			meshInformerFactory.Appmesh().V1beta1().VirtualServices(),
		)

		if err != nil {
			klog.Fatalf("Error running controller: %s", err)
		}

		kubeInformerFactory.Start(stopCh)
		meshInformerFactory.Start(stopCh)

		httpServer := controller.NewServer(cfg.server)
		go func() {
			klog.Fatal(httpServer.ListenAndServe())
		}()

		klog.Infof("Running controller with threadiness=%d", threadiness)
		if err := c.Run(threadiness, stopCh); err != nil {
			klog.Fatal(err)
		}
	},
}

type controllerConfig struct {
	client controller.ClientOptions
	server controller.ServerOptions
	aws    aws.CloudOptions
}

func getConfig() (controllerConfig, error) {
	viper.SetDefault("master", "")
	viper.SetDefault("listenAddress", ":10555")

	return controllerConfig{
		client: controller.ClientOptions{
			Master:     viper.GetString("master"),
			Kubeconfig: viper.GetString("kubeconfig"),
		},
		server: controller.ServerOptions{
			Address: viper.GetString("listenAddress"),
		},
		aws: aws.CloudOptions{
			Region: viper.GetString("aws-region"),
		},
	}, nil
}

// addKlogFlags adds flags from k8s.io/klog
func addKlogFlags(fs *flag.FlagSet) {
	local := goflag.NewFlagSet(os.Args[0], goflag.ExitOnError)
	klog.InitFlags(local)
	local.VisitAll(func(fl *goflag.Flag) {
		fl.Name = normalize(fl.Name)
		fs.AddGoFlag(fl)
	})
}

// normalize replaces underscores with hyphens
// we should always use hyphens instead of underscores when registering component flags
func normalize(s string) string {
	return strings.Replace(s, "_", "-", -1)
}
