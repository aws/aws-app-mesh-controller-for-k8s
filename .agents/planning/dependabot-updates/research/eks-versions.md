# Research: EKS Supported Kubernetes Versions (June 2026)

## Currently Supported Versions

| K8s Version | EKS Release Date | Support Tier | End of Support |
|-------------|-----------------|--------------|----------------|
| **1.35** | January 27, 2026 | Standard | March 27, 2027 |
| **1.34** | October 2, 2025 | Standard | December 2, 2026 |
| **1.33** | May 29, 2025 | Standard | July 29, 2026 |
| **1.32** | January 23, 2025 | Extended | March 23, 2027 |
| **1.31** | September 26, 2024 | Extended | November 26, 2026 |
| **1.30** | May 23, 2024 | Extended | July 23, 2026 |

## Support Tiers

- **Standard Support** — included in base EKS pricing, 14 months from EKS release
- **Extended Support** — additional cost per cluster hour, 12 months after standard ends (26 total months per version)

## Implications for AppMesh Controller

The AppMesh controller runs on customer EKS clusters. Given AppMesh sunset is **September 2026**:

- K8s **1.30** extended support ends July 2026 (before sunset) — customers should be off by then
- K8s **1.31** extended support ends November 2026 (after sunset)
- K8s **1.32** extended support ends March 2027 (well after sunset)
- K8s **1.33–1.35** in standard support through and beyond sunset

**Conclusion:** Targeting k8s.io v0.32 (Kubernetes 1.32) with client-go backwards compatibility means the controller will work on all currently-supported EKS versions (1.30–1.35) that customers could be running before AppMesh sunset.

## Version Skew Policy

Kubernetes guarantees that client-go is compatible with clusters ±2 minor versions. A v0.32 client is officially compatible with K8s 1.30–1.34 clusters, and in practice works with 1.35 as well (just won't have client types for 1.33+ APIs).

## Source

https://docs.aws.amazon.com/eks/latest/userguide/kubernetes-versions.html (retrieved 2026-06-02)
