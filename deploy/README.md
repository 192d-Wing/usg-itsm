# Deployment

Kubernetes deployment artifacts (Phase 4). Planned layout:

- `helm/` — umbrella chart plus a subchart per service.
- `kustomize/` — environment overlays (dev / staging / prod).
- `argocd/` — GitOps `Application` manifests for pull-based delivery.

All images are sourced from a private **Harbor** registry so the platform
installs fully air-gapped. TLS is issued in-cluster by cert-manager with an
internal CA (TLS 1.3 / mTLS per ADR-0007).
