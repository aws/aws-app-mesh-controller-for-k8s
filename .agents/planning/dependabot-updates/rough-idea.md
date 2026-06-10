# Rough Idea: Address AppMesh Controller Dependabot Updates

## Source
GitHub repo: https://github.com/aws/aws-app-mesh-controller-for-k8s
Local checkout: ~/workplace/appmesh-controller/aws-app-mesh-controller-for-k8s

## Problem
The repository has many pending Dependabot PRs that need to be merged. These dependency updates need to be compatible with currently available EKS Kubernetes versions.

## Scope
1. Review pending Dependabot PRs on GitHub
2. Verify compatibility of proposed dependency versions with current EKS Kubernetes versions
3. Update Go libraries accordingly
4. Update Go version if needed for compatibility
5. Ensure the controller builds and tests pass with updated dependencies

## Current State (as of 2026-06-02)
- Go version: 1.20
- Kubernetes libraries: v0.26.x (k8s.io/api, k8s.io/apimachinery, k8s.io/client-go)
- controller-runtime: v0.14.6
- AWS SDK: v1.44.252
- Helm: v3.11.3
- Docker base image: golang:1.20
