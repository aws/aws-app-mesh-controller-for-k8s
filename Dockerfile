FROM golang:1.13-stretch as builder
WORKDIR /go/src/github.com/aws/aws-app-mesh-controller-for-k8s

ENV GOPRIVATE *

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
COPY --from=builder /go/src/github.com/aws/aws-app-mesh-controller-for-k8s/ATTRIBUTION.txt /ATTRIBUTION.txt

ENTRYPOINT ["/bin/app-mesh-controller"]
