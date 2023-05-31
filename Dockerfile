# gcr.io/distroless/static:static
FROM gcr.io/distroless/static@sha256:7198a357ff3a8ef750b041324873960cf2153c11cc50abb9d8d5f8bb089f6b4e

WORKDIR /
COPY log2rbac .
USER nonroot:nonroot

ENTRYPOINT ["/log2rbac"]
CMD ["--zap-encoder=console"]
