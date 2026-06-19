# Build from the repository root:
#   docker build -f services/gateway/Dockerfile -t usg-itsm/gateway .
#
# Multi-stage: a static Go binary on a distroless non-root base. The service
# module's `replace` directive points at ../../pkg, so only pkg + this service
# are needed in the build context (no go.work required).
# Pin digests in production for air-gapped reproducibility.

FROM golang:1.25 AS build
WORKDIR /src
COPY pkg/ ./pkg/
COPY services/gateway/ ./services/gateway/

WORKDIR /src/services/gateway
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/gateway ./cmd/gateway

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build /out/gateway /gateway
EXPOSE 8443
USER nonroot:nonroot
ENTRYPOINT ["/gateway"]
