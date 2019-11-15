PKG=github.com/aws/aws-app-mesh-controller-for-k8s
GIT_COMMIT?=$(shell git rev-parse HEAD)
BUILD_DATE?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS?="-X main.version=${VERSION} -X main.gitCommit=${GIT_COMMIT} -X main.buildDate=${BUILD_DATE} -s -w"
GO111MODULE=on
# Docker
IMAGE=amazon/app-mesh-controller
REPO=$(AWS_ACCOUNT).dkr.ecr.$(AWS_REGION).amazonaws.com/$(IMAGE)
VERSION=v0.3.0

.PHONY: eks-appmesh-controller
eks-appmesh-controller:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux go build -ldflags ${LDFLAGS} -o _output/bin/app-mesh-controller ./cmd/app-mesh-controller

.PHONY: darwin
darwin:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=darwin go build -ldflags ${LDFLAGS} -o _output/bin/app-mesh-controller-osx ./cmd/app-mesh-controller

.PHONY: linux
linux:
	mkdir -p _output/bin
	CGO_ENABLED=0 GOOS=linux go build -ldflags ${LDFLAGS} -o _output/bin/app-mesh-controller ./cmd/app-mesh-controller


.PHONY: code-gen
code-gen:
	./scripts/update-codegen.sh

.PHONY: verify-codegen
verify-codegen:
	./scripts/verify-codegen.sh

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

.PHONY: deploy
deploy:
	./scripts/deploy.sh

.PHONY: example
example:
	./scripts/example.sh

.PHONY: release-example-gen
release-example-gen:
	RELEASE=1 ./scripts/example.sh

.PHONY: clean
clean:
	rm -rf ./_output

.PHONY: mock-gen
mock-gen:
	./scripts/mockgen.sh

PACKAGES:=$(shell go list ./... | sed -n '1!p' | grep ${PKG}/pkg/controller)
test:
	echo "mode: count" > coverage-all.out
	$(foreach pkg,$(PACKAGES), \
		go test -p=1 -cover -covermode=count -coverprofile=coverage.out ${pkg}; \
		tail -n +2 coverage.out >> coverage-all.out;)

cover: test
	go tool cover -html=coverage-all.out

go-fmt:
	gofmt -l pkg/* | grep ".*\.go"; if [ "$$?" = "0" ]; then exit 1; fi;
