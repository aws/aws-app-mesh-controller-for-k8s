# Task: Update Dockerfile and Verify Docker Build

## Description
Update the Dockerfile to use `golang:1.23` as the builder base image and verify the container image builds successfully with `make docker-build`.

## Background
The Dockerfile currently uses `golang:1.20` as the builder stage. With Go 1.23 now required by the dependencies, the Dockerfile must be updated. The `make docker-build` target runs `docker buildx build --platform linux/amd64`.

## Reference Documentation
**Required:**
- Design: `.agents/planning/dependabot-updates/design/detailed-design.md`

**Note:** You MUST read the detailed design document before beginning implementation.

## Technical Requirements
1. Update `Dockerfile`: `golang:1.20` → `golang:1.23`
2. Verify `make docker-build` succeeds (requires `AWS_ACCOUNT` and `AWS_REGION` env vars — may need to run docker build directly if those aren't set)
3. Verify the built image starts: `docker run --rm <image> --help`

## Dependencies
- Step 7 complete (code compiles and tests pass)

## Implementation Approach
1. Update the `FROM` line in `Dockerfile`
2. Run `docker build --platform=linux/amd64 -t appmesh-controller:test .` (or `make docker-build` if env vars are set)
3. Run `docker run --rm appmesh-controller:test --help` to verify the binary works

**Files to update:**
- `Dockerfile`

## Acceptance Criteria

1. **Dockerfile Updated**
   - Given the Dockerfile
   - When inspected
   - Then the builder stage uses `golang:1.23`

2. **Docker Build Succeeds**
   - Given the updated Dockerfile
   - When `docker build --platform=linux/amd64 -t appmesh-controller:test .` is run
   - Then it exits with status 0

3. **Container Starts**
   - Given the built image
   - When `docker run --rm appmesh-controller:test --help` is run
   - Then the controller prints usage/help output

## Metadata
- **Complexity**: Low
- **Labels**: Docker, Build, Infrastructure
- **Required Skills**: Docker, multi-stage builds
