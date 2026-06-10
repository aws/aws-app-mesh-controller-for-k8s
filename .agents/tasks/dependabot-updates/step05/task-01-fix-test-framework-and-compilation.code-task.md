# Task: Fix Test Framework and Remaining Compilation Errors

## Description
Fix `test/framework/framework.go` which uses the removed `client.NewDelegatingClient`, and resolve any other remaining compilation errors across the codebase so that `make controller` succeeds on the full project.

## Background
`client.NewDelegatingClient` was removed in controller-runtime v0.15. The test framework uses it to create a client that reads from cache but writes directly. The replacement is to use the manager's client (which has this behavior built-in) or construct a client with appropriate options. Additionally, any other compilation errors from removed/renamed imports must be fixed in this step.

## Reference Documentation
**Required:**
- Design: `.agents/planning/dependabot-updates/design/detailed-design.md`

**Note:** You MUST read the detailed design document before beginning implementation.

## Technical Requirements
1. Replace `client.NewDelegatingClient(client.NewDelegatingClientInput{CacheReader: cache, Client: realClient})` with the v0.20 equivalent
2. Fix any removed/renamed imports across the codebase (e.g., `sigs.k8s.io/controller-runtime/pkg/runtime/inject`)
3. Fix any other type mismatches or removed APIs that surface during full compilation
4. Achieve `make controller` success (full codebase compiles)

## Dependencies
- Steps 2–4 complete

## Implementation Approach
1. Run `make controller` and collect all remaining compilation errors
2. Fix `test/framework/framework.go`: replace `client.NewDelegatingClient` with `client.New(restCfg, client.Options{Scheme: k8sSchema, Cache: &client.CacheOptions{Reader: cache}})` or equivalent
3. Remove any imports of `sigs.k8s.io/controller-runtime/pkg/runtime/inject` (already handled in custom_source.go but may exist elsewhere)
4. Fix any remaining type mismatches (e.g., `workqueue.RateLimitingInterface` → `workqueue.TypedRateLimitingInterface[reconcile.Request]` if needed)
5. Iterate: run `make controller`, fix errors, repeat until clean

**Primary files:**
- `test/framework/framework.go`
- Any other files surfaced by compilation errors

## Acceptance Criteria

1. **Test Framework Compiles**
   - Given `test/framework/framework.go`
   - When `make controller` is run
   - Then no `NewDelegatingClient` or related errors occur

2. **Full Build Succeeds**
   - Given the entire codebase
   - When `make controller` is run
   - Then it exits with status 0

3. **No Removed Import References**
   - Given the codebase
   - When searched for `pkg/runtime/inject`
   - Then no references exist

## Metadata
- **Complexity**: Medium
- **Labels**: Test Framework, Compilation, Migration, Cleanup
- **Required Skills**: Go compilation errors, controller-runtime client API, debugging
