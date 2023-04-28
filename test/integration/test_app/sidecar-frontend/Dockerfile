FROM public.ecr.aws/amazonlinux/amazonlinux:2 AS builder
RUN yum update -y && \
    yum install -y ca-certificates unzip tar gzip git && \
    yum clean all && \
    rm -rf /var/cache/yum

RUN curl -LO https://golang.org/dl/go1.19.1.linux-amd64.tar.gz && \
    tar -C /usr/local -xzvf go1.19.1.linux-amd64.tar.gz

ENV PATH="${PATH}:/usr/local/go/bin"
ENV GOPATH="${HOME}/go"
ENV PATH="${PATH}:${GOPATH}/bin"

ENV GOPROXY=direct

# go.mod and go.sum go into their own layers.
# This ensures `go mod download` happens only when go.mod and go.sum change.
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix nocgo -o /sidecar-frontend .

FROM public.ecr.aws/amazonlinux/amazonlinux:2
RUN yum update -y && \
    yum install -y ca-certificates && \
    yum clean all && \
    rm -rf /var/cache/yum
COPY --from=builder /sidecar-frontend /sidecar-frontend

ENTRYPOINT ["/sidecar-frontend"]
