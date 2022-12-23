# syntax=docker/dockerfile:experimental

# Build the controller binary
FROM --platform=${BUILDPLATFORM} golang:1.17 as builder

WORKDIR /workspace

COPY go.mod go.sum ./

# uncomment if using vendor
# COPY ./vendor ./vendor

ARG GOPROXY
ENV GOPROXY=${GOPROXY}

RUN go mod download

COPY ./main.go ./ATTRIBUTION.txt ./
COPY .git/ ./.git/
COPY pkg/ ./pkg/
COPY apis/ ./apis/
COPY controllers/ ./controllers/
COPY mocks/ ./mocks/
COPY webhooks/ ./webhooks/

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
