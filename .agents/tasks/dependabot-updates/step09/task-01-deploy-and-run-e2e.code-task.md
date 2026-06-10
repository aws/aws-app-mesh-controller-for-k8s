# Task: Deploy to EKS 1.32 and Run E2E Tests

## Description
Deploy the upgraded controller to an EKS 1.32 cluster and run the integration and e2e test suites to validate correct behavior in a real environment.

## Background
The controller manages AppMesh resources on EKS clusters. After upgrading all dependencies, we need to verify that the controller correctly reconciles resources, injects sidecars, and integrates with CloudMap on a real cluster. Existing e2e infrastructure is available.

## Reference Documentation
**Required:**
- Design: `.agents/planning/dependabot-updates/design/detailed-design.md`

**Note:** You MUST read the detailed design document before beginning implementation.

## Technical Requirements
1. Push test image to an accessible container registry (ECR)
2. Deploy controller to EKS 1.32 cluster using Helm
3. Run integration tests: `cd test/integration && go test ./...`
4. Run e2e tests: `cd test/e2e && go test ./...`
5. Verify key scenarios: mesh CRUD, VirtualNode/VirtualService/VirtualRouter lifecycle, VirtualGateway/GatewayRoute lifecycle, sidecar injection, CloudMap integration, validation webhooks
6. Check controller logs for unexpected errors

## Dependencies
- Step 8 complete (working container image)
- EKS 1.32 cluster accessible
- ECR repository available

## Implementation Approach
1. Build and push image to ECR
2. Deploy via Helm: `make helm-deploy` (with appropriate env vars)
3. Wait for controller pods to be running and healthy
4. Run integration tests
5. Run e2e tests
6. Review controller logs for errors/warnings
7. Document any pre-existing test failures (separate from upgrade issues)

## Acceptance Criteria

1. **Controller Deploys Successfully**
   - Given the EKS 1.32 cluster
   - When the controller is deployed via Helm
   - Then pods are Running and Ready

2. **E2E Tests Pass**
   - Given the deployed controller
   - When `cd test/e2e && go test ./...` is run
   - Then the test suite passes

3. **No Unexpected Errors in Logs**
   - Given the running controller
   - When logs are reviewed
   - Then no unexpected crashes, panics, or error patterns appear

## Metadata
- **Complexity**: High
- **Labels**: E2E, Integration, EKS, Deployment, Verification
- **Required Skills**: EKS, Helm, ECR, e2e testing, kubectl, log analysis
