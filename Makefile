PKG=github.com/aws/aws-app-mesh-controller-for-k8s
GIT_COMMIT?=$(shell git rev-parse HEAD)
BUILD_DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS?="-X main.version=${VERSION} -X main.gitCommit=${GIT_COMMIT} -X main.buildDate=${BUILD_DATE} -s -w"
GO111MODULE=on
# Docker
IMAGE=amazon/app-mesh-controller
REGION=$(shell aws configure get region)
REPO=$(AWS_ACCOUNT).dkr.ecr.$(AWS_REGION).amazonaws.com/$(IMAGE)
VERSION=1.0.0-alpha

.PHONY: eks-appmesh-controller
eks-appmesh-controller:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -ldflags ${LDFLAGS} -o _output/bin/app-mesh-controller ./cmd/app-mesh-controller

.PHONY: darwin
darwin:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=darwin go build -ldflags ${LDFLAGS} -o _output/bin/app-mesh-controller-osx ./cmd/app-mesh-controller

.PHONY: code-gen
code-gen:
	./hack/update-codegen.sh

.PHONY: image
image:
	docker build -t $(IMAGE):latest .

.PHONY: image-release
image-release:
	docker build -t $(IMAGE):$(VERSION) .

.PHONY: push
push:
ifeq ($(AWS_ACCOUNT),)
	$(error AWS_ACCOUNT is not set)
endif
	docker tag $(IMAGE):latest $(REPO):latest
	docker push $(REPO):latest

.PHONY: push-release
push-release:
	docker tag $(IMAGE):$(VERSION) $(REPO):$(VERSION)
	docker push $(REPO):$(VERSION)

.PHONY: deployk8s
deployk8s:
	./hack/deploy.sh

.PHONY: example
example:
	./hack/example.sh

.PHONY: clean
clean:
	rm -rf ./_output
