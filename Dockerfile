# gcr.io/distroless/static:static
FROM gcr.io/distroless/static@sha256:7e5c6a2a4ae854242874d36171b31d26e0539c98fc6080f942f16b03e82851ab

WORKDIR /
COPY log2rbac .
USER nonroot:nonroot

ENTRYPOINT ["/log2rbac"]
CMD ["--zap-encoder=console"]
