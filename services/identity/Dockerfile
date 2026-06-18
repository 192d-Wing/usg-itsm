# Build from the repository root:
#   docker build -f services/identity/Dockerfile -t usg-itsm/identity .

FROM golang:1.25 AS build
WORKDIR /src

COPY go.work ./
COPY pkg/ ./pkg/
COPY services/identity/ ./services/identity/

RUN go work edit -dropuse ./services/gateway

ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" \
    -o /out/identity ./services/identity/cmd/identity

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build /out/identity /identity
EXPOSE 8444
USER nonroot:nonroot
ENTRYPOINT ["/identity"]
