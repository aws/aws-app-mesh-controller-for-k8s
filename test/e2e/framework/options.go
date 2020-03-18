package framework

import (
	"flag"
	"github.com/pkg/errors"
	"k8s.io/client-go/tools/clientcmd"
)

var GlobalOptions Options

func init() {
	GlobalOptions.BindFlags()
}

type Options struct {
	KubeConfig  string
	ClusterName string
	AWSRegion   string
	AWSVPCID    string

	// appMesh controller image. leave empty to use default one from helm chart.
	ControllerImage string

	// appmesh injector image, leave empty to use default one from helm chart.
	InjectorImage string
}

func (options *Options) BindFlags() {
	flag.StringVar(&options.KubeConfig, clientcmd.RecommendedConfigPathFlag, "", "Path to kubeconfig containing embedded authinfo (required)")
	flag.StringVar(&options.ClusterName, "cluster-name", "", `Kubernetes cluster name (required)`)
	flag.StringVar(&options.AWSRegion, "aws-region", "", `AWS Region for the kubernetes cluster`)
	flag.StringVar(&options.AWSVPCID, "aws-vpc-id", "", `AWS VPC ID for the kubernetes cluster`)

	flag.StringVar(&options.ControllerImage, "controller-image", "", `appMesh controller image. leave empty to use default one from helm chart`)
	flag.StringVar(&options.InjectorImage, "injector-image", "", `appMesh injector image, leave empty to use default one from helm chart`)
}

func (options *Options) Validate() error {
	if len(options.KubeConfig) == 0 {
		return errors.Errorf("%s must be set!", clientcmd.RecommendedConfigPathFlag)
	}
	if len(options.ClusterName) == 0 {
		return errors.Errorf("%s must be set!", "cluster-name")
	}
	if len(options.AWSRegion) == 0 {
		return errors.Errorf("%s must be set!", "aws-region")
	}
	if len(options.AWSVPCID) == 0 {
		return errors.Errorf("%s must be set!", "aws-vpc-id")
	}
	return nil
}
