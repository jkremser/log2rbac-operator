# gcr.io/distroless/static:static
FROM gcr.io/distroless/static@sha256:d8afc7d6973f357162e2283551cf3347b2bb847a03d24510ee837f289505f8e3

WORKDIR /
COPY log2rbac .
USER nonroot:nonroot

ENTRYPOINT ["/log2rbac"]
CMD ["--zap-encoder=console"]
