# 0009 — IPv6-only internal networking on Cilium

- Status: Accepted
- Date: 2026-06-18

## Context

The target Kubernetes environment is IPv6-first and uses Cilium (eBPF) as the
CNI. Operators want a single, modern address family inside the cluster and
IPv4 reachability handled only where external clients require it.

## Decision

- **All internal traffic is IPv6.** Pod-to-pod and service-to-service
  communication, Service ClusterIPs, and probes use IPv6. Services bind
  `[::]:<port>` (Go's default `:<port>` listener binds `[::]` and works on the
  IPv6-only pod network).
- **IPv4 is terminated at the edge only**, via a **VIP on the load balancer**.
  No IPv4 exists inside the mesh; the LB translates to the IPv6 backend.
- **Cilium is the CNI.** NetworkPolicy, LB/IPAM, and L7 controls are authored
  for Cilium (`CiliumNetworkPolicy` where L3/L4 `NetworkPolicy` is insufficient),
  with IPv6 enabled.

## Consequences

- Helm values and Service manifests set `ipFamilyPolicy: SingleStack` with
  `ipFamilies: [IPv6]`; only the edge LB Service exposes IPv4 via the VIP.
- mTLS/cert SANs must include IPv6 addresses; the dev self-signed cert already
  lists the IPv6 loopback.
- Default-deny `CiliumNetworkPolicy` per namespace; service-to-service flows are
  allowlisted explicitly (defense in depth alongside mTLS, ADR-0007).
- The **local docker-compose** dev stack remains IPv4 `localhost` — the IPv6
  requirement applies to the Kubernetes environment, not laptop development.
- Concrete cluster endpoints are environment configuration, not committed to
  the repository.
