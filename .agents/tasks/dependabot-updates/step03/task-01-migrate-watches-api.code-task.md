# Task: Migrate Controller Watches() API

## Description
Update all 6 controller `SetupWithManager()` methods to use the new `Watches()` / `source.Kind()` API from controller-runtime v0.20. Also update the custom `NotificationChannel` source (`pkg/k8s/custom_source.go`) to satisfy the new `source.Source` interface and fix the handler calls within it to pass context.

## Background
In controller-runtime v0.15+, `Watches()` no longer accepts `&source.Kind{Type: &X{}}` with a separate handler argument. Instead, `source.Kind` is now a function that returns a configured source: `source.Kind(cache, object, handler)`. The CloudMap controller also uses a custom `NotificationChannel` source that implements `source.Source` — its `Start()` method signature and internal handler calls need updating.

## Reference Documentation
**Required:**
- Design: `.agents/planning/dependabot-updates/design/detailed-design.md`

**Additional References (if relevant to this task):**
- `.agents/planning/dependabot-updates/research/controller-runtime-breaking-changes.md` (Watches section)

**Note:** You MUST read the detailed design document before beginning implementation. Section "3. Controller Watches() API Migration" has before/after examples.

## Technical Requirements
1. Update all `Watches(&source.Kind{Type: &X{}}, handler)` calls to `Watches(source.Kind(mgr.GetCache(), &X{}, handler))`
2. Update `pkg/k8s/custom_source.go`:
   - Remove `inject.Stoppable` implementation (inject package removed in v0.15)
   - Update `Start()` signature if the `source.Source` interface changed
   - Update internal handler calls (`handler.Create(...)`, `handler.Delete(...)`, `handler.Update(...)`) to pass context
3. Update the CloudMap controller's `Watches` call for the custom NotificationChannel source
4. Remove unused `source` import if `source.Kind` struct is no longer used (it's now a function)

## Dependencies
- Step 2 complete (EventHandler interfaces have context parameter)

## Implementation Approach
1. For each of the 5 standard controllers, replace `Watches(&source.Kind{Type: &X{}}, handler)` with `Watches(source.Kind(mgr.GetCache(), &X{}, handler))`
2. Update `pkg/k8s/custom_source.go`:
   - Remove `InjectStopChannel` method and `inject` import
   - Use context from `Start(ctx, ...)` instead of injected stop channel
   - Add `ctx context.Context` to internal handler calls
3. Update CloudMap controller's Watches call for the notification channel
4. Verify with `make controller`

**Controllers to update:**
- `controllers/appmesh/virtualnode_controller.go` (4 Watches calls)
- `controllers/appmesh/virtualservice_controller.go` (3 Watches calls)
- `controllers/appmesh/virtualrouter_controller.go` (2 Watches calls)
- `controllers/appmesh/virtualgateway_controller.go` (1 Watches call)
- `controllers/appmesh/gatewayroute_controller.go` (2 Watches calls)
- `controllers/appmesh/cloudmap_controller.go` (1 Watches call — custom source)

**Custom source:**
- `pkg/k8s/custom_source.go`

## Acceptance Criteria

1. **Standard Controllers Use New API**
   - Given each controller's `SetupWithManager()` method
   - When inspected
   - Then all `Watches` calls use `source.Kind(mgr.GetCache(), &X{}, handler)` pattern

2. **Custom Source Updated**
   - Given `pkg/k8s/custom_source.go`
   - When inspected
   - Then `inject.Stoppable` is removed, and handler calls include context

3. **CloudMap Controller Compiles**
   - Given the cloudmap controller's custom source Watches call
   - When `make controller` is run
   - Then no Watches/source errors occur

4. **Build Succeeds**
   - Given all controller changes
   - When `make controller` is run
   - Then it compiles without Watches/source.Kind errors

## Metadata
- **Complexity**: Medium
- **Labels**: Controllers, Watches, source.Kind, Migration, controller-runtime
- **Required Skills**: Go interfaces, controller-runtime Source API, controller builder pattern
