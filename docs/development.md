# Development

## Before you get started

Please read the [contributing](../CONTRIBUTING.md) guidelines before getting started.

## Guide

Following steps will help you get ready with a local development stack to contribute and test your changes before publishing a PR.

- Follow the [installation](install.md) steps. It is recommended to use the Helm to setup App Mesh controller.
- Clone and checkout aws-app-mesh-controller-for-k8s.
- Make code or configuration changes.
- Build and push container image.
```
make image push
```
- Deploy latest changes to appmesh-controller
```
make deploy
```
- Use examples from [aws-app-mesh-examples](https://github.com/aws/aws-app-mesh-examples/tree/master/walkthroughs) to verify the controller behavior.

## Updating App Mesh CRD

Following steps can be used as a checklist when updating CRD to use the latest features available via App Mesh.

- [ ] Update `aws-go-sdk` in go.mod to use the latest types from App Mesh
- [ ] Update CRD schema in `deploy/all.yaml`
- [ ] Update CRD structs in `pkg/apis/appmesh/v1beta1/types.go`
- [ ] Update deepcopy functions using `make code-gen`
- [ ] Update App Mesh client wrapper `pkg/aws/appmesh.go`
- [ ] Update controller(s) under `pkg/controller/`

