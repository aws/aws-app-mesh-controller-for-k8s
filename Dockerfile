# syntax=docker/dockerfile:experimental

# Build the controller binary
FROM --platform=${TARGETPLATFORM} golang:1.16 as builder

WORKDIR /workspace

COPY . ./
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
ENV GOPROXY direct
RUN go mod download

ARG TARGETOS
ARG TARGETARCH

# Build
ENV VERSION_PKG=github.com/aws/aws-app-mesh-controller-for-k8s/pkg/version
RUN GIT_VERSION=$(git describe --tags --dirty --always) && \
    GIT_COMMIT=$(git rev-parse HEAD) && \
    BUILD_DATE=$(date +%Y-%m-%dT%H:%M:%S%z) && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GO111MODULE=on go build \
    -ldflags="-X ${VERSION_PKG}.GitVersion=${GIT_VERSION} -X ${VERSION_PKG}.GitCommit=${GIT_COMMIT} -X ${VERSION_PKG}.BuildDate=${BUILD_DATE}" -a -o controller main.go

# Build the container image
FROM public.ecr.aws/eks-distro-build-tooling/eks-distro-minimal-base:2021-10-19-1634675452

WORKDIR /
COPY --from=builder /workspace/controller .
COPY --from=builder /workspace/ATTRIBUTION.txt .

ENTRYPOINT ["/controller"]
