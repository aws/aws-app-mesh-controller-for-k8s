# Requirements Clarification

## Questions and Answers

### Q1: What is the target scope — do we do all the work in a single branch/PR, or break it into multiple PRs?

**Answer:** Single PR. The changes are too intertwined to split — controller-runtime upgrade changes interfaces that mocks implement, and transitive dependency trees conflict if only partially upgraded. The intermediate state won't compile.

### Q2: What level of test verification is required before merging?

**Answer:** Full e2e on EKS. Unit tests passing + Docker image build + integration/e2e tests running on an actual EKS cluster.

### Q3: Which EKS Kubernetes version(s) should we test against?

**Answer:** Just EKS 1.32. Single version validation is sufficient given the sunsetting timeline.

### Q4: Do you have an existing EKS cluster and test infrastructure for e2e, or does that need to be set up?

**Answer:** Existing infra is available. No need to provision test clusters as part of this work.

### Q5: Should we also update the AWS SDK (v1.44.252) while we're in here, or leave it alone?

**Answer:** Leave it alone. It works, no security concern, and minimizes risk for a sunsetting service.

### Q6: The codebase uses `github.com/golang/mock` (archived) across 43 files. Should we migrate to `go.uber.org/mock` (the maintained fork), or just pin the current version and leave it?

**Answer:** Pin golang/mock at current version. Less churn for a sunsetting service. Mocks still need regeneration due to interface changes, but keep the same tool/import path.

### Q7: Are there any CI/CD pipelines (GitHub Actions, etc.) that need updating for the new Go version, or is that out of scope?

**Answer:** In scope, but sequenced last. Get the build fully working locally (compile, unit tests, e2e) before updating GitHub Actions. The CI changes should be the final piece of the PR.

### Q8: Do you want to close the existing Dependabot PRs after this work lands, or leave them for GitHub to auto-close when the deps are already up-to-date on mainline?

**Answer:** Doesn't matter. Dependabot will likely auto-close them. Not a concern.

### Q9: For the `replace` directives in go.mod (containerd, docker/distribution, opencontainers/runc) — these were pinned for security. Should we remove them if the newer direct dependencies are already at safe versions, or keep them as defensive measures?

**Answer:** Remove them if the newer dependency versions resolve safely without them. Stale `replace` directives can block getting newer safe versions. Verify with `go mod tidy` and check for known vulnerabilities after removal.

