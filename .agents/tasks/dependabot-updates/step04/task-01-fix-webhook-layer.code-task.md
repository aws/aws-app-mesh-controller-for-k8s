# Task: Fix Webhook Layer and admission.Decoder

## Description
Update `pkg/webhook/mutating_handler.go` and `pkg/webhook/validating_handler.go` to remove the deprecated `admission.DecoderInjector` interface and `*admission.Decoder` field pattern. In controller-runtime v0.20, `admission.Decoder` is now an interface and injection is no longer used — the decoder should be obtained from the admission request or constructed directly.

## Background
The codebase uses a custom webhook abstraction in `pkg/webhook/` that wraps controller-runtime's admission handling. These handlers implement `admission.DecoderInjector` and store `*admission.Decoder` — both patterns removed in v0.20. The `admission.Decoder` type changed from a concrete struct to an interface in v0.18, and `DecoderInjector` was removed in v0.20.

## Reference Documentation
**Required:**
- Design: `.agents/planning/dependabot-updates/design/detailed-design.md`

**Additional References (if relevant to this task):**
- `.agents/planning/dependabot-updates/research/controller-runtime-breaking-changes.md` (webhook sections in v0.17, v0.18, v0.20)

**Note:** You MUST read the detailed design document before beginning implementation.

## Technical Requirements
1. Remove `admission.DecoderInjector` interface assertions and `InjectDecoder` methods
2. Replace `*admission.Decoder` fields with decoder obtained via `admission.NewDecoder(scheme)`
3. Update handler constructors to accept a `runtime.Scheme` and create the decoder internally
4. Update all callers (webhook registration in `main.go` or wherever handlers are created)
5. Update corresponding test files (`mutating_handler_test.go`, `validating_handler_test.go`)

## Dependencies
- Step 1 complete (controller-runtime v0.20 in go.mod)
- Independent of Steps 2–3

## Implementation Approach
1. Read `pkg/webhook/mutating_handler.go` and `validating_handler.go` fully to understand current pattern
2. Remove `DecoderInjector` interface assertion and `InjectDecoder` method from both
3. Add `scheme *runtime.Scheme` to constructor or create decoder in handler initialization
4. Use `admission.NewDecoder(scheme)` to get the decoder
5. Update tests to construct handlers with the new pattern
6. Verify with `make controller`

**Files to update:**
- `pkg/webhook/mutating_handler.go`
- `pkg/webhook/validating_handler.go`
- `pkg/webhook/mutating_handler_test.go`
- `pkg/webhook/validating_handler_test.go`
- Any file that constructs these handlers (likely `main.go` or a webhook setup function)

## Acceptance Criteria

1. **DecoderInjector Removed**
   - Given both webhook handler files
   - When inspected
   - Then no references to `admission.DecoderInjector` or `InjectDecoder` exist

2. **Decoder Properly Constructed**
   - Given the handler initialization
   - When inspected
   - Then decoder is created via `admission.NewDecoder(scheme)` or equivalent v0.20 API

3. **Webhook Build Succeeds**
   - Given all webhook changes
   - When `make controller` is run
   - Then no admission/decoder compilation errors occur

4. **Tests Updated and Pass**
   - Given the webhook test files
   - When `make test` is run
   - Then webhook handler tests compile and pass

## Metadata
- **Complexity**: Medium
- **Labels**: Webhooks, admission.Decoder, Migration, controller-runtime
- **Required Skills**: Go interfaces, controller-runtime admission API, webhook patterns
