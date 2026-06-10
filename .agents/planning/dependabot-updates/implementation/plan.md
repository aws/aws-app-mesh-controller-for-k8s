# Implementation Plan: AppMesh Controller Dependency Upgrade

## Checklist

- [x] Step 1: Update go.mod with new dependency versions
- [x] Step 2: Migrate EventHandler interfaces
- [x] Step 3: Migrate controller Watches() API
- [x] Step 4: Fix webhook layer and admission.Decoder
- [x] Step 5: Fix test framework and custom source
- [x] Step 6: Regenerate mocks
- [x] Step 7: Fix remaining compilation errors and run unit tests
- [x] Step 8: Update Dockerfile and verify Docker build
- [ ] Step 9: Deploy to EKS and run integration tests
  - [x] Controller deploys and runs on EKS 1.31 (fargatecluster) — verified healthy
  - [x] Test script updated to support EKS (branch: fix-integ-test-eks)
  - [ ] Integration tests passing — `sidecar` suite fails on first attempt (pre-existing test issue, not upgrade-related), causing signal handler panic on retry
- [x] Step 10: Update GitHub Actions

---

## Step 1: Update go.mod with new dependency versions

**Objective:** Get go.mod to declare the correct dependency versions, even though the code won't compile yet.

**Implementation guidance:**
1. Change `go 1.20` → `go 1.23` in go.mod
2. Update direct dependencies:
   - `sigs.k8s.io/controller-runtime` → `v0.20.4` (latest patch)
   - All `k8s.io/*` packages → `v0.32.x` (must be in lockstep)
   - `golang.org/x/crypto` → latest
   - `golang.org/x/oauth2` → latest
   - `github.com/docker/docker` → `v25.0.13+incompatible`
   - `helm.sh/helm/v3` → `v3.14.3`
   - `github.com/sirupsen/logrus` → `v1.9.1`
3. Remove `replace` directives (containerd, docker/distribution, runc)
4. Run `go mod tidy` (will fail due to code issues — that's expected)
5. Manually resolve any version conflicts in go.mod

**Test requirements:** No tests expected to pass yet — this step establishes the dependency baseline.

**Integration:** Foundation for all subsequent steps.

**Demo:** `go mod download` succeeds; go.mod reflects target versions.

---

## Step 2: Migrate EventHandler interfaces

**Objective:** Update all custom EventHandler implementations to include `context.Context` in their method signatures.

**Implementation guidance:**
1. For each file matching `pkg/*/enqueue_requests_for_*_events.go`:
   - Add `"context"` to imports if not present
   - Add `ctx context.Context` as first parameter to `Create()`, `Update()`, `Delete()`, `Generic()`
2. Update corresponding test files (`*_test.go`) that call these methods directly — add `context.Background()` or `context.TODO()` as first arg
3. Files to update:
   - `pkg/virtualnode/enqueue_requests_for_mesh_events.go`
   - `pkg/virtualnode/enqueue_requests_for_backendgroup_events.go`
   - `pkg/virtualnode/enqueue_requests_for_virtualservice_events.go`
   - `pkg/virtualservice/enqueue_requests_for_mesh_events.go`
   - `pkg/virtualservice/enqueue_requests_for_virtualnode_events.go`
   - `pkg/virtualservice/enqueue_requests_for_virtualrouter_events.go`
   - `pkg/virtualrouter/enqueue_requests_for_mesh_events.go`
   - `pkg/virtualrouter/enqueue_requests_for_virtualnode_events.go`
   - `pkg/virtualgateway/enqueue_requests_for_mesh_events.go`
   - `pkg/gatewayroute/enqueue_requests_for_mesh_events.go`
   - `pkg/gatewayroute/enqueue_requests_for_virtualgateway_events.go`
   - `pkg/cloudmap/enqueue_requests_for_pod_events.go`

**Test requirements:** Tests in corresponding `*_test.go` files updated to pass new signature. Verify with `go build ./pkg/...` (may still fail due to other issues).

**Integration:** Builds on Step 1's go.mod. Required before Step 3 (controllers reference these handlers).

**Demo:** `go build ./pkg/virtualnode/...` compiles without EventHandler errors.

---

## Step 3: Migrate controller Watches() API

**Objective:** Update all controller `SetupWithManager()` methods to use the new `Watches()` / `source.Kind()` API.

**Implementation guidance:**
1. The new API for source.Kind changed significantly across v0.15→v0.20. In v0.20:
   - `source.Kind` is now a function: `source.Kind(cache, object, handler, ...predicates)`
   - `Watches()` now takes a single `source.Source` argument
2. Update pattern in each controller:
   ```go
   // Before:
   Watches(&source.Kind{Type: &appmesh.Mesh{}}, r.enqueueRequestsForMeshEvents)
   // After:
   Watches(source.Kind(mgr.GetCache(), &appmesh.Mesh{}, r.enqueueRequestsForMeshEvents))
   ```
3. For the CloudMap controller's custom notification channel source (`pkg/k8s/custom_source.go`):
   - Check if `source.Source` interface changed
   - Update `custom_source.go` to match new interface if needed
   - Update the `Watches()` call in `cloudmap_controller.go`
4. Controllers to update:
   - `controllers/appmesh/virtualnode_controller.go`
   - `controllers/appmesh/virtualservice_controller.go`
   - `controllers/appmesh/virtualrouter_controller.go`
   - `controllers/appmesh/virtualgateway_controller.go`
   - `controllers/appmesh/gatewayroute_controller.go`
   - `controllers/appmesh/cloudmap_controller.go`

**Test requirements:** Controller test files may need updates if they test SetupWithManager. Verify with `go build ./controllers/...`.

**Integration:** Depends on Step 2 (handlers must have correct signatures).

**Demo:** `go build ./controllers/...` compiles without Watches/source.Kind errors.

---

## Step 4: Fix webhook layer and admission.Decoder

**Objective:** Update the webhook handling code for controller-runtime v0.20 compatibility.

**Implementation guidance:**
1. Check `pkg/webhook/mutating_handler.go` and `validating_handler.go`:
   - If they use `admission.Decoder` as a concrete struct → update to interface usage
   - If they implement `admission.Defaulter`/`admission.Validator` → migrate to `CustomDefaulter`/`CustomValidator`
   - More likely: they use lower-level `admission.Handler` directly — verify and fix any type mismatches
2. Check `webhooks/appmesh/*.go` files for any direct use of removed types
3. Check `webhooks/core/pod_mutator.go`
4. Update webhook registration in `main.go` if needed

**Test requirements:** `go build ./webhooks/...` and `go build ./pkg/webhook/...` compile. Webhook unit tests pass.

**Integration:** Independent of Steps 2–3; can be done in parallel.

**Demo:** `go build ./webhooks/... ./pkg/webhook/...` compiles cleanly.

---

## Step 5: Fix test framework and custom source

**Objective:** Fix remaining compilation issues in test code and utility packages.

**Implementation guidance:**
1. `test/framework/framework.go` — replace `client.NewDelegatingClient(client.NewDelegatingClientInput{...})` with `client.New(cfg, client.Options{...})` or equivalent
2. `pkg/k8s/custom_source.go` — ensure `source.Source` interface is satisfied (the `Start()` method signature may have changed)
3. Fix any other compilation errors surfaced by `go build ./...`
4. Address any removed/renamed imports across the codebase

**Test requirements:** `go build ./...` succeeds (full codebase compiles).

**Integration:** Depends on Steps 2–4 being complete.

**Demo:** `go build ./...` exits with status 0.

---

## Step 6: Regenerate mocks

**Objective:** Regenerate all mock files to match updated interface signatures.

**Implementation guidance:**
1. Check `hack/gen_mocks.sh` for the mock generation command
2. Ensure `mockgen` (from `github.com/golang/mock`) is installed at a compatible version
3. Run mock generation script: `./hack/gen_mocks.sh`
4. If the script doesn't cover all mocks (some may be generated inline), find and regenerate manually:
   - `mocks/aws-app-mesh-controller-for-k8s/pkg/*/mock_*.go`
   - `mocks/apimachinery/pkg/conversion/mock_scope.go`
   - `pkg/cloudmap/mock_*.go`
   - `pkg/aws/services/mock_*.go`
5. Verify all mock files compile against new interfaces

**Test requirements:** `go build ./...` still succeeds. `go test ./...` should now pass (or get much closer).

**Integration:** Depends on Step 5 (all interfaces must be finalized first).

**Demo:** `go test ./pkg/... ./controllers/... ./webhooks/...` passes.

---

## Step 7: Fix remaining compilation errors and run unit tests

**Objective:** Achieve a fully passing `go test ./...` with zero compilation errors.

**Implementation guidance:**
1. Run `go test ./...` and fix any remaining failures:
   - Signature mismatches in test helper functions
   - Deprecated API usage flagged by new linters
   - Behavioral changes in fake client (e.g., status subresource handling from v0.15)
   - Any `workqueue.RateLimitingInterface` type changes in tests
2. Run `go vet ./...` to catch any static analysis issues
3. Verify no security vulnerabilities in final dependency tree: `go list -m all | grep -i vuln` or `govulncheck ./...`

**Test requirements:** `go test ./...` passes with zero failures. `go vet ./...` clean.

**Integration:** Depends on Step 6.

**Demo:** `go test ./...` exits 0. `go vet ./...` exits 0.

---

## Step 8: Update Dockerfile and verify Docker build

**Objective:** Ensure the container image builds successfully with Go 1.23.

**Implementation guidance:**
1. Update Dockerfile: `golang:1.20` → `golang:1.23`
2. Run `docker build --platform=linux/amd64 -t appmesh-controller:test .`
3. Verify the built image runs: `docker run --rm appmesh-controller:test --help`
4. Check image size is reasonable (shouldn't change much)

**Test requirements:** Docker build succeeds. Container starts and prints help/version.

**Integration:** Depends on Step 7 (code must compile cleanly).

**Demo:** `docker build` succeeds; `docker run` shows controller help output.

---

## Step 9: Deploy to EKS 1.32 and run e2e tests

**Objective:** Validate the upgraded controller works correctly in a real EKS environment.

**Implementation guidance:**
1. Push test image to ECR (or equivalent registry accessible by EKS cluster)
2. Deploy controller to EKS 1.32 cluster using Helm chart or kustomize
3. Run integration tests: `cd test/integration && go test ./...`
4. Run e2e tests: `cd test/e2e && go test ./...`
5. Verify key scenarios:
   - Mesh CRUD operations
   - VirtualNode/VirtualService/VirtualRouter lifecycle
   - VirtualGateway/GatewayRoute lifecycle
   - Sidecar injection (pod mutation webhook)
   - CloudMap integration
   - Validation webhooks reject invalid resources
6. Check controller logs for unexpected errors/warnings

**Test requirements:** All e2e and integration tests pass. No unexpected controller crashes or error logs.

**Integration:** Depends on Step 8 (need a working container image).

**Demo:** e2e test suite passes on EKS 1.32. Controller logs show healthy operation.

---

## Step 10: Update GitHub Actions

**Objective:** Ensure CI pipelines use the correct Go version and pass.

**Implementation guidance:**
1. Find all Go version references in `.github/workflows/*.yml`
2. Update `go-version` fields from `'1.20'` to `'1.23'`
3. Check if any actions reference specific tool versions that need updating (e.g., golangci-lint, controller-gen)
4. Verify the workflow files are syntactically valid
5. Push branch and confirm CI passes on GitHub

**Test requirements:** CI pipeline runs and passes on the PR.

**Integration:** Final step — only done after Steps 1–9 confirm everything works locally.

**Demo:** GitHub Actions CI runs green on the PR.
