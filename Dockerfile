# gcr.io/distroless/static:static
FROM gcr.io/distroless/static@sha256:11364b4198394878b7689ad61c5ea2aae2cd2ed127c09fc7b68ca8ed63219030

WORKDIR /
COPY log2rbac .
USER nonroot:nonroot

ENTRYPOINT ["/log2rbac"]
CMD ["--zap-encoder=console"]
