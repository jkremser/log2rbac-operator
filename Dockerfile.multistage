# build args
ARG GOLANG_VERSION=1.17.5
ARG GIT_SHA=unknown
ARG VERSION=unknown

# Build the log2rbac binary
FROM docker.io/golang:${GOLANG_VERSION} as builder

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY internal/ internal/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-w -s -X main.gitSha=${GIT_SHA} -X main.version=${VERSION}" -a -o log2rbac main.go

# Use distroless as minimal base image to package the log2rbac (manager) binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
ARG GOLANG_VERSION
ARG GIT_SHA
ARG VERSION
LABEL BASE_IMAGE="gcr.io/distroless/static:nonroot" \
      GOLANG_VERSION=${GOLANG_VERSION} \
      GIT_SHA=${GIT_SHA} \
      VERSION=${VERSION}
ENV GOLANG_VERSION=${GOLANG_VERSION} \
    GIT_SHA=${GIT_SHA} \
    VERSION=${VERSION}

WORKDIR /
COPY --from=builder /workspace/log2rbac .
USER 65532:65532

ENTRYPOINT ["/log2rbac"]
CMD ["--zap-encoder=console"]
