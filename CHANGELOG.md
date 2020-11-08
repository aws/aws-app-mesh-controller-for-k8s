# CHANGELOG

## v1.2.0

### Summary
This release includes support for outlier detection, circuit breakers using connection pools, support for additional Envoy config parameters, new Envoy version v1.15.0.0-prod, GitHub Actions integration for automated tests and bug fixes.

### Changes
* Update API docs with outlier detection and connection pools ([#384](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/384)  @fawadkhaliq)
* Make maxPendingRequests optional and disable preview aws-sdk-go ([#383](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/383) @fawadkhaliq)
* Add outlier detection and connection pool walkthrough links to tutorials ([#382](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/382) @fawadkhaliq)
* Add integration tests to gh actions kind setup ([#381](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/381) @fawadkhaliq)
* Build and publish the controller image in the CI ([#380](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/380) @fawadkhaliq)
* Add kinD support to run integration tests ([#378](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/378) @fawadkhaliq)
* Fix jaeger tracing collector endpoint and tracer type ([#379](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/379) @fawadkhaliq)
* Add gh actions workflow to use self-hosted runners ([#376](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/376) @fawadkhaliq)
* Add issue and feature request templates ([#375](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/375) @fawadkhaliq)
* Leave app ports empty for virtual nodes without listeners ([#373](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/373) @fawadkhaliq)
* Handle resource delete scenarios ([#367](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/367) @achevuru)
* Adjusted RBAC permissions for the controller ([#369](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/369) @fawadkhaliq)
* Add IAM policies for the controller and envoy ([#368](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/368) @fawadkhaliq)
* Add integration tests for outlier detection and connection pools ([#366](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/366) @fawadkhaliq)
* Add CRDs for connection pools in virtual node and virtual gateway ([#363](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/363) @fawadkhaliq)
* Envoy configurability in AppMesh Controller ([#365](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/365) @achevuru)
* Added docs about sidecar installation ([#364](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/364) @fawadkhaliq)
* Update Envoy image version to v1.15.1.0-prod ([#358](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/358) @landesherr)
* Fix VirtualNode's ServiceDiscovery Validation Logic ([#362](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/362) @achevuru)
* Fix for Jaeger tracing with Envoy v1.15.0 ([#359](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/359) @achevuru)
* Add CRDs for virtual node listener outlier detection ([#356](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/356) @fawadkhaliq)
* Disable CircleCI and add unit test workflow for GitHub Actions ([#355](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/355) @fawadkhaliq)
* Enable GitHub Actions with a build workflow ([#354](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/354) @fawadkhaliq)
* TLS support in e2e test suite ([#345](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/345) @achevuru)
* TLS Integration tests ([#344](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/344) @achevuru)
* CloudMap Integration tests ([#343](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/343) @achevuru)

## v1.1.1

### Summary

This release includes several minor enhancements and bug fixes. Some of the enhancements are Envoy preStop hook, expose ability to override default XRay image, readiness probe for Envoy, expose optional resource limits for sidecars and have a way to choose between default opt-in / opt-out sidecar injection mode per namespace

### Changes

- Envoy PreStop hook support ( [#312](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/312) @achevuru)
- Controller gets stuck in a crash loop ( [#314](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/314)  @achevuru)
- Expose configuration for x-ray sidecar image ( [#287](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/287)  @cmdallas)
- Update proxy route manager to v3-prod ( [#321](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/321)  @flashyang)
- Add readiness probe for envoy container ( [#325](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/325)  @fawadkhaliq)
- Provide default opt-in and opt-out options for sidecar injection ( [#338](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/338) @fawadkhaliq )
- Add optional resource limits for init and sidecar containers ( [#326](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/326)  @fawadkhaliq )
- Fix the account ID flag setup in cloud config ( [#330](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/330) @fawadkhaliq )
- Enhance Validation for VirtualNodes and VirtualRouters specs ( [#331](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/331)  @achevuru)
- Enhanced unit and integration tests ( [#322](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/322)  [#332](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/332) [#323](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/323) [#324](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/324) [#327](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/327) @fawadkhaliq)
- Add howto-k8s-tls-file-based tutorial ( [#335](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/335)  @fawadkhaliq)
- Update default Envoy sidecar image to v1.15.0 ( [#336](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/336)  @lavignes @abaptiste)
- Support to configure DNS TTL value of CloudMap services ( [#337](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/337)  @achevuru)

## v1.1.0

### Summary

This release includes new custom resources (Virtual Gateway and Gateway Routes) and new default Envoy image.

### Changes

* Custom resources added:
    * Virtual Gateway: A virtual gateway allows resources outside your mesh to communicate to resources that are inside your mesh. The virtual gateway represents an Envoy proxy running in a Kubernetes service. Unlike a virtual node, which represents a proxy running with an application, a virtual gateway represents the proxy deployed by itself ( [#249](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/249) @fawadkhaliq )
    * Gateway Route: A gateway route is used to specify the routes from Virtual Gateway to the backend Virtual Services. ( [#256](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/256) @fawadkhaliq )
* Bump Envoy image version to v1.12.4.0-prod ( [#301](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/301) @karanvasnani)
* Added the support to inject Virtual Gateway configuration to Envoys ( [#262](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/262) @fawadkhaliq)
* Updated APISpec and usage guide docs to include Virtual Gateways ([#304](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/304), [#307](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/307) @M00nF1sh @fawadkhaliq  )
* Use latest aws-sdk-go for Virtual Gateways ( [#309](https://github.com/aws/aws-app-mesh-controller-for-k8s/pull/309) @achevuru)

## v1.0.0

## Summary

This is a major release for App Mesh Kubernetes Controller. It includes changes around bug fixes from previous versions, data model changes (**backward incompatible**), scale improvements and new features

### Changes

* Custom resources supported:

    * [Mesh](https://aws.github.io/aws-app-mesh-controller-for-k8s/reference/api_spec/#appmesh.k8s.aws/v1beta2.Mesh): represents the Mesh object in AWS App Mesh. A service mesh is a logical boundary for network traffic between the services that reside within it. After you create your service mesh, you can create virtual services, virtual nodes, virtual routers, and routes to distribute traffic between the applications in your mesh.
    * [VirtualNode](https://aws.github.io/aws-app-mesh-controller-for-k8s/reference/api_spec/#appmesh.k8s.aws/v1beta2.VirtualNode): represents the Virtual Node object in AWS App Mesh. A virtual node acts as a logical pointer to a particular deployment in Kubernetes. 
    * [VirtualRouter](https://aws.github.io/aws-app-mesh-controller-for-k8s/reference/api_spec/#appmesh.k8s.aws/v1beta2.VirtualRouter): represent the Virtual Router object in AWS App Mesh and embeds App Mesh Route in it. Virtual routers handle traffic for one or more virtual services within your mesh. After you create a virtual router, you can create and associate routes for your virtual router that direct incoming requests to different virtual nodes.
    * [VirtualService](https://aws.github.io/aws-app-mesh-controller-for-k8s/reference/api_spec/#appmesh.k8s.aws/v1beta2.VirtualService): represents the Virtual Service object in AWS App Mesh. A virtual service is an abstraction of a real service that is provided by a virtual node directly or indirectly by means of a virtual router.
* App Mesh injector has been merged with the controller and there will be a single binary moving forward that provides AppMesh CRD controller and webhooks for sidecar injections
* Decoupled Kubernetes resources from AWS App Mesh resource name. Now you can use `awsName` in resource spec to denote the resource in AWS App Mesh. For example, `awsName` for VirtualNode. 
    * Note: The default generated `awsName` for VirtualNode is `${name}_${namespace}` of k8s VirtualNode resource. It's using `_` instead of `-` compared with old controller versions(<v1.0.0). If you want to **reuse existing appMesh resources in AWS created by old controllers**, you need to explicitly specify `awsName` in k8s VirtualNode resource.
    * Note: The default generated `awsName` for VirtualService is `${name}.${namespace}` of k8s VirtualService resource. Compared with old controller versions(<v1.0.0), you shouldn't specify the k8s VirtualService's name to be the DNS name anymore.  Explicitly specify `awsName` in k8s VirtualService resource if the DNS name your clients talk to didn't match this default generated `awsName`.

* Use typed references for defining relationships between resources within a Kubernetes cluster. For example, a VirtualRouter will have `VirtualNodeRef` instead of resource name
* Decoupled VirtualRouter from VirtualService. VirtualRouter will have a separate CRD that VirtualService can refer to. You can also use VirtualNode as VirtualService provider directly.
* Use namespaceSelector on Mesh to denote Mesh membership for resources within namespaces. Each individual resource no longer have meshName in spec.
* Use podSelector on VirtualNode to denote VirtualNode membership. Label selectors of two VirtualNodes within the same namespace should not overlap. Controller will reject such formation.
* Support to configure HTTP, GRPC and TCP timeouts on App Mesh virtual nodes and routes. There are two types of timeouts: per-request, which controls the amount of time that a requester will wait to complete a response, and idle, that controls the time at which the connection will be terminated if there are no active streams
* Support to use shared mesh. Shared mesh allows resources created by different accounts to communicate with each other in the same mesh.
* Additional minor changes:
    * `DNSServiceDiscovery.hostName` renamed to `DNSServiceDiscovery.hostname`
    * `ServiceDiscovery.cloudMap` renamed to `ServiceDiscovery.awsCloudMap`
    * `VirtualRouter.spec.routes.[].http` renamed to `VirtualRouter.spec.routes.[].httpRoute`(same change is done for tcp/https/grpc) 
    * `perRetryTimeoutMillis` renamed to be `perRetryTimeout` with a defined Duration struct
    * `virtualNode.spec.serviceDiscovery.awsCloudMap.attributes` from `map[string]string` to be array of `awsCloudMapAttribute`
    * used same cases for acronyms (e.g. `certificateAuthorityArns` renamed to be `certificateAuthorityARNs`)

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
