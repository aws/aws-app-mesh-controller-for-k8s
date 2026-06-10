# Task: Migrate EventHandler Implementations

## Description
Add `context.Context` as the first parameter to all custom EventHandler method signatures (`Create`, `Update`, `Delete`, `Generic`) across 12 handler files. This is required by controller-runtime v0.15+ where `handler.EventHandler` interface methods now require a context parameter.

## Background
In controller-runtime v0.15, the `handler.EventHandler` interface changed to require `context.Context` as the first parameter on all methods. The codebase has 12 custom event handler files implementing this interface. The change is mechanical — add the parameter, and where methods already use `context.Background()` internally (e.g., in `Update` methods that call helper functions), replace that with the passed-in `ctx`.

## Reference Documentation
**Required:**
- Design: `.agents/planning/dependabot-updates/design/detailed-design.md`

**Additional References (if relevant to this task):**
- `.agents/planning/dependabot-updates/research/controller-runtime-breaking-changes.md` (EventHandler section)

**Note:** You MUST read the detailed design document before beginning implementation. Section "2. EventHandler Interface Migration" has before/after examples.

## Technical Requirements
1. Add `ctx context.Context` as the first parameter to `Create()`, `Update()`, `Delete()`, and `Generic()` on all 12 handler files
2. Add `"context"` to imports if not already present
3. Replace any internal `context.Background()` calls with the passed-in `ctx` parameter
4. Ensure the `var _ handler.EventHandler = (*structName)(nil)` interface assertions still compile

## Dependencies
- Step 1 complete (go.mod updated to controller-runtime v0.20.4)

## Implementation Approach
1. For each of the 12 files listed below, update all four method signatures to include `ctx context.Context` as the first parameter
2. Where methods already create `context.Background()` internally (e.g., the `Update` handlers that call helper functions like `enqueueVirtualNodesForMesh`), pass `ctx` instead
3. Verify with `make controller` (may still have other errors, but EventHandler signatures should be satisfied)

**Files to update:**
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

## Acceptance Criteria

1. **All Handlers Have Context Parameter**
   - Given each of the 12 handler files
   - When inspected
   - Then all `Create`, `Update`, `Delete`, and `Generic` methods have `ctx context.Context` as their first parameter

2. **Context Propagated Internally**
   - Given handler methods that previously called `context.Background()`
   - When updated
   - Then they use the passed-in `ctx` instead

3. **Interface Assertion Compiles**
   - Given each handler struct with `var _ handler.EventHandler = (*structName)(nil)`
   - When `make controller` is run
   - Then the interface assertion compiles without errors

4. **No Regressions in Handler Logic**
   - Given the handler implementations
   - When reviewed
   - Then no business logic was altered — only the context parameter was added

## Metadata
- **Complexity**: Low
- **Labels**: EventHandler, Migration, Mechanical, controller-runtime
- **Required Skills**: Go interfaces, context.Context pattern
