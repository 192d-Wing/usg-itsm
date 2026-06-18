# Build from the repository root:
#   docker build -f services/gateway/Dockerfile -t usg-itsm/gateway .
#
# Multi-stage: a static Go binary on a distroless non-root base.
# Pin digests in production for air-gapped reproducibility.

FROM golang:1.25 AS build
WORKDIR /src

# Copy the workspace and modules needed by the gateway (it depends on /pkg).
COPY go.work ./
COPY pkg/ ./pkg/
COPY services/gateway/ ./services/gateway/

# Drop the identity module from the workspace so this image builds standalone.
RUN go work edit -dropuse ./services/identity

ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" \
    -o /out/gateway ./services/gateway/cmd/gateway

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build /out/gateway /gateway
EXPOSE 8443
USER nonroot:nonroot
ENTRYPOINT ["/gateway"]
