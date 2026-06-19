# Build from the repository root:
#   docker build -f services/notification/Dockerfile -t usg-itsm/notification .

FROM golang:1.25 AS build
WORKDIR /src
COPY pkg/ ./pkg/
COPY services/notification/ ./services/notification/

WORKDIR /src/services/notification
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/notification ./cmd/notification

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=build /out/notification /notification
EXPOSE 8446
USER nonroot:nonroot
ENTRYPOINT ["/notification"]
