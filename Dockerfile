FROM gcr.io/distroless/static:nonroot

WORKDIR /
COPY log2rbac .
USER nonroot:nonroot

ENTRYPOINT ["/log2rbac"]
CMD ["--zap-encoder=console"]
