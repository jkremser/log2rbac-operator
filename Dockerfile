# gcr.io/distroless/static:static
FROM gcr.io/distroless/static@sha256:cb0f70353c21d1e472a9ed2055c8a62fd842de52dd2db9dfb31c6e019648bf51

WORKDIR /
COPY log2rbac .
USER nonroot:nonroot

ENTRYPOINT ["/log2rbac"]
CMD ["--zap-encoder=console"]
