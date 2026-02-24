# Image URL to use all building/pushing image targets
IMAGE_NAME=amazon/appmesh-controller
REPO=$(AWS_ACCOUNT).dkr.ecr.$(AWS_REGION).amazonaws.com/$(IMAGE_NAME)
REPO_FULL_NAME=aws/aws-app-mesh-controller-for-k8s
BINARY_NAME ?= "appmesh-controller"
MAKEFILE_PATH = $(dir $(realpath -s $(firstword $(MAKEFILE_LIST))))
VERSION ?= $(shell git describe --dirty --tags --always)
IMAGE ?= $(REPO):$(VERSION)
PREVIEW=false
ENABLE_BACKEND_GROUPS?=false
WAIT_PROXY_READY=false
SIDECAR_IMAGE_TAG=v1.34.12.1-prod

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,crdVersions=v1"

# app mesh aws-sdk-go override in case we need to build against a custom version
APPMESH_SDK_OVERRIDE ?= "n"

ifeq ($(APPMESH_SDK_OVERRIDE), "y")
PREVIEW=true
endif

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: controller

# Run tests
test: generate fmt vet manifests
	go test -race ./pkg/... ./controllers/... ./webhooks/... -coverprofile cover.out

# Build controller binary
controller: generate fmt vet
	go build -o bin/controller main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests
	kustomize build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: check-env manifests
	cd config/controller && kustomize edit set image controller=$(IMAGE)
	kustomize build config/default | kubectl apply -f -

helm-lint:
	${MAKEFILE_PATH}/test/helm/helm-lint.sh

helm-deploy: check-env manifests
	helm upgrade -i appmesh-controller config/helm/appmesh-controller --namespace appmesh-system \
		--set image.repository=$(REPO) \
		--set image.tag=$(VERSION) \
		--set preview=$(PREVIEW) \
		--set enableBackendGroups=$(ENABLE_BACKEND_GROUPS) \
		--set sidecar.waitUntilProxyReady=$(WAIT_PROXY_READY) \
		--set sidecar.image.tag=$(SIDECAR_IMAGE_TAG)

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=controller-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	kustomize build config/crd > config/helm/appmesh-controller/crds/crds.yaml

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet: setup-appmesh-sdk-override
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
# Requires buildx to be installed: https://docs.docker.com/buildx/working-with-buildx/
# By default the TARGETPLATFORM is set to linux/amd64, change the value if you are building for linux/arm64
docker-build: check-env test
	docker buildx build --platform linux/amd64 --build-arg GOPROXY=$(GOPROXY) -t $(IMAGE) . --load

docker-push: check-env
	docker push $(IMAGE)

integration-test: ## Run the integration using kind clusters
	@./scripts/test-with-kind.sh

delete-all-kind-clusters:	## Delete all local kind clusters
	@kind get clusters | \
	while read name ; do \
	kind delete cluster --name $$name; \
	done
	@rm -rf build/tmp-test*

setup-appmesh-sdk-override:
	@if [ "$(APPMESH_SDK_OVERRIDE)" = "y" ] ; then \
	    ./appmesh_models_override/setup.sh ; \
	fi

cleanup-appmesh-sdk-override:
	@if [ "$(APPMESH_SDK_OVERRIDE)" = "y" ] ; then \
	    ./appmesh_models_override/cleanup.sh ; \
	fi

version:
	@echo $(VERSION)

ekscharts-sync:
	${MAKEFILE_PATH}/scripts/sync-to-eks-charts.sh -b ${BINARY_NAME} -r ${REPO_FULL_NAME}

ekscharts-sync-release:
	${MAKEFILE_PATH}/scripts/sync-to-eks-charts.sh -b ${BINARY_NAME} -r ${REPO_FULL_NAME} -n

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

check-env:
	@:$(call check_var, AWS_ACCOUNT, AWS account ID for publishing docker images)
	@:$(call check_var, AWS_REGION, AWS region for publishing docker images)
	
check_var = \
    $(strip $(foreach 1,$1, \
        $(call __check_var,$1,$(strip $(value 2)))))
__check_var = \
    $(if $(value $1),, \
      $(error Undefined variable $1$(if $2, ($2))))
