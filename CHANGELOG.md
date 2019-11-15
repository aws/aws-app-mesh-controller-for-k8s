# CHANGELOG

## v0.3.0

### Summary

This version introduces [Grpc/Http2 support](https://github.com/aws/aws-app-mesh-roadmap/issues/96) and prometheus intstrumentation for App Mesh control plane metrics.

### Changes

* Attribution document ([#103](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/103), @nckturner)
* Implement Prometheus instrumentation ([#99](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/99), @stefanprodan)
* added kustomization file for remote reference ([#97](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/97), @jasonrichardsmith)
* Grpc/Http2 support ([#94](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/94), @skiyani)
* Fix code-gen to deal with GOPATH and go.mod setups. ([#90](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/90), @kiranmeduri)

## v0.2.0

### Summary

This version introduces [http header matching](https://github.com/aws/aws-app-mesh-roadmap/issues/15) and [explicit priority to routes](https://github.com/aws/aws-app-mesh-roadmap/issues/77).

### Changes

* Show CircleCI status for master branch only ([#92](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/92), @nckturner)
* Add go fmt check to CI and fix code formatting ([#89](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/89), @stefanprodan)
* Fix virtual node condition message label ([#88](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/88), @stefanprodan)
* helm, eksctl install instructions ([#84](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/84), @nckturner)
* Status badge for circleCI ([#87](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/87), @nckturner)
* Add CircleCI config ([#86](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/86), @stefanprodan)
* Expose klog flags in cobra flag set ([#85](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/85), @nckturner)
* Add enum to httpRetryEvents ([#76](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/76), @nckturner)
* Fix Prometheus scraping, enable pprof and update to go 1.13 ([#81](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/81), @stefenprodan)
* Added support to set retry-policy on App Mesh routes ([#66](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/66), @kiranmeduri)
* Use Virtual Service name for Virtual Router ([#72](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/72), @bcelenza)
* Fix documentation to highlight breaking CRD change ([#74](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/74), @kiranmeduri)
* Add logo ([#68](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/68), @nckturner)
* Disambiguate router names in example ([#67](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/67), @bcelenza)
* Add http-header match and priority to routes ([#56](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/56), @kiranmeduri)
* Using RetryOnConflict to deal with concurrent updates to resources ([#64](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/64), @kiranmeduri)
* Fix unit tests and added Makefile target ([#61](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/61), @kiranmeduri)
* Update installation instruction for aws-app-mesh-inject ([#60](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/60), @fmedery)

## v0.1.2

### Summary

This version introduces support for virtual nodes using [AWS Cloud Map](https://github.com/aws/aws-app-mesh-roadmap/issues/11) as a source for service discovery instead of DNS, as well as [TCP route support](https://github.com/aws/aws-app-mesh-roadmap/issues/4).

### Changes

* Add support for AWS Cloud Map as service-discovery ([#53](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/53), @kiranmeduri)
* Remove broken makefile target ([#59](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/59), @nckturner)
* Added TCP route support ([#46](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/46), @kiranmeduri)
* Adding logging field to virtual-node ([#45](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/45), @kiranmeduri)

## v0.1.1

### Summary

This is a patch release without major functionality changes, mostly documentation and installation improvements.

### Changes

* Rename hack/ to scripts/ ([#39](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/39), @nckturner)
* Include security disclosure statement ([#38](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/38), @vipulsabhaya)
* Update install.md ([#36](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/36), @geremyCohen)
* Updated install and example page ([#32](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/32), @jqmichael)
* Add install links for v0.1.0 ([#31](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/31), @nckturner)

## v0.1.0

### Summary

This is the initial release.  It implements a controller that watches custom resources in a Kubernetes cluster, including virtual nodes, virtual services and meshes.  It translates these into AWS App Mesh resources and creates or deletes them in via the AWS API.

### Changes

* Namespaced resource names ([#27](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/27), @nckturner)
* Simplified virtual node name users need to define in the CRDs ([#24](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/24), @jqmichael)
* Add to example and install instructions (@nckturner)
* Fix virtual nodes and services creation ([#21](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/21), @stefanprodan)
* Improve Docs ([#22](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/22), @nckturner)
* Add design and install docs and update README ([#16](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/16), @nckturner)
* Fix virtual nodes and services deletion ([#20](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/20), @stefanprodan)
* Clean Up Objects in App Mesh ([#12](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/12), @nckturner)
* Improve demo ([#9](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/9), @nckturner)
* Update routes when virtual service changes ([#7](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/7), @stefanprodan)
* Fix klog error logging before flag.Parse ([#4](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/4), @stefanprodan)
* Allow mesh reuse across namespaces ([#5](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/5), @stefanprodan)
* Virtual Node Updates ([#2](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/2), @nckturner)
* Example ([#1](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/1), @nckturner)
* Initial Controller Implementation (@nckturner)
