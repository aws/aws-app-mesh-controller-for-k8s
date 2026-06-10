# Research: Dependency Upgrade Compatibility

## Current State (as of 2026-06-02)

| Component | Current Version | Age |
|-----------|----------------|-----|
| Go | 1.20 | ~3 years old (released Feb 2023) |
| k8s.io/client-go | v0.26.2 | Maps to Kubernetes 1.26 |
| controller-runtime | v0.14.6 | For k8s 1.26 |
| AWS SDK (v1) | v1.44.252 | Old, but still maintained |
| Helm | v3.11.3 | ~3 years old |
| Docker | 20.10.24 | Very old, security concerns |

## Pending Dependabot PRs (6 open)

| PR | Dependency | From → To | Security? |
|----|-----------|-----------|-----------|
| #811 | github.com/sirupsen/logrus | 1.9.0 → 1.9.1 | Minor |
| #809 | golang.org/x/crypto | 0.21.0 → 0.45.0 | **Yes - security** |
| #806 | golang.org/x/oauth2 | 0.7.0 → 0.27.0 | Yes |
| #805 | github.com/docker/docker | 20.10.24 → 25.0.13 | **Yes - security** |
| #795 | golang.org/x/crypto | 0.21.0 → 0.35.0 | Superseded by #809 |
| #765 | helm.sh/helm/v3 | 3.11.3 → 3.14.3 | Yes |

**Note:** PR #795 is superseded by #809 (same dep, newer target).

## EKS Kubernetes Versions (June 2026)

### Standard Support
- **1.35** (released Jan 2026, standard support until March 2027)
- **1.34** (released Oct 2025, standard support until Dec 2026)
- **1.33** (released May 2025, standard support until July 2026)

### Extended Support
- **1.32** (released Jan 2025, extended support until March 2027)
- **1.31** (released Sep 2024, extended support until Nov 2026)
- **1.30** (released May 2024, extended support until July 2026)

### Key Insight
The current k8s libraries (v0.26) target Kubernetes 1.26 which is **no longer supported** by EKS at all. The controller must be upgraded to remain compatible with currently-supported EKS versions.

## client-go ↔ Kubernetes Compatibility Matrix

| client-go version | K8s target | Go requirement |
|-------------------|-----------|----------------|
| v0.26.x (current) | K8s 1.26 | Go 1.19+ |
| v0.30.x | K8s 1.30 | Go 1.22+ |
| v0.31.x | K8s 1.31 | Go 1.22+ |
| v0.32.x | K8s 1.32 | Go 1.23+ |
| v0.33.x | K8s 1.33 | Go 1.23+ |
| v0.34.x | K8s 1.34 | Go 1.23+ |

client-go is backwards compatible: older clients work with newer clusters (with degraded functionality for new APIs). But since this is a controller for a sunsetting service (AppMesh sunset Sep 2026), we need to balance "modern enough" with "minimal risk."

## controller-runtime ↔ Kubernetes Compatibility

| controller-runtime | k8s.io libs | Go requirement |
|-------------------|-------------|----------------|
| v0.14.x (current) | v0.26.x | Go 1.19+ |
| v0.18.x | v0.30.x | Go 1.22+ |
| v0.19.x | v0.31.x | Go 1.22+ |
| v0.20.x | v0.32.x | Go 1.23+ |
| v0.21.x | v0.33.x | Go 1.23+ |

## Go Version Landscape (June 2026)

| Version | Status |
|---------|--------|
| Go 1.26 | Latest (released Feb 2026) |
| Go 1.25 | Supported |
| Go 1.24 | EOL (Feb 2025 → Aug 2025) - wait, per search: released Feb 2025 |
| Go 1.23 | EOL (Aug 2025) |
| Go 1.22 | EOL |
| Go 1.20 (current) | **Long EOL** |

**Go release policy**: Only the latest 2 releases receive security patches. As of June 2026, that's Go 1.25 and 1.26.

## Upgrade Path Options

### Option A: Conservative (k8s 1.30, Go 1.22)
- controller-runtime v0.18.x → k8s.io v0.30.x → Go 1.22
- Covers EKS 1.30 (extended support until July 2026)
- Minimal breaking changes
- **Problem**: Go 1.22 is already EOL

### Option B: Moderate (k8s 1.31, Go 1.22)
- controller-runtime v0.19.x → k8s.io v0.31.x → Go 1.22
- Covers EKS 1.30-1.31 well
- **Problem**: Go 1.22 is already EOL

### Option C: Recommended (k8s 1.32, Go 1.23)
- controller-runtime v0.20.x → k8s.io v0.32.x → Go 1.23
- Covers EKS 1.30-1.32 well (1.30 via backwards compat, 1.32 native)
- Go 1.23 is EOL but recently so
- **Problem**: Go 1.23 EOL since Aug 2025

### Option D: Aggressive (k8s 1.33+, Go 1.23+)
- controller-runtime v0.21+ → k8s.io v0.33+ → Go 1.23+
- Targets latest EKS standard support versions
- Higher risk of breaking API changes
- **Consideration**: AppMesh is sunsetting Sept 2026 — is this much effort justified?

## Risk Assessment

Given AppMesh sunset (September 2026), we need to balance:
1. **Security** — golang.org/x/crypto and docker updates are security-critical
2. **EKS compatibility** — controller must work on customer EKS clusters still running AppMesh
3. **Effort vs. reward** — service is sunsetting in ~3 months

## Recommended Approach

**Target: k8s.io v0.30.x (k8s 1.30) with Go 1.22+**

Rationale:
- k8s 1.30 is the oldest currently-supported EKS version (extended support until July 2026, aligning with AppMesh sunset)
- client-go backwards compat means v0.30 client works fine with 1.30-1.35 clusters
- Minimizes breaking changes vs jumping to v0.32+
- Gets us off Go 1.20 (critical for security patches in x/crypto, x/net etc)
- The Dependabot PRs for x/crypto, docker, helm can all be addressed in this upgrade

Alternatively, jumping to k8s 1.32/Go 1.23 (Option C) covers more ground but introduces more breaking changes in controller-runtime (v0.20 has significant API changes).

## Dependencies That Will Need Manual Attention

Beyond the Dependabot PRs, upgrading to controller-runtime v0.18+ will transitively require:
- Updating all k8s.io/* packages in lockstep (api, apimachinery, client-go, apiserver, apiextensions-apiserver)
- Updating Go version in go.mod and Dockerfile
- Potentially updating `golang/mock` → `go.uber.org/mock` (golang/mock archived)
- Helm v3 upgrade (already covered by PR #765)
- Removing/updating `replace` directives for docker, containerd, runc

## Sources

- EKS versions: https://docs.aws.amazon.com/eks/latest/userguide/kubernetes-versions.html
- client-go compatibility: https://github.com/kubernetes/client-go/blob/master/README.md
- controller-runtime releases: https://github.com/kubernetes-sigs/controller-runtime/releases
- controller-runtime v0.19.0 go.mod (k8s 1.31, Go 1.22)
- controller-runtime v0.20.0 go.mod (k8s 1.32, Go 1.23)
- Dependabot PRs: https://github.com/aws/aws-app-mesh-controller-for-k8s/pulls?q=is%3Apr+is%3Aopen+author%3Aapp%2Fdependabot
