# Build the manager binary
FROM registry.access.redhat.com/ubi8/go-toolset:1.17.7 as builder

ARG GOVERSION="1.17.8"
ARG VERSION=""

WORKDIR /opt/app-root
COPY . .

RUN go build -ldflags "-X main.version=${VERSION}"

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.5-204

COPY --from=builder /opt/app-root/nflow-generator ./

ENTRYPOINT ["./nflow-generator"]