FROM golang:1.12-stretch as builder
WORKDIR /go/src/github.com/aws/aws-app-mesh-controller-for-k8s

# Force the go compiler to use modules.
ENV GO111MODULE=on

# go.mod and go.sum go into their own layers.
COPY go.mod .
COPY go.sum .

# This ensures `go mod download` happens only when go.mod and go.sum change.
RUN go mod download

COPY . .
RUN make

FROM amazonlinux:2
RUN yum install -y ca-certificates
COPY --from=builder /go/src/github.com/aws/aws-app-mesh-controller-for-k8s/_output/bin/app-mesh-controller /bin/app-mesh-controller

ENTRYPOINT ["/bin/app-mesh-controller"]