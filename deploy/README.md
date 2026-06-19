# Deployment

Kubernetes deployment artifacts. Layout:

- `kustomize/base/` — namespace, config, the four app services
  (gateway, identity, ticketing, notification) and dev infra
  (Postgres, NATS/JetStream, Keycloak). All Services are IPv6 SingleStack.
- `kustomize/overlays/dev/` — the dev cluster: gateway `LoadBalancer` holding
  the Cilium anycast VIP, and the image registry.

All images are sourced from a private **Harbor** registry so the platform
installs fully air-gapped.

## Dev cluster

`itsm.dev.mil` (AAAA → the anycast VIP) is the single entry point: the gateway
serves the SPA and proxies `/api`. The VIP is assigned to the gateway
`LoadBalancer` Service by Cilium LB IPAM (`overlays/dev/ip-pool.yaml` +
`gateway-lb.yaml`).

### Prerequisites (must be provided per environment)

1. **Images in a reachable registry.** Build and push all four service images
   (the gateway image also bundles the built SPA). Set the registry in
   `overlays/dev/kustomization.yaml` (`images:`):
   ```sh
   for s in gateway identity ticketing notification; do
     docker build -f services/$s/Dockerfile -t harbor.dev.mil/usg-itsm/$s:latest .
     docker push harbor.dev.mil/usg-itsm/$s:latest
   done
   ```
2. **DNS:** `itsm.dev.mil` AAAA → `2601:443:c200:575::60`.
3. **Secrets:** replace the dev placeholders in `base/config.yaml` (Postgres
   password, `DATABASE_URL`, Keycloak admin) with Sealed-Secrets or Vault.
4. **Keycloak reachability:** the browser reaches Keycloak at
   `https://itsm.dev.mil/realms/...` (matching `OIDC_ISSUER` and the token
   `iss`). Route `/realms`, `/resources`, `/admin` through the gateway/ingress
   to the `keycloak` Service, or expose Keycloak on its own host and update
   `OIDC_ISSUER` + the realm redirect URIs.
5. **Edge TLS:** dev runs `ENVIRONMENT=dev` so services use in-memory
   self-signed certs (browsers warn). For trusted certs and internal mTLS,
   issue certs via cert-manager and set `TLS_CERT_FILE`/`TLS_KEY_FILE` and
   `INTERNAL_CA_FILE` (ADR-0007).

### Apply

```sh
kubectl kustomize deploy/kustomize/overlays/dev          # render & review
kubectl apply -k deploy/kustomize/overlays/dev           # apply
kubectl -n usg-itsm get pods,svc                         # watch rollout
kubectl -n usg-itsm get svc gateway -o wide              # confirm the VIP
```

> Cilium must have LB IPAM enabled and announce the VIP (BGP or L2). The
> `CiliumLoadBalancerIPPool` (`ip-pool.yaml`) holds `2601:443:c200:575::60/128`.

This is a working **dev** baseline, not yet hardened: add
`CiliumNetworkPolicy` default-deny + allowlists, PodDisruptionBudgets, resource
tuning, and HA Postgres/NATS for staging/prod.

## Networking (ADR-0009)

- **CNI: Cilium** (eBPF). NetworkPolicy is authored as `CiliumNetworkPolicy`,
  default-deny per namespace, with explicit service-to-service allowlists.
- **IPv6-only internally.** Services use `ipFamilyPolicy: SingleStack` /
  `ipFamilies: [IPv6]`; pods and ClusterIPs are IPv6. Containers bind
  `[::]:<port>`.
- **IPv4 at the edge only**, via a **load-balancer VIP** that fronts the IPv6
  backend. The edge LB Service is the sole place IPv4 appears.
- Cluster endpoints (API server, ingress VIPs) are per-environment values, not
  committed here.
