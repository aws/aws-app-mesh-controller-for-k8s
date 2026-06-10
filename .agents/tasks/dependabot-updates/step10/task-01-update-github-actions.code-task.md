# Task: Update GitHub Actions

## Description
Update all GitHub Actions workflow files to use Go 1.23 and verify any tool version references (golangci-lint, controller-gen) are compatible.

## Background
The CI workflows currently reference `go-version: '1.20.*'`. After confirming everything works locally (Steps 1–9), this final step updates CI so the PR passes GitHub Actions checks.

## Reference Documentation
**Required:**
- Design: `.agents/planning/dependabot-updates/design/detailed-design.md`

**Note:** You MUST read the detailed design document before beginning implementation.

## Technical Requirements
1. Update `go-version` in all workflow files from `'1.20.*'` to `'1.23'`
2. Check if any actions reference specific tool versions that need updating (golangci-lint, controller-gen, etc.)
3. Verify workflow YAML is syntactically valid
4. Check the `release.yaml` workflow's `DEFAULT_GO_VERSION` env var

## Dependencies
- Steps 1–9 complete (everything works locally)

## Implementation Approach
1. Update `.github/workflows/unit-test.yml`: `go-version: '1.23'`
2. Update `.github/workflows/build.yml`: `go-version: '1.23'`
3. Update `.github/workflows/release.yaml`: `DEFAULT_GO_VERSION` env var to `'1.23'`
4. Check for any other Go version references in workflow files
5. Validate YAML syntax

**Files to update:**
- `.github/workflows/unit-test.yml`
- `.github/workflows/build.yml`
- `.github/workflows/release.yaml`

## Acceptance Criteria

1. **All Workflows Updated**
   - Given all workflow files in `.github/workflows/`
   - When inspected for Go version references
   - Then all reference Go 1.23

2. **Valid YAML**
   - Given the workflow files
   - When validated
   - Then no syntax errors exist

3. **CI Passes**
   - Given the PR is pushed
   - When GitHub Actions runs
   - Then all CI checks pass green

## Metadata
- **Complexity**: Low
- **Labels**: CI, GitHub Actions, Infrastructure
- **Required Skills**: GitHub Actions YAML, CI/CD
