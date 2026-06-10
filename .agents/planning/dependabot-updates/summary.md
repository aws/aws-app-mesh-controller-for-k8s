# Project Summary: AppMesh Controller Dependency Upgrade

## Artifacts Created

| File | Purpose |
|------|---------|
| `.agents/planning/dependabot-updates/rough-idea.md` | Initial concept and current state |
| `.agents/planning/dependabot-updates/idea-honing.md` | Requirements Q&A (9 questions) |
| `.agents/planning/dependabot-updates/research/dependency-compatibility.md` | Dependency versions, EKS compat, upgrade options |
| `.agents/planning/dependabot-updates/research/eks-versions.md` | EKS supported K8s versions (June 2026) |
| `.agents/planning/dependabot-updates/research/controller-runtime-breaking-changes.md` | Breaking changes v0.14→v0.20, impact on codebase |
| `.agents/planning/dependabot-updates/design/detailed-design.md` | Full design document |
| `.agents/planning/dependabot-updates/implementation/plan.md` | 10-step implementation plan with checklist |
| `.agents/planning/dependabot-updates/summary.md` | This file |

## Design Summary

**Goal:** Single atomic PR upgrading Go 1.20→1.23, controller-runtime v0.14→v0.20, k8s.io v0.26→v0.32, plus addressing all 6 Dependabot PRs.

**Key changes:**
- ~12 EventHandler files: add `context.Context` parameter
- 6 controllers: migrate `Watches()` API
- ~43 mock files: regenerate
- Dockerfile + CI: update Go version
- go.mod: major dependency version bumps, remove stale `replace` directives

**Verification:** Unit tests + Docker build + full e2e on EKS 1.32.

## Implementation Approach

10 incremental steps, each building toward a compilable/testable state:

1. Update go.mod (dependency baseline)
2. Migrate EventHandler interfaces (mechanical — add ctx param)
3. Migrate Watches() API (6 controllers)
4. Fix webhook layer
5. Fix test framework + custom source
6. Regenerate mocks
7. Fix remaining issues, pass unit tests
8. Docker build
9. E2e on EKS 1.32
10. Update GitHub Actions (last)

## Next Steps

1. Review the implementation plan at `implementation/plan.md`
2. Begin implementation starting at Step 1
3. Work through each step, checking off the checklist as you go

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| controller-runtime v0.20 Watches API differs from documented | Check actual source at upgrade time; use IDE autocomplete |
| Helm v3.14 transitive deps conflict | May need additional replace directives temporarily |
| Mock regeneration produces different output | Review generated diffs; mocks are purely test code |
| e2e tests have bitrotted independently | Fix any pre-existing test failures separately |
