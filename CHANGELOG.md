# CHANGELOG

## v0.2.0

### Summary

This version introduces [http header matching](https://github.com/aws/aws-app-mesh-roadmap/issues/15) and [explicit priority to routes](https://github.com/aws/aws-app-mesh-roadmap/issues/77).

### Changes

* Show CircleCI status for master branch only (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/92, @nckturner)
* Add go fmt check to CI and fix code formatting (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/89, @stefanprodan)
* Fix virtual node condition message label (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/88, @stefanprodan)
* helm, eksctl install instructions (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/84, @nckturner)
* Status badge for circleCI (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/87, @nckturner)
* Add CircleCI config (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/86, @stefanprodan)
* Expose klog flags in cobra flag set (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/85, @nckturner)
* Add enum to httpRetryEvents (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/76, @nckturner)
* Fix Prometheus scraping, enable pprof and update to go 1.13 (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/81, @stefenprodan)
* Added support to set retry-policy on App Mesh routes (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/66, @kiranmeduri)
* Use Virtual Service name for Virtual Router (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/72, @bcelenza)
* Fix documentation to highlight breaking CRD change (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/74, @kiranmeduri)
* Add logo (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/68, @nckturner)
* Disambiguate router names in example (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/67, @bcelenza)
* Add http-header match and priority to routes (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/56, @kiranmeduri)
* Using RetryOnConflict to deal with concurrent updates to resources (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/64, @kiranmeduri)
* Fix unit tests and added Makefile target (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/61, @kiranmeduri)
* Update installation instruction for aws-app-mesh-inject (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/60, @fmedery)

## v0.1.2

### Summary

This version introduces support for virtual nodes using [AWS Cloud Map](https://github.com/aws/aws-app-mesh-roadmap/issues/11) as a source for service discovery instead of DNS, as well as [TCP route support](https://github.com/aws/aws-app-mesh-roadmap/issues/4).

### Changes

* Add support for AWS Cloud Map as service-discovery (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/53, @kiranmeduri)
* Remove broken makefile target (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/59, @nckturner)
* Added TCP route support (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/46, @kiranmeduri)
* Adding logging field to virtual-node (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/45, @kiranmeduri)

## v0.1.1

### Summary

This is a patch release without major functionality changes, mostly documentation and installation improvements.

### Changes

* Rename hack/ to scripts/ (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/39, @nckturner)
* Include security disclosure statement (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/38, @vipulsabhaya)
* Update install.md (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/36, @geremyCohen)
* Updated install and example page (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/32, @jqmichael)
* Add install links for v0.1.0 (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/31, @nckturner)

## v0.1.0

### Summary

This is the initial release.  It implements a controller that watches custom resources in a Kubernetes cluster, including virtual nodes, virtual services and meshes.  It translates these into AWS App Mesh resources and creates or deletes them in via the AWS API.

### Changes

* Namespaced resource names (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/27, @nckturner)
* Simplified virtual node name users need to define in the CRDs (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/24, @jqmichael)
* Add to example and install instructions (@nckturner)
* Fix virtual nodes and services creation (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/21, @stefanprodan)
* Improve Docs (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/22, @nckturner)
* Add design and install docs and update README (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/16, @nckturner)
* Fix virtual nodes and services deletion (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/20, @stefanprodan)
* Clean Up Objects in App Mesh (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/12, @nckturner)
* Improve demo (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/9, @nckturner)
* Update routes when virtual service changes (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/7, @stefanprodan)
* Fix klog error logging before flag.Parse (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/4, @stefanprodan)
* Allow mesh reuse across namespaces (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/5, @stefanprodan)
* Virtual Node Updates (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/2, @nckturner)
* Example (https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/1, @nckturner)
* Initial Controller Implementation (@nckturner)
