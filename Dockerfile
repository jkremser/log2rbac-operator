# gcr.io/distroless/static:static
FROM gcr.io/distroless/static@sha256:5759d194607e472ff80fff5833442d3991dd89b219c96552837a2c8f74058617

WORKDIR /
COPY log2rbac .
USER nonroot:nonroot

ENTRYPOINT ["/log2rbac"]
CMD ["--zap-encoder=console"]
