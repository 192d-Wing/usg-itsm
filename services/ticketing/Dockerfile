# Build from the repository root:
#   docker build -f services/ticketing/Dockerfile -t usg-itsm/ticketing .
#
# The service module's `replace` directive points at ../../pkg, so only pkg +
# this service are needed in the build context (no go.work required).

FROM golang:1.25 AS build
WORKDIR /src
COPY pkg/ ./pkg/
COPY services/ticketing/ ./services/ticketing/

WORKDIR /src/services/ticketing
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/ticketing ./cmd/ticketing

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build /out/ticketing /ticketing
EXPOSE 8445
USER nonroot:nonroot
ENTRYPOINT ["/ticketing"]
