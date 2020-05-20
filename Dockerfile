# Build the manager binary
FROM golang:1.14 as builder

WORKDIR /workspace

COPY . ./
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Build
ENV VERSION_PKG=github.com/aws/aws-app-mesh-controller-for-k8s/pkg/version
RUN GIT_VERSION=$(git describe --tags --dirty) && \
    GIT_COMMIT=$(git rev-parse HEAD) && \
    BUILD_DATE=$(date +%Y-%m-%dT%H:%M:%S%z) && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build \
    -ldflags="-X ${VERSION_PKG}.GitVersion=${GIT_VERSION} -X ${VERSION_PKG}.GitCommit=${GIT_COMMIT} -X ${VERSION_PKG}.BuildDate=${BUILD_DATE}" -a -o manager main.go

# Build the container image
FROM amazonlinux:2
RUN yum update -y && \
    yum install -y ca-certificates && \
    yum clean all && \
    rm -rf /var/cache/yum

WORKDIR /
COPY --from=builder /workspace/manager .

ENTRYPOINT ["/manager"]
