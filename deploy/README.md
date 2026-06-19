# Deployment

Kubernetes deployment artifacts (Phase 4). Planned layout:

- `helm/` — umbrella chart plus a subchart per service.
- `kustomize/` — environment overlays (dev / staging / prod).
- `argocd/` — GitOps `Application` manifests for pull-based delivery.

All images are sourced from a private **Harbor** registry so the platform
installs fully air-gapped. TLS is issued in-cluster by cert-manager with an
internal CA (TLS 1.3 / mTLS per ADR-0007).

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
