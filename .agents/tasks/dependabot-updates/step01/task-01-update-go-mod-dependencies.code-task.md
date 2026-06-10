# Task: Update go.mod with New Dependency Versions

## Description
Update go.mod to declare Go 1.23, bump controller-runtime to v0.20.4, all k8s.io packages to v0.32.x, and address the 6 pending Dependabot PRs (x/crypto, x/oauth2, docker, helm, logrus). Remove stale `replace` directives and resolve the dependency graph. The code won't compile after this step — the goal is establishing the correct dependency baseline.

## Background
The aws-app-mesh-controller-for-k8s currently uses Go 1.20, controller-runtime v0.14.6, and k8s.io v0.26.x — all significantly outdated. There are 6 open Dependabot PRs for security-critical dependencies. This is the foundation step for a single atomic PR upgrading everything at once; subsequent steps will fix compilation errors introduced by the API changes in newer controller-runtime versions.

## Reference Documentation
**Required:**
- Design: `.agents/planning/dependabot-updates/design/detailed-design.md`

**Additional References (if relevant to this task):**
- `.agents/planning/dependabot-updates/research/dependency-compatibility.md` (target versions and compatibility matrix)

**Note:** You MUST read the detailed design document before beginning implementation. The dependency-compatibility research contains the exact version targets.

## Technical Requirements
1. Update `go` directive from `1.20` to `1.23`
2. Update `sigs.k8s.io/controller-runtime` from `v0.14.6` to `v0.20.4`
3. Update all `k8s.io/*` packages to `v0.32.x` (must be in lockstep)
4. Update `golang.org/x/crypto` to latest (≥v0.45.0)
5. Update `golang.org/x/oauth2` to latest (≥v0.27.0)
6. Update `github.com/docker/docker` to `v25.0.13+incompatible`
7. Update `helm.sh/helm/v3` to `v3.14.3`
8. Update `github.com/sirupsen/logrus` to `v1.9.1`
9. Remove `replace` directives for containerd, docker/distribution, and runc
10. Run `go mod tidy` (expected to fail due to code incompatibilities — that's OK)
11. Manually resolve any version conflicts so `go mod download` succeeds
12. Leave AWS SDK v1 at v1.44.252 (no update)

## Dependencies
- Go 1.23 must be installed locally (or use `go install golang.org/dl/go1.23.0` + `go1.23.0 download`)
- Network access to download modules from proxy.golang.org

## Implementation Approach
1. Read current `go.mod` to understand existing dependency versions and replace directives
2. Update the `go` directive to `1.23`
3. Update direct dependencies to target versions in go.mod
4. Remove the stale `replace` directives (containerd, docker/distribution, runc)
5. Run `go mod tidy` — it will report errors about source incompatibilities; that's expected
6. If `go mod tidy` fails completely, manually edit go.sum or use `go mod download` to pull versions
7. Resolve any transitive dependency conflicts (may need temporary `replace` directives for Helm)
8. Verify `go mod download` exits 0

## Acceptance Criteria

1. **Go Version Updated**
   - Given the go.mod file
   - When inspected
   - Then the `go` directive reads `1.23`

2. **Controller-Runtime at Target Version**
   - Given the go.mod file
   - When inspected
   - Then `sigs.k8s.io/controller-runtime` is at `v0.20.4`

3. **K8s Libraries in Lockstep**
   - Given the go.mod file
   - When all `k8s.io/*` dependencies are inspected
   - Then they are all at `v0.32.x` versions

4. **Security Dependencies Updated**
   - Given the go.mod file
   - When `golang.org/x/crypto`, `golang.org/x/oauth2`, and `github.com/docker/docker` are inspected
   - Then they are at their target versions (crypto ≥v0.45.0, oauth2 ≥v0.27.0, docker v25.0.13)

5. **Replace Directives Removed**
   - Given the go.mod file
   - When checked for replace directives
   - Then no replace directives exist for containerd, docker/distribution, or runc

6. **Module Download Succeeds**
   - Given the updated go.mod
   - When `go mod download` is executed
   - Then it exits with status 0 (all modules are resolvable)

7. **AWS SDK Unchanged**
   - Given the go.mod file
   - When `github.com/aws/aws-sdk-go` is inspected
   - Then it remains at `v1.44.252`

## Metadata
- **Complexity**: Medium
- **Labels**: Dependencies, go.mod, Security, Foundation
- **Required Skills**: Go modules, dependency resolution, Kubernetes ecosystem versioning
