# Build from the repository root:
#   docker build -f services/gateway/Dockerfile -t usg-itsm/gateway .
#
# Three stages: build the SPA, build the static Go binary, then assemble a
# distroless image that serves both the API (proxied) and the SPA.
# Pin digests in production for air-gapped reproducibility.

FROM node:22 AS web
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

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
COPY --from=web /web/dist /web
ENV WEB_DIR=/web
EXPOSE 8443
USER nonroot:nonroot
ENTRYPOINT ["/gateway"]
