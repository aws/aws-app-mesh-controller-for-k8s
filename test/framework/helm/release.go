package helm

import (
	"fmt"
	"github.com/aws/aws-app-mesh-controller-for-k8s/test/framework/utils"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"strings"
)

type Manager interface {
	// reset appMesh controller to default one installed by helm charts
	ResetAppMeshController() error
	// upgrade appMesh controller to new one with image overridden.
	UpgradeAppMeshController(controllerImage string) error
	// reset appMesh injector to default one installed by helm charts
	ResetAppMeshInjector() error
	// upgrade appMesh injector to new one with image overridden.
	UpgradeAppMeshInjector(injectorImage string) error

	// Upgrade a helm release
	UpgradeHelmRelease(chartRepo string, chartName string, namespace string, releaseName string, vals map[string]interface{}) (*release.Release, error)
}

func NewManager(kubeConfig string) Manager {
	return &defaultManager{
		kubeConfig: kubeConfig,
		logger:     utils.NewGinkgoLogger(),
	}
}

type defaultManager struct {
	kubeConfig string
	logger     *zap.Logger
}

func (m *defaultManager) ResetAppMeshController() error {
	vals := make(map[string]interface{})
	_, err := m.UpgradeHelmRelease(eksHelmChartsRepo, appMeshControllerHelmChart, appMeshSystemNamespace, appMeshControllerHelmReleaseName, vals)
	return err
}

func (m *defaultManager) UpgradeAppMeshController(controllerImage string) error {
	vals := make(map[string]interface{})
	imageRepo, imageTag, err := splitImageRepoAndTag(controllerImage)
	if err != nil {
		return err
	}
	vals["image"] = map[string]interface{}{
		"repository": imageRepo,
		"tag":        imageTag,
	}
	_, err = m.UpgradeHelmRelease(eksHelmChartsRepo, appMeshControllerHelmChart, appMeshSystemNamespace, appMeshControllerHelmReleaseName, vals)
	return err
}

func (m *defaultManager) ResetAppMeshInjector() error {
	vals := make(map[string]interface{})
	_, err := m.UpgradeHelmRelease(eksHelmChartsRepo, appMeshInjectorHelmChart, appMeshSystemNamespace, appMeshInjectorHelmReleaseName, vals)
	return err
}

func (m *defaultManager) UpgradeAppMeshInjector(injectorImage string) error {
	vals := make(map[string]interface{})
	imageRepo, imageTag, err := splitImageRepoAndTag(injectorImage)
	if err != nil {
		return err
	}
	vals["image"] = map[string]interface{}{
		"repository": imageRepo,
		"tag":        imageTag,
	}
	_, err = m.UpgradeHelmRelease(eksHelmChartsRepo, appMeshInjectorHelmChart, appMeshSystemNamespace, appMeshInjectorHelmReleaseName, vals)
	return err
}

func (m *defaultManager) UpgradeHelmRelease(chartRepo string, chartName string,
	namespace string, releaseName string, vals map[string]interface{}) (*release.Release, error) {
	cfgFlags := genericclioptions.NewConfigFlags(false)
	cfgFlags.KubeConfig = &m.kubeConfig
	cfgFlags.Namespace = &namespace
	actionConfig := new(action.Configuration)
	actionConfig.Init(cfgFlags, namespace, "secrets", func(format string, v ...interface{}) {
		message := fmt.Sprintf(format, v...)
		m.logger.Info(message)
	})
	upgradeAction := action.NewUpgrade(actionConfig)
	upgradeAction.ChartPathOptions.RepoURL = chartRepo
	upgradeAction.Namespace = namespace
	upgradeAction.ResetValues = true
	upgradeAction.Wait = true

	cp, err := upgradeAction.ChartPathOptions.LocateChart(chartName, cli.New())
	chartRequested, err := loader.Load(cp)
	if err != nil {
		return nil, err
	}
	return upgradeAction.Run(releaseName, chartRequested, vals)
}

// splitImageRepoAndTag parses a docker image in format <imageRepo>:<imageTag> into `imageRepo` and `imageTag`
func splitImageRepoAndTag(dockerImage string) (string, string, error) {
	parts := strings.Split(dockerImage, ":")
	if len(parts) != 2 {
		return "", "", errors.Errorf("dockerImage expects <imageRepo>:<imageTag>, got: %s", dockerImage)
	}
	return parts[0], parts[1], nil
}
