# Research: controller-runtime Breaking Changes (v0.14 → v0.20)

## Summary of Impact

Upgrading from v0.14.6 to v0.20.x spans **6 major releases**, each with breaking changes. The biggest impact is in **v0.15** (massive release with many API changes). Subsequent releases are incrementally smaller.

## Release-by-Release Breaking Changes

### v0.15.0 (k8s 1.27) — ⚠️ LARGEST IMPACT

This is described as "probably the largest release in the history of the project."

| Change | Impact on This Codebase | Effort |
|--------|------------------------|--------|
| **EventHandler requires `context.Context`** | 🔴 HIGH — All custom event handlers (`enqueue_requests_for_*_events.go`) need `context.Context` added to `Create()`, `Update()`, `Delete()`, `Generic()` signatures | ~12 files |
| **`builder.Watches` signature changed** — takes `client.Object` instead of `source.Source` | 🔴 HIGH — All `Watches(&source.Kind{Type: &X{}}, handler)` calls must change to new syntax | 6 controllers |
| **`pkg/inject` removed** (dependency injection) | 🟢 LOW — The codebase has its own `pkg/inject` package (sidecar injection), NOT the controller-runtime one | No impact |
| **`client.NewDelegatingClient` removed** | 🟡 MED — Used in `test/framework/framework.go` | 1 file |
| **`webhook.Server` became interface + `webhook.NewServer`** | 🟡 MED — Need to check main.go webhook setup | 1 file |
| **`Validator`/`CustomValidator` interfaces return warnings** | 🟡 MED — Webhook validators may need signature updates | ~6 webhook files |
| **`client.Client` interface requires new methods** (`GroupVersionKindFor`, `IsObjectNamespaced`) | 🟢 LOW — Only matters if we implement custom clients (we don't) | No impact |
| **Removed deprecated `cache.ObjectAll`, `MultiNamespacedCacheBuilder`** | 🟢 LOW — Not used in codebase | No impact |

### v0.16.0 (k8s 1.28)

| Change | Impact on This Codebase | Effort |
|--------|------------------------|--------|
| Removed deprecated manager/webhook/cluster options | 🟢 LOW — Already removed in v0.15 migration | Minimal |
| Granular cache configuration API | 🟢 LOW — Only impacts custom cache config | No impact |
| Secure metrics serving | 🟢 LOW — Optional feature | No impact |

### v0.17.0 (k8s 1.29)

| Change | Impact on This Codebase | Effort |
|--------|------------------------|--------|
| `admission.Validator` and `admission.Defaulter` deprecated (removed in v0.20) | 🟡 MED — Need to check if webhook handlers use these | Check needed |
| Fake client: Only set TypeMeta for unstructured | 🟢 LOW — Test-only impact | Minimal |
| Removed `apiutil.NewDiscoveryRESTMapper` | 🟢 LOW — Not used | No impact |

### v0.18.0 (k8s 1.30)

| Change | Impact on This Codebase | Effort |
|--------|------------------------|--------|
| **Source, Event, Predicate, Handler: Generics support** | 🟡 MED — Event handler types may need updating (but old interfaces should still work) | Review needed |
| `admission.Decoder` is now an interface | 🟡 MED — If webhooks use the decoder directly | ~2 files |
| `ControllerManagerConfiguration` removed | 🟢 LOW — Not used | No impact |

### v0.19.0 (k8s 1.31)

| Change | Impact on This Codebase | Effort |
|--------|------------------------|--------|
| `client.WarningHandler` options removed | 🟢 LOW — Not used | No impact |
| Controller names must be unique (validation added) | 🟢 LOW — Names derived from type, already unique | No impact |
| Generic `TypedReconciler` added | 🟢 LOW — New feature, not breaking for old code | No impact |

### v0.20.0 (k8s 1.32) — OUR TARGET

| Change | Impact on This Codebase | Effort |
|--------|------------------------|--------|
| **`admission.Defaulter` and `admission.Validator` REMOVED** | 🔴 HIGH — Must migrate to `CustomDefaulter`/`CustomValidator` | ~12 webhook files |
| Deprecated `SyncPeriod` cluster option removed | 🟢 LOW — Not used directly | No impact |
| Webhooks stop deleting unknown fields in CustomDefaulter | 🟢 LOW — Behavioral change, not API | No impact |

## Codebase-Specific Impact Assessment

### High-Impact Changes (must fix to compile)

1. **EventHandler signature change (v0.15)** — 12+ files in `pkg/*/enqueue_requests_for_*_events.go`
   - Old: `Create(e event.CreateEvent, queue workqueue.RateLimitingInterface)`
   - New: `Create(ctx context.Context, e event.CreateEvent, queue workqueue.RateLimitingInterface)`

2. **Watches() signature change (v0.15)** — 6 controllers
   - Old: `Watches(&source.Kind{Type: &appmesh.Mesh{}}, handler)`
   - New: `Watches(source.Kind(..., &appmesh.Mesh{}), handler)` (or similar new API)

3. **admission.Defaulter/Validator removal (v0.20)** — 12 webhook files in `webhooks/appmesh/`
   - Must migrate to `CustomDefaulter`/`CustomValidator` pattern
   - Note: The codebase has its own `pkg/webhook` abstraction — need to check if it wraps the controller-runtime interfaces

4. **client.NewDelegatingClient removal (v0.15)** — 1 test file
   - `test/framework/framework.go` uses this directly

### Medium-Impact Changes

5. **golang/mock → go.uber.org/mock migration** — 43 files using `github.com/golang/mock`
   - `golang/mock` is archived; must switch to `go.uber.org/mock`
   - All mocks need regeneration

6. **workqueue.RateLimitingInterface changes** — EventHandler signatures reference this
   - The new event handler interface uses `workqueue.TypedRateLimitingInterface[reconcile.Request]`

### Low-Impact / No-Impact

- The codebase's own `pkg/inject` is for sidecar injection (not controller-runtime's removed DI package)
- No use of `MultiNamespacedCacheBuilder`, `ControllerManagerConfiguration`, or `NewDiscoveryRESTMapper`
- Controllers use standard `For()` / `Complete()` pattern — no custom client implementations

## Recommended Migration Strategy

Given the volume of changes, a **staged approach** is safest:

1. **Stage 1**: Update Go version, fix go.mod, update non-breaking deps (x/crypto, logrus, etc.)
2. **Stage 2**: Update controller-runtime to v0.20 and fix all compilation errors
   - EventHandler signature updates (~12 files, mechanical)
   - Watches() API migration (~6 controllers)
   - Webhook migration to CustomDefaulter/CustomValidator
   - client.NewDelegatingClient → alternative in test framework
3. **Stage 3**: golang/mock → go.uber.org/mock + regenerate mocks
4. **Stage 4**: Update remaining deps (docker, helm) and clean up `replace` directives
5. **Stage 5**: Build, test, verify

## Risk Assessment

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Event handler changes introduce bugs | Low | Changes are mechanical (add ctx param) |
| Webhook migration breaks validation | Medium | Custom webhook layer may need redesign |
| Mock regeneration breaks tests | Low | Mechanical regeneration |
| Helm upgrade introduces incompatibilities | Low | Helm v3 API is stable |
| Docker library upgrade breaks build | Low | Only used indirectly through helm |

## Sources

- https://github.com/kubernetes-sigs/controller-runtime/releases/tag/v0.15.0
- https://github.com/kubernetes-sigs/controller-runtime/releases/tag/v0.16.0
- https://github.com/kubernetes-sigs/controller-runtime/releases/tag/v0.17.0
- https://github.com/kubernetes-sigs/controller-runtime/releases/tag/v0.18.0
- https://github.com/kubernetes-sigs/controller-runtime/releases/tag/v0.19.0
- https://github.com/kubernetes-sigs/controller-runtime/releases/tag/v0.20.0
