# Build from the repository root:
#   docker build -f services/identity/Dockerfile -t usg-itsm/identity .
#
# The service module's `replace` directive points at ../../pkg, so only pkg +
# this service are needed in the build context (no go.work required).

FROM golang:1.25 AS build
WORKDIR /src
COPY pkg/ ./pkg/
COPY services/identity/ ./services/identity/

WORKDIR /src/services/identity
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/identity ./cmd/identity

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build /out/identity /identity
EXPOSE 8444
USER nonroot:nonroot
ENTRYPOINT ["/identity"]
