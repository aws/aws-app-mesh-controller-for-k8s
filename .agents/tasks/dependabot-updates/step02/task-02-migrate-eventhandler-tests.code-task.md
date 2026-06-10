# Task: Migrate EventHandler Tests

## Description
Update all 7 EventHandler test files to pass `context.Background()` as the first argument when directly calling handler methods (`Create`, `Update`, `Delete`, `Generic`). This makes the tests match the updated handler signatures from task-01.

## Background
After task-01 added `context.Context` to all EventHandler method signatures, the corresponding test files that call these methods directly will fail to compile. Each test invocation needs `context.Background()` added as the first argument.

## Reference Documentation
**Required:**
- Design: `.agents/planning/dependabot-updates/design/detailed-design.md`

**Note:** You MUST read the detailed design document before beginning implementation.

## Technical Requirements
1. Add `context.Background()` as the first argument to all direct calls to `Create()`, `Update()`, `Delete()`, and `Generic()` in the test files
2. Add `"context"` to imports if not already present
3. Ensure tests compile and pass (handler logic is unchanged, so test assertions remain valid)

## Dependencies
- Task-01 (migrate-eventhandler-implementations) must be completed first

## Implementation Approach
1. For each test file, find all calls to handler methods (e.g., `h.Create(e, queue)`)
2. Add `context.Background()` as the first argument (e.g., `h.Create(context.Background(), e, queue)`)
3. Add `"context"` to the import block if not present
4. Run `make test` to verify (may still fail due to other Step 3+ issues, but should not fail on signature mismatches)

**Files to update:**
- `pkg/virtualnode/enqueue_requests_for_mesh_events_test.go`
- `pkg/virtualservice/enqueue_requests_for_mesh_events_test.go`
- `pkg/virtualrouter/enqueue_requests_for_mesh_events_test.go`
- `pkg/virtualgateway/enqueue_requests_for_mesh_events_test.go`
- `pkg/gatewayroute/enqueue_requests_for_mesh_events_test.go`
- `pkg/gatewayroute/enqueue_requests_for_virtualgateway_events_test.go`
- `pkg/cloudmap/enqueue_requests_for_pod_events_test.go`

## Acceptance Criteria

1. **All Test Calls Include Context**
   - Given each test file calling handler methods
   - When inspected
   - Then all calls pass `context.Background()` as the first argument

2. **Tests Compile**
   - Given the updated test files
   - When `make test` is run
   - Then no signature mismatch errors occur for EventHandler methods

3. **Test Logic Unchanged**
   - Given the test files
   - When reviewed
   - Then no test assertions or setup logic was modified — only the context argument was added

## Metadata
- **Complexity**: Low
- **Labels**: EventHandler, Tests, Migration, Mechanical
- **Required Skills**: Go testing, context.Context
