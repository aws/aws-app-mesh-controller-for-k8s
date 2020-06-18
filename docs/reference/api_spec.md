<p>Packages:</p>
<ul>
<li>
<a href="#appmesh.k8s.aws%2fv1beta2">appmesh.k8s.aws/v1beta2</a>
</li>
</ul>
<h2 id="appmesh.k8s.aws/v1beta2">appmesh.k8s.aws/v1beta2</h2>
Resource Types:
<ul><li>
<a href="#appmesh.k8s.aws/v1beta2.Mesh">Mesh</a>
</li><li>
<a href="#appmesh.k8s.aws/v1beta2.VirtualNode">VirtualNode</a>
</li><li>
<a href="#appmesh.k8s.aws/v1beta2.VirtualRouter">VirtualRouter</a>
</li><li>
<a href="#appmesh.k8s.aws/v1beta2.VirtualService">VirtualService</a>
</li></ul>
<h3 id="appmesh.k8s.aws/v1beta2.Mesh">Mesh
</h3>
<p>
<p>Mesh is the Schema for the meshes API</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
appmesh.k8s.aws/v1beta2
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>Mesh</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.16/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.MeshSpec">
MeshSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>awsName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>AWSName is the AppMesh Mesh object&rsquo;s name.
If unspecified or empty, it defaults to be &ldquo;${name}&rdquo; of k8s Mesh</p>
</td>
</tr>
<tr>
<td>
<code>namespaceSelector</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.16/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>NamespaceSelector selects Namespaces using labels to designate mesh membership.
This field follows standard label selector semantics:
if present but empty, it selects all namespaces.
if absent, it selects no namespace.</p>
</td>
</tr>
<tr>
<td>
<code>egressFilter</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.EgressFilter">
EgressFilter
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The egress filter rules for the service mesh.
If unspecified, default settings from AWS API will be applied. Refer to AWS Docs for default settings.</p>
</td>
</tr>
<tr>
<td>
<code>meshOwner</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The AWS IAM account ID of the service mesh owner.
Required if the account ID is not your own.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.MeshStatus">
MeshStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.VirtualNode">VirtualNode
</h3>
<p>
<p>VirtualNode is the Schema for the virtualnodes API</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
appmesh.k8s.aws/v1beta2
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>VirtualNode</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.16/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualNodeSpec">
VirtualNodeSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>awsName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>AWSName is the AppMesh VirtualNode object&rsquo;s name.
If unspecified or empty, it defaults to be &ldquo;${name}_${namespace}&rdquo; of k8s VirtualNode</p>
</td>
</tr>
<tr>
<td>
<code>podSelector</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.16/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodSelector selects Pods using labels to designate VirtualNode membership.
This field follows standard label selector semantics:
if present but empty, it selects all pods within namespace.
if absent, it selects no pod.</p>
</td>
</tr>
<tr>
<td>
<code>listeners</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.Listener">
[]Listener
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The listener that the virtual node is expected to receive inbound traffic from</p>
</td>
</tr>
<tr>
<td>
<code>serviceDiscovery</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.ServiceDiscovery">
ServiceDiscovery
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The service discovery information for the virtual node.</p>
</td>
</tr>
<tr>
<td>
<code>backends</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.Backend">
[]Backend
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The backends that the virtual node is expected to send outbound traffic to.</p>
</td>
</tr>
<tr>
<td>
<code>backendDefaults</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.BackendDefaults">
BackendDefaults
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>A reference to an object that represents the defaults for backends.</p>
</td>
</tr>
<tr>
<td>
<code>logging</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.Logging">
Logging
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The inbound and outbound access logging information for the virtual node.</p>
</td>
</tr>
<tr>
<td>
<code>meshRef</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.MeshReference">
MeshReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>A reference to k8s Mesh CR that this VirtualNode belongs to.
The admission controller populates it using Meshes&rsquo;s selector, and prevents users from setting this field.</p>
<p>Populated by the system.
Read-only.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualNodeStatus">
VirtualNodeStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.VirtualRouter">VirtualRouter
</h3>
<p>
<p>VirtualRouter is the Schema for the virtualrouters API</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
appmesh.k8s.aws/v1beta2
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>VirtualRouter</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.16/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualRouterSpec">
VirtualRouterSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>awsName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>AWSName is the AppMesh VirtualRouter object&rsquo;s name.
If unspecified or empty, it defaults to be &ldquo;${name}_${namespace}&rdquo; of k8s VirtualRouter</p>
</td>
</tr>
<tr>
<td>
<code>listeners</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualRouterListener">
[]VirtualRouterListener
</a>
</em>
</td>
<td>
<p>The listeners that the virtual router is expected to receive inbound traffic from</p>
</td>
</tr>
<tr>
<td>
<code>routes</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.Route">
[]Route
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The routes associated with VirtualRouter</p>
</td>
</tr>
<tr>
<td>
<code>meshRef</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.MeshReference">
MeshReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>A reference to k8s Mesh CR that this VirtualRouter belongs to.
The admission controller populates it using Meshes&rsquo;s selector, and prevents users from setting this field.</p>
<p>Populated by the system.
Read-only.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualRouterStatus">
VirtualRouterStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.VirtualService">VirtualService
</h3>
<p>
<p>VirtualService is the Schema for the virtualservices API</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
appmesh.k8s.aws/v1beta2
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>VirtualService</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.16/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualServiceSpec">
VirtualServiceSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>awsName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>AWSName is the AppMesh VirtualService object&rsquo;s name.
If unspecified or empty, it defaults to be &ldquo;${name}.${namespace}&rdquo; of k8s VirtualService</p>
</td>
</tr>
<tr>
<td>
<code>provider</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualServiceProvider">
VirtualServiceProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The provider for virtual services. You can specify a single virtual node or virtual router.</p>
</td>
</tr>
<tr>
<td>
<code>meshRef</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.MeshReference">
MeshReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>A reference to k8s Mesh CR that this VirtualService belongs to.
The admission controller populates it using Meshes&rsquo;s selector, and prevents users from setting this field.</p>
<p>Populated by the system.
Read-only.</p>
</td>
</tr>
</table>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualServiceStatus">
VirtualServiceStatus
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.AWSCloudMapInstanceAttribute">AWSCloudMapInstanceAttribute
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.AWSCloudMapServiceDiscovery">AWSCloudMapServiceDiscovery</a>)
</p>
<p>
<p>AWSCloudMapInstanceAttribute refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_AwsCloudMapInstanceAttribute.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_AwsCloudMapInstanceAttribute.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>key</code></br>
<em>
string
</em>
</td>
<td>
<p>The name of an AWS Cloud Map service instance attribute key.</p>
</td>
</tr>
<tr>
<td>
<code>value</code></br>
<em>
string
</em>
</td>
<td>
<p>The value of an AWS Cloud Map service instance attribute key.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.AWSCloudMapServiceDiscovery">AWSCloudMapServiceDiscovery
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.ServiceDiscovery">ServiceDiscovery</a>)
</p>
<p>
<p>AWSCloudMapServiceDiscovery refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_AwsCloudMapServiceDiscovery.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_AwsCloudMapServiceDiscovery.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>namespaceName</code></br>
<em>
string
</em>
</td>
<td>
<p>The name of the AWS Cloud Map namespace to use.</p>
</td>
</tr>
<tr>
<td>
<code>serviceName</code></br>
<em>
string
</em>
</td>
<td>
<p>The name of the AWS Cloud Map service to use.</p>
</td>
</tr>
<tr>
<td>
<code>attributes</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.AWSCloudMapInstanceAttribute">
[]AWSCloudMapInstanceAttribute
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>A string map that contains attributes with values that you can use to filter instances by any custom attribute that you specified when you registered the instance</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.AccessLog">AccessLog
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.Logging">Logging</a>)
</p>
<p>
<p>AccessLog refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_AccessLog.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_AccessLog.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>file</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.FileAccessLog">
FileAccessLog
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The file object to send virtual node access logs to.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.Backend">Backend
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualNodeSpec">VirtualNodeSpec</a>)
</p>
<p>
<p>Backend refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_Backend.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_Backend.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>virtualService</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualServiceBackend">
VirtualServiceBackend
</a>
</em>
</td>
<td>
<p>Specifies a virtual service to use as a backend for a virtual node.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.BackendDefaults">BackendDefaults
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualNodeSpec">VirtualNodeSpec</a>)
</p>
<p>
<p>BackendDefaults refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_BackendDefaults.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_BackendDefaults.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>clientPolicy</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.ClientPolicy">
ClientPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>A reference to an object that represents a client policy.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.ClientPolicy">ClientPolicy
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.BackendDefaults">BackendDefaults</a>, 
<a href="#appmesh.k8s.aws/v1beta2.VirtualServiceBackend">VirtualServiceBackend</a>)
</p>
<p>
<p>ClientPolicy refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_ClientPolicy.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_ClientPolicy.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>tls</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.ClientPolicyTLS">
ClientPolicyTLS
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>A reference to an object that represents a Transport Layer Security (TLS) client policy.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.ClientPolicyTLS">ClientPolicyTLS
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.ClientPolicy">ClientPolicy</a>)
</p>
<p>
<p>ClientPolicyTLS refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_ClientPolicyTls.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_ClientPolicyTls.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>enforce</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Whether the policy is enforced.
If unspecified, default settings from AWS API will be applied. Refer to AWS Docs for default settings.</p>
</td>
</tr>
<tr>
<td>
<code>ports</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.PortNumber">
[]PortNumber
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The range of ports that the policy is enforced for.</p>
</td>
</tr>
<tr>
<td>
<code>validation</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.TLSValidationContext">
TLSValidationContext
</a>
</em>
</td>
<td>
<p>A reference to an object that represents a TLS validation context.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.DNSServiceDiscovery">DNSServiceDiscovery
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.ServiceDiscovery">ServiceDiscovery</a>)
</p>
<p>
<p>DNSServiceDiscovery refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_DnsServiceDiscovery.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_DnsServiceDiscovery.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>hostname</code></br>
<em>
string
</em>
</td>
<td>
<p>Specifies the DNS service discovery hostname for the virtual node.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.Duration">Duration
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.GRPCRetryPolicy">GRPCRetryPolicy</a>, 
<a href="#appmesh.k8s.aws/v1beta2.GRPCTimeout">GRPCTimeout</a>, 
<a href="#appmesh.k8s.aws/v1beta2.HTTPRetryPolicy">HTTPRetryPolicy</a>, 
<a href="#appmesh.k8s.aws/v1beta2.HTTPTimeout">HTTPTimeout</a>, 
<a href="#appmesh.k8s.aws/v1beta2.TCPTimeout">TCPTimeout</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>unit</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.DurationUnit">
DurationUnit
</a>
</em>
</td>
<td>
<p>A unit of time.</p>
</td>
</tr>
<tr>
<td>
<code>value</code></br>
<em>
int64
</em>
</td>
<td>
<p>A number of time units.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.DurationUnit">DurationUnit
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.Duration">Duration</a>)
</p>
<p>
</p>
<h3 id="appmesh.k8s.aws/v1beta2.EgressFilter">EgressFilter
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.MeshSpec">MeshSpec</a>)
</p>
<p>
<p>EgressFilter refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_EgressFilter.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_EgressFilter.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.EgressFilterType">
EgressFilterType
</a>
</em>
</td>
<td>
<p>The egress filter type.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.EgressFilterType">EgressFilterType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.EgressFilter">EgressFilter</a>)
</p>
<p>
</p>
<h3 id="appmesh.k8s.aws/v1beta2.FileAccessLog">FileAccessLog
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.AccessLog">AccessLog</a>)
</p>
<p>
<p>FileAccessLog refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_FileAccessLog.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_FileAccessLog.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>path</code></br>
<em>
string
</em>
</td>
<td>
<p>The file path to write access logs to.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.GRPCRetryPolicy">GRPCRetryPolicy
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.GRPCRoute">GRPCRoute</a>)
</p>
<p>
<p>GRPCRetryPolicy refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_GrpcRetryPolicy.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_GrpcRetryPolicy.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>grpcRetryEvents</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.GRPCRetryPolicyEvent">
[]GRPCRetryPolicyEvent
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>httpRetryEvents</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.HTTPRetryPolicyEvent">
[]HTTPRetryPolicyEvent
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>tcpRetryEvents</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.TCPRetryPolicyEvent">
[]TCPRetryPolicyEvent
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>maxRetries</code></br>
<em>
int64
</em>
</td>
<td>
<p>The maximum number of retry attempts.</p>
</td>
</tr>
<tr>
<td>
<code>perRetryTimeout</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.Duration">
Duration
</a>
</em>
</td>
<td>
<p>An object that represents a duration of time.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.GRPCRetryPolicyEvent">GRPCRetryPolicyEvent
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.GRPCRetryPolicy">GRPCRetryPolicy</a>)
</p>
<p>
</p>
<h3 id="appmesh.k8s.aws/v1beta2.GRPCRoute">GRPCRoute
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.Route">Route</a>)
</p>
<p>
<p>GRPCRoute refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_GrpcRoute.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_GrpcRoute.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>match</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.GRPCRouteMatch">
GRPCRouteMatch
</a>
</em>
</td>
<td>
<p>An object that represents the criteria for determining a request match.</p>
</td>
</tr>
<tr>
<td>
<code>action</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.GRPCRouteAction">
GRPCRouteAction
</a>
</em>
</td>
<td>
<p>An object that represents the action to take if a match is determined.</p>
</td>
</tr>
<tr>
<td>
<code>retryPolicy</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.GRPCRetryPolicy">
GRPCRetryPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>An object that represents a retry policy.</p>
</td>
</tr>
<tr>
<td>
<code>timeout</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.GRPCTimeout">
GRPCTimeout
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>An object that represents a grpc timeout.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.GRPCRouteAction">GRPCRouteAction
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.GRPCRoute">GRPCRoute</a>)
</p>
<p>
<p>GRPCRouteAction refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_GrpcRouteAction.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_GrpcRouteAction.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>weightedTargets</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.WeightedTarget">
[]WeightedTarget
</a>
</em>
</td>
<td>
<p>An object that represents the targets that traffic is routed to when a request matches the route.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.GRPCRouteMatch">GRPCRouteMatch
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.GRPCRoute">GRPCRoute</a>)
</p>
<p>
<p>GRPCRouteMatch refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_GrpcRouteMatch.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_GrpcRouteMatch.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>methodName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The method name to match from the request. If you specify a name, you must also specify a serviceName.</p>
</td>
</tr>
<tr>
<td>
<code>serviceName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The fully qualified domain name for the service to match from the request.</p>
</td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.GRPCRouteMetadata">
[]GRPCRouteMetadata
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>An object that represents the data to match from the request.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.GRPCRouteMetadata">GRPCRouteMetadata
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.GRPCRouteMatch">GRPCRouteMatch</a>)
</p>
<p>
<p>GRPCRouteMetadata refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_GrpcRouteMetadata.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_GrpcRouteMetadata.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>The name of the route.</p>
</td>
</tr>
<tr>
<td>
<code>match</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.GRPCRouteMetadataMatchMethod">
GRPCRouteMetadataMatchMethod
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>An object that represents the data to match from the request.</p>
</td>
</tr>
<tr>
<td>
<code>invert</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Specify True to match anything except the match criteria. The default value is False.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.GRPCRouteMetadataMatchMethod">GRPCRouteMetadataMatchMethod
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.GRPCRouteMetadata">GRPCRouteMetadata</a>)
</p>
<p>
<p>GRPCRouteMetadataMatchMethod refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_GrpcRouteMetadataMatchMethod.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_GrpcRouteMetadataMatchMethod.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>exact</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The value sent by the client must match the specified value exactly.</p>
</td>
</tr>
<tr>
<td>
<code>prefix</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The value sent by the client must begin with the specified characters.</p>
</td>
</tr>
<tr>
<td>
<code>range</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.MatchRange">
MatchRange
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>An object that represents the range of values to match on</p>
</td>
</tr>
<tr>
<td>
<code>regex</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The value sent by the client must include the specified characters.</p>
</td>
</tr>
<tr>
<td>
<code>suffix</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The value sent by the client must end with the specified characters.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.GRPCTimeout">GRPCTimeout
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.GRPCRoute">GRPCRoute</a>, 
<a href="#appmesh.k8s.aws/v1beta2.ListenerTimeout">ListenerTimeout</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>perRequest</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.Duration">
Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>An object that represents per request timeout duration.</p>
</td>
</tr>
<tr>
<td>
<code>idle</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.Duration">
Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>An object that represents idle timeout duration.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.HTTPRetryPolicy">HTTPRetryPolicy
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.HTTPRoute">HTTPRoute</a>)
</p>
<p>
<p>HTTPRetryPolicy refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_HttpRetryPolicy.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_HttpRetryPolicy.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>httpRetryEvents</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.HTTPRetryPolicyEvent">
[]HTTPRetryPolicyEvent
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>tcpRetryEvents</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.TCPRetryPolicyEvent">
[]TCPRetryPolicyEvent
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>maxRetries</code></br>
<em>
int64
</em>
</td>
<td>
<p>The maximum number of retry attempts.</p>
</td>
</tr>
<tr>
<td>
<code>perRetryTimeout</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.Duration">
Duration
</a>
</em>
</td>
<td>
<p>An object that represents a duration of time</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.HTTPRetryPolicyEvent">HTTPRetryPolicyEvent
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.GRPCRetryPolicy">GRPCRetryPolicy</a>, 
<a href="#appmesh.k8s.aws/v1beta2.HTTPRetryPolicy">HTTPRetryPolicy</a>)
</p>
<p>
</p>
<h3 id="appmesh.k8s.aws/v1beta2.HTTPRoute">HTTPRoute
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.Route">Route</a>)
</p>
<p>
<p>HTTPRoute refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_HttpRoute.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_HttpRoute.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>match</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.HTTPRouteMatch">
HTTPRouteMatch
</a>
</em>
</td>
<td>
<p>An object that represents the criteria for determining a request match.</p>
</td>
</tr>
<tr>
<td>
<code>action</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.HTTPRouteAction">
HTTPRouteAction
</a>
</em>
</td>
<td>
<p>An object that represents the action to take if a match is determined.</p>
</td>
</tr>
<tr>
<td>
<code>retryPolicy</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.HTTPRetryPolicy">
HTTPRetryPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>An object that represents a retry policy.</p>
</td>
</tr>
<tr>
<td>
<code>timeout</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.HTTPTimeout">
HTTPTimeout
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>An object that represents a http timeout.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.HTTPRouteAction">HTTPRouteAction
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.HTTPRoute">HTTPRoute</a>)
</p>
<p>
<p>HTTPRouteAction refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_HttpRouteAction.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_HttpRouteAction.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>weightedTargets</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.WeightedTarget">
[]WeightedTarget
</a>
</em>
</td>
<td>
<p>An object that represents the targets that traffic is routed to when a request matches the route.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.HTTPRouteHeader">HTTPRouteHeader
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.HTTPRouteMatch">HTTPRouteMatch</a>)
</p>
<p>
<p>HTTPRouteHeader refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_HttpRouteHeader.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_HttpRouteHeader.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>A name for the HTTP header in the client request that will be matched on.</p>
</td>
</tr>
<tr>
<td>
<code>match</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.HeaderMatchMethod">
HeaderMatchMethod
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The HeaderMatchMethod object.</p>
</td>
</tr>
<tr>
<td>
<code>invert</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>Specify True to match anything except the match criteria. The default value is False.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.HTTPRouteMatch">HTTPRouteMatch
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.HTTPRoute">HTTPRoute</a>)
</p>
<p>
<p>HTTPRouteMatch refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_HttpRouteMatch.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_HttpRouteMatch.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>headers</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.HTTPRouteHeader">
[]HTTPRouteHeader
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>An object that represents the client request headers to match on.</p>
</td>
</tr>
<tr>
<td>
<code>method</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The client request method to match on.</p>
</td>
</tr>
<tr>
<td>
<code>prefix</code></br>
<em>
string
</em>
</td>
<td>
<p>Specifies the path to match requests with</p>
</td>
</tr>
<tr>
<td>
<code>scheme</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The client request scheme to match on</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.HTTPTimeout">HTTPTimeout
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.HTTPRoute">HTTPRoute</a>, 
<a href="#appmesh.k8s.aws/v1beta2.ListenerTimeout">ListenerTimeout</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>perRequest</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.Duration">
Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>An object that represents per request timeout duration.</p>
</td>
</tr>
<tr>
<td>
<code>idle</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.Duration">
Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>An object that represents idle timeout duration.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.HeaderMatchMethod">HeaderMatchMethod
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.HTTPRouteHeader">HTTPRouteHeader</a>)
</p>
<p>
<p>HeaderMatchMethod refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_HeaderMatchMethod.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_HeaderMatchMethod.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>exact</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The value sent by the client must match the specified value exactly.</p>
</td>
</tr>
<tr>
<td>
<code>prefix</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The value sent by the client must begin with the specified characters.</p>
</td>
</tr>
<tr>
<td>
<code>range</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.MatchRange">
MatchRange
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>An object that represents the range of values to match on.</p>
</td>
</tr>
<tr>
<td>
<code>regex</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The value sent by the client must include the specified characters.</p>
</td>
</tr>
<tr>
<td>
<code>suffix</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The value sent by the client must end with the specified characters.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.HealthCheckPolicy">HealthCheckPolicy
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.Listener">Listener</a>)
</p>
<p>
<p>HealthCheckPolicy refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_HealthCheckPolicy.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_HealthCheckPolicy.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>healthyThreshold</code></br>
<em>
int64
</em>
</td>
<td>
<p>The number of consecutive successful health checks that must occur before declaring listener healthy.</p>
</td>
</tr>
<tr>
<td>
<code>intervalMillis</code></br>
<em>
int64
</em>
</td>
<td>
<p>The time period in milliseconds between each health check execution.</p>
</td>
</tr>
<tr>
<td>
<code>path</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The destination path for the health check request.
This value is only used if the specified protocol is http or http2. For any other protocol, this value is ignored.</p>
</td>
</tr>
<tr>
<td>
<code>port</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.PortNumber">
PortNumber
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The destination port for the health check request.</p>
</td>
</tr>
<tr>
<td>
<code>protocol</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.PortProtocol">
PortProtocol
</a>
</em>
</td>
<td>
<p>The protocol for the health check request</p>
</td>
</tr>
<tr>
<td>
<code>timeoutMillis</code></br>
<em>
int64
</em>
</td>
<td>
<p>The amount of time to wait when receiving a response from the health check, in milliseconds.</p>
</td>
</tr>
<tr>
<td>
<code>unhealthyThreshold</code></br>
<em>
int64
</em>
</td>
<td>
<p>The number of consecutive failed health checks that must occur before declaring a virtual node unhealthy.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.Listener">Listener
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualNodeSpec">VirtualNodeSpec</a>)
</p>
<p>
<p>Listener refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_Listener.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_Listener.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>portMapping</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.PortMapping">
PortMapping
</a>
</em>
</td>
<td>
<p>The port mapping information for the listener.</p>
</td>
</tr>
<tr>
<td>
<code>healthCheck</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.HealthCheckPolicy">
HealthCheckPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The health check information for the listener.</p>
</td>
</tr>
<tr>
<td>
<code>tls</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.ListenerTLS">
ListenerTLS
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>A reference to an object that represents the Transport Layer Security (TLS) properties for a listener.</p>
</td>
</tr>
<tr>
<td>
<code>timeout</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.ListenerTimeout">
ListenerTimeout
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>A reference to an object that represents</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.ListenerTLS">ListenerTLS
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.Listener">Listener</a>)
</p>
<p>
<p>ListenerTLS refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_ListenerTls.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_ListenerTls.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>certificate</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.ListenerTLSCertificate">
ListenerTLSCertificate
</a>
</em>
</td>
<td>
<p>A reference to an object that represents a listener&rsquo;s TLS certificate.</p>
</td>
</tr>
<tr>
<td>
<code>mode</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.ListenerTLSMode">
ListenerTLSMode
</a>
</em>
</td>
<td>
<p>ListenerTLS mode</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.ListenerTLSACMCertificate">ListenerTLSACMCertificate
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.ListenerTLSCertificate">ListenerTLSCertificate</a>)
</p>
<p>
<p>ListenerTLSACMCertificate refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_ListenerTlsAcmCertificate.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_ListenerTlsAcmCertificate.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>certificateARN</code></br>
<em>
string
</em>
</td>
<td>
<p>The Amazon Resource Name (ARN) for the certificate.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.ListenerTLSCertificate">ListenerTLSCertificate
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.ListenerTLS">ListenerTLS</a>)
</p>
<p>
<p>ListenerTLSCertificate refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_ListenerTlsCertificate.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_ListenerTlsCertificate.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>acm</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.ListenerTLSACMCertificate">
ListenerTLSACMCertificate
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>A reference to an object that represents an AWS Certificate Manager (ACM) certificate.</p>
</td>
</tr>
<tr>
<td>
<code>file</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.ListenerTLSFileCertificate">
ListenerTLSFileCertificate
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>A reference to an object that represents a local file certificate.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.ListenerTLSFileCertificate">ListenerTLSFileCertificate
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.ListenerTLSCertificate">ListenerTLSCertificate</a>)
</p>
<p>
<p>ListenerTLSFileCertificate refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_ListenerTlsFileCertificate.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_ListenerTlsFileCertificate.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>certificateChain</code></br>
<em>
string
</em>
</td>
<td>
<p>The certificate chain for the certificate.</p>
</td>
</tr>
<tr>
<td>
<code>privateKey</code></br>
<em>
string
</em>
</td>
<td>
<p>The private key for a certificate stored on the file system of the virtual node that the proxy is running on.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.ListenerTLSMode">ListenerTLSMode
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.ListenerTLS">ListenerTLS</a>)
</p>
<p>
</p>
<h3 id="appmesh.k8s.aws/v1beta2.ListenerTimeout">ListenerTimeout
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.Listener">Listener</a>)
</p>
<p>
<p>ListenerTimeout refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_ListenerTimeout.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_ListenerTimeout.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>tcp</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.TCPTimeout">
TCPTimeout
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Specifies tcp timeout information for the virtual node.</p>
</td>
</tr>
<tr>
<td>
<code>http</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.HTTPTimeout">
HTTPTimeout
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Specifies http timeout information for the virtual node.</p>
</td>
</tr>
<tr>
<td>
<code>http2</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.HTTPTimeout">
HTTPTimeout
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Specifies http2 information for the virtual node.</p>
</td>
</tr>
<tr>
<td>
<code>grpc</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.GRPCTimeout">
GRPCTimeout
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Specifies grpc timeout information for the virtual node.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.Logging">Logging
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualNodeSpec">VirtualNodeSpec</a>)
</p>
<p>
<p>Logging refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_Logging.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_Logging.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>accessLog</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.AccessLog">
AccessLog
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The access log configuration for a virtual node.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.MatchRange">MatchRange
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.GRPCRouteMetadataMatchMethod">GRPCRouteMetadataMatchMethod</a>, 
<a href="#appmesh.k8s.aws/v1beta2.HeaderMatchMethod">HeaderMatchMethod</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>start</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>The start of the range.</p>
</td>
</tr>
<tr>
<td>
<code>end</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>The end of the range.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.MeshCondition">MeshCondition
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.MeshStatus">MeshStatus</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.MeshConditionType">
MeshConditionType
</a>
</em>
</td>
<td>
<p>Type of mesh condition.</p>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.16/#conditionstatus-v1-core">
Kubernetes core/v1.ConditionStatus
</a>
</em>
</td>
<td>
<p>Status of the condition, one of True, False, Unknown.</p>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.16/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Last time the condition transitioned from one status to another.</p>
</td>
</tr>
<tr>
<td>
<code>reason</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The reason for the condition&rsquo;s last transition.</p>
</td>
</tr>
<tr>
<td>
<code>message</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>A human readable message indicating details about the transition.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.MeshConditionType">MeshConditionType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.MeshCondition">MeshCondition</a>)
</p>
<p>
</p>
<h3 id="appmesh.k8s.aws/v1beta2.MeshReference">MeshReference
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualNodeSpec">VirtualNodeSpec</a>, 
<a href="#appmesh.k8s.aws/v1beta2.VirtualRouterSpec">VirtualRouterSpec</a>, 
<a href="#appmesh.k8s.aws/v1beta2.VirtualServiceSpec">VirtualServiceSpec</a>)
</p>
<p>
<p>MeshReference holds a reference to Mesh.appmesh.k8s.aws</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of Mesh CR</p>
</td>
</tr>
<tr>
<td>
<code>uid</code></br>
<em>
k8s.io/apimachinery/pkg/types.UID
</em>
</td>
<td>
<p>UID is the UID of Mesh CR</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.MeshSpec">MeshSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.Mesh">Mesh</a>)
</p>
<p>
<p>MeshSpec defines the desired state of Mesh
refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_MeshSpec.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_MeshSpec.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>awsName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>AWSName is the AppMesh Mesh object&rsquo;s name.
If unspecified or empty, it defaults to be &ldquo;${name}&rdquo; of k8s Mesh</p>
</td>
</tr>
<tr>
<td>
<code>namespaceSelector</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.16/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>NamespaceSelector selects Namespaces using labels to designate mesh membership.
This field follows standard label selector semantics:
if present but empty, it selects all namespaces.
if absent, it selects no namespace.</p>
</td>
</tr>
<tr>
<td>
<code>egressFilter</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.EgressFilter">
EgressFilter
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The egress filter rules for the service mesh.
If unspecified, default settings from AWS API will be applied. Refer to AWS Docs for default settings.</p>
</td>
</tr>
<tr>
<td>
<code>meshOwner</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The AWS IAM account ID of the service mesh owner.
Required if the account ID is not your own.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.MeshStatus">MeshStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.Mesh">Mesh</a>)
</p>
<p>
<p>MeshStatus defines the observed state of Mesh</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>meshARN</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>MeshARN is the AppMesh Mesh object&rsquo;s Amazon Resource Name</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.MeshCondition">
[]MeshCondition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The current Mesh status.</p>
</td>
</tr>
<tr>
<td>
<code>observedGeneration</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>The generation observed by the Mesh controller.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.PortMapping">PortMapping
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.Listener">Listener</a>, 
<a href="#appmesh.k8s.aws/v1beta2.VirtualRouterListener">VirtualRouterListener</a>)
</p>
<p>
<p>PortMapping refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_PortMapping.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_PortMapping.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>port</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.PortNumber">
PortNumber
</a>
</em>
</td>
<td>
<p>The port used for the port mapping.</p>
</td>
</tr>
<tr>
<td>
<code>protocol</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.PortProtocol">
PortProtocol
</a>
</em>
</td>
<td>
<p>The protocol used for the port mapping.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.PortNumber">PortNumber
(<code>int64</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.ClientPolicyTLS">ClientPolicyTLS</a>, 
<a href="#appmesh.k8s.aws/v1beta2.HealthCheckPolicy">HealthCheckPolicy</a>, 
<a href="#appmesh.k8s.aws/v1beta2.PortMapping">PortMapping</a>)
</p>
<p>
</p>
<h3 id="appmesh.k8s.aws/v1beta2.PortProtocol">PortProtocol
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.HealthCheckPolicy">HealthCheckPolicy</a>, 
<a href="#appmesh.k8s.aws/v1beta2.PortMapping">PortMapping</a>)
</p>
<p>
</p>
<h3 id="appmesh.k8s.aws/v1beta2.Route">Route
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualRouterSpec">VirtualRouterSpec</a>)
</p>
<p>
<p>Route refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_RouteSpec.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_RouteSpec.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Route&rsquo;s name</p>
</td>
</tr>
<tr>
<td>
<code>grpcRoute</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.GRPCRoute">
GRPCRoute
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>An object that represents the specification of a gRPC route.</p>
</td>
</tr>
<tr>
<td>
<code>httpRoute</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.HTTPRoute">
HTTPRoute
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>An object that represents the specification of an HTTP route.</p>
</td>
</tr>
<tr>
<td>
<code>http2Route</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.HTTPRoute">
HTTPRoute
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>An object that represents the specification of an HTTP/2 route.</p>
</td>
</tr>
<tr>
<td>
<code>tcpRoute</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.TCPRoute">
TCPRoute
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>An object that represents the specification of a TCP route.</p>
</td>
</tr>
<tr>
<td>
<code>priority</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>The priority for the route.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.ServiceDiscovery">ServiceDiscovery
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualNodeSpec">VirtualNodeSpec</a>)
</p>
<p>
<p>ServiceDiscovery refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_ServiceDiscovery.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_ServiceDiscovery.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>awsCloudMap</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.AWSCloudMapServiceDiscovery">
AWSCloudMapServiceDiscovery
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Specifies any AWS Cloud Map information for the virtual node.</p>
</td>
</tr>
<tr>
<td>
<code>dns</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.DNSServiceDiscovery">
DNSServiceDiscovery
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Specifies the DNS information for the virtual node.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.TCPRetryPolicyEvent">TCPRetryPolicyEvent
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.GRPCRetryPolicy">GRPCRetryPolicy</a>, 
<a href="#appmesh.k8s.aws/v1beta2.HTTPRetryPolicy">HTTPRetryPolicy</a>)
</p>
<p>
</p>
<h3 id="appmesh.k8s.aws/v1beta2.TCPRoute">TCPRoute
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.Route">Route</a>)
</p>
<p>
<p>TCPRoute refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_TcpRoute.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_TcpRoute.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>action</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.TCPRouteAction">
TCPRouteAction
</a>
</em>
</td>
<td>
<p>The action to take if a match is determined.</p>
</td>
</tr>
<tr>
<td>
<code>timeout</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.TCPTimeout">
TCPTimeout
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>An object that represents a tcp timeout.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.TCPRouteAction">TCPRouteAction
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.TCPRoute">TCPRoute</a>)
</p>
<p>
<p>TCPRouteAction refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_TcpRouteAction.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_TcpRouteAction.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>weightedTargets</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.WeightedTarget">
[]WeightedTarget
</a>
</em>
</td>
<td>
<p>An object that represents the targets that traffic is routed to when a request matches the route.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.TCPTimeout">TCPTimeout
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.ListenerTimeout">ListenerTimeout</a>, 
<a href="#appmesh.k8s.aws/v1beta2.TCPRoute">TCPRoute</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>idle</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.Duration">
Duration
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>An object that represents idle timeout duration.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.TLSValidationContext">TLSValidationContext
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.ClientPolicyTLS">ClientPolicyTLS</a>)
</p>
<p>
<p>TLSValidationContext refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_TlsValidationContext.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_TlsValidationContext.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>trust</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.TLSValidationContextTrust">
TLSValidationContextTrust
</a>
</em>
</td>
<td>
<p>A reference to an object that represents a TLS validation context trust</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.TLSValidationContextACMTrust">TLSValidationContextACMTrust
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.TLSValidationContextTrust">TLSValidationContextTrust</a>)
</p>
<p>
<p>TLSValidationContextACMTrust refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_TlsValidationContextAcmTrust.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_TlsValidationContextAcmTrust.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>certificateAuthorityARNs</code></br>
<em>
[]string
</em>
</td>
<td>
<p>One or more ACM Amazon Resource Name (ARN)s.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.TLSValidationContextFileTrust">TLSValidationContextFileTrust
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.TLSValidationContextTrust">TLSValidationContextTrust</a>)
</p>
<p>
<p>TLSValidationContextFileTrust refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_TlsValidationContextFileTrust.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_TlsValidationContextFileTrust.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>certificateChain</code></br>
<em>
string
</em>
</td>
<td>
<p>The certificate trust chain for a certificate stored on the file system of the virtual node that the proxy is running on.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.TLSValidationContextTrust">TLSValidationContextTrust
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.TLSValidationContext">TLSValidationContext</a>)
</p>
<p>
<p>TLSValidationContextTrust refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_TlsValidationContextTrust.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_TlsValidationContextTrust.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>acm</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.TLSValidationContextACMTrust">
TLSValidationContextACMTrust
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>A reference to an object that represents a TLS validation context trust for an AWS Certicate Manager (ACM) certificate.</p>
</td>
</tr>
<tr>
<td>
<code>file</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.TLSValidationContextFileTrust">
TLSValidationContextFileTrust
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>An object that represents a TLS validation context trust for a local file.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.VirtualNodeCondition">VirtualNodeCondition
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualNodeStatus">VirtualNodeStatus</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualNodeConditionType">
VirtualNodeConditionType
</a>
</em>
</td>
<td>
<p>Type of VirtualNode condition.</p>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.16/#conditionstatus-v1-core">
Kubernetes core/v1.ConditionStatus
</a>
</em>
</td>
<td>
<p>Status of the condition, one of True, False, Unknown.</p>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.16/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Last time the condition transitioned from one status to another.</p>
</td>
</tr>
<tr>
<td>
<code>reason</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The reason for the condition&rsquo;s last transition.</p>
</td>
</tr>
<tr>
<td>
<code>message</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>A human readable message indicating details about the transition.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.VirtualNodeConditionType">VirtualNodeConditionType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualNodeCondition">VirtualNodeCondition</a>)
</p>
<p>
</p>
<h3 id="appmesh.k8s.aws/v1beta2.VirtualNodeReference">VirtualNodeReference
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualNodeServiceProvider">VirtualNodeServiceProvider</a>, 
<a href="#appmesh.k8s.aws/v1beta2.WeightedTarget">WeightedTarget</a>)
</p>
<p>
<p>VirtualNodeReference holds a reference to VirtualNode.appmesh.k8s.aws</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>namespace</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Namespace is the namespace of VirtualNode CR.
If unspecified, defaults to the referencing object&rsquo;s namespace</p>
</td>
</tr>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of VirtualNode CR</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.VirtualNodeServiceProvider">VirtualNodeServiceProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualServiceProvider">VirtualServiceProvider</a>)
</p>
<p>
<p>VirtualNodeServiceProvider refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_VirtualNodeServiceProvider.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_VirtualNodeServiceProvider.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>virtualNodeRef</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualNodeReference">
VirtualNodeReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Reference to Kubernetes VirtualNode CR in cluster that is acting as a service provider. Exactly one of &lsquo;virtualNodeRef&rsquo; or &lsquo;virtualNodeARN&rsquo; must be specified.</p>
</td>
</tr>
<tr>
<td>
<code>virtualNodeARN</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Amazon Resource Name to AppMesh VirtualNode object that is acting as a service provider. Exactly one of &lsquo;virtualNodeRef&rsquo; or &lsquo;virtualNodeARN&rsquo; must be specified.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.VirtualNodeSpec">VirtualNodeSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualNode">VirtualNode</a>)
</p>
<p>
<p>VirtualNodeSpec defines the desired state of VirtualNode
refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_VirtualNodeSpec.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_VirtualNodeSpec.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>awsName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>AWSName is the AppMesh VirtualNode object&rsquo;s name.
If unspecified or empty, it defaults to be &ldquo;${name}_${namespace}&rdquo; of k8s VirtualNode</p>
</td>
</tr>
<tr>
<td>
<code>podSelector</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.16/#labelselector-v1-meta">
Kubernetes meta/v1.LabelSelector
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>PodSelector selects Pods using labels to designate VirtualNode membership.
This field follows standard label selector semantics:
if present but empty, it selects all pods within namespace.
if absent, it selects no pod.</p>
</td>
</tr>
<tr>
<td>
<code>listeners</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.Listener">
[]Listener
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The listener that the virtual node is expected to receive inbound traffic from</p>
</td>
</tr>
<tr>
<td>
<code>serviceDiscovery</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.ServiceDiscovery">
ServiceDiscovery
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The service discovery information for the virtual node.</p>
</td>
</tr>
<tr>
<td>
<code>backends</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.Backend">
[]Backend
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The backends that the virtual node is expected to send outbound traffic to.</p>
</td>
</tr>
<tr>
<td>
<code>backendDefaults</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.BackendDefaults">
BackendDefaults
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>A reference to an object that represents the defaults for backends.</p>
</td>
</tr>
<tr>
<td>
<code>logging</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.Logging">
Logging
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The inbound and outbound access logging information for the virtual node.</p>
</td>
</tr>
<tr>
<td>
<code>meshRef</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.MeshReference">
MeshReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>A reference to k8s Mesh CR that this VirtualNode belongs to.
The admission controller populates it using Meshes&rsquo;s selector, and prevents users from setting this field.</p>
<p>Populated by the system.
Read-only.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.VirtualNodeStatus">VirtualNodeStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualNode">VirtualNode</a>)
</p>
<p>
<p>VirtualNodeStatus defines the observed state of VirtualNode</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>virtualNodeARN</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>VirtualNodeARN is the AppMesh VirtualNode object&rsquo;s Amazon Resource Name</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualNodeCondition">
[]VirtualNodeCondition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The current VirtualNode status.</p>
</td>
</tr>
<tr>
<td>
<code>observedGeneration</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>The generation observed by the VirtualNode controller.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.VirtualRouterCondition">VirtualRouterCondition
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualRouterStatus">VirtualRouterStatus</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualRouterConditionType">
VirtualRouterConditionType
</a>
</em>
</td>
<td>
<p>Type of VirtualRouter condition.</p>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.16/#conditionstatus-v1-core">
Kubernetes core/v1.ConditionStatus
</a>
</em>
</td>
<td>
<p>Status of the condition, one of True, False, Unknown.</p>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.16/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Last time the condition transitioned from one status to another.</p>
</td>
</tr>
<tr>
<td>
<code>reason</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The reason for the condition&rsquo;s last transition.</p>
</td>
</tr>
<tr>
<td>
<code>message</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>A human readable message indicating details about the transition.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.VirtualRouterConditionType">VirtualRouterConditionType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualRouterCondition">VirtualRouterCondition</a>)
</p>
<p>
</p>
<h3 id="appmesh.k8s.aws/v1beta2.VirtualRouterListener">VirtualRouterListener
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualRouterSpec">VirtualRouterSpec</a>)
</p>
<p>
<p>VirtualRouterListener refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_VirtualRouterListener.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_VirtualRouterListener.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>portMapping</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.PortMapping">
PortMapping
</a>
</em>
</td>
<td>
<p>The port mapping information for the listener.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.VirtualRouterReference">VirtualRouterReference
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualRouterServiceProvider">VirtualRouterServiceProvider</a>)
</p>
<p>
<p>VirtualRouterReference holds a reference to VirtualRouter.appmesh.k8s.aws</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>namespace</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Namespace is the namespace of VirtualRouter CR.
If unspecified, defaults to the referencing object&rsquo;s namespace</p>
</td>
</tr>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of VirtualRouter CR</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.VirtualRouterServiceProvider">VirtualRouterServiceProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualServiceProvider">VirtualServiceProvider</a>)
</p>
<p>
<p>VirtualRouterServiceProvider refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_VirtualRouterServiceProvider.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_VirtualRouterServiceProvider.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>virtualRouterRef</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualRouterReference">
VirtualRouterReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Reference to Kubernetes VirtualRouter CR in cluster that is acting as a service provider. Exactly one of &lsquo;virtualRouterRef&rsquo; or &lsquo;virtualRouterARN&rsquo; must be specified.</p>
</td>
</tr>
<tr>
<td>
<code>virtualRouterARN</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Amazon Resource Name to AppMesh VirtualRouter object that is acting as a service provider. Exactly one of &lsquo;virtualRouterRef&rsquo; or &lsquo;virtualRouterARN&rsquo; must be specified.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.VirtualRouterSpec">VirtualRouterSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualRouter">VirtualRouter</a>)
</p>
<p>
<p>VirtualRouterSpec defines the desired state of VirtualRouter
refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_VirtualRouterSpec.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_VirtualRouterSpec.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>awsName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>AWSName is the AppMesh VirtualRouter object&rsquo;s name.
If unspecified or empty, it defaults to be &ldquo;${name}_${namespace}&rdquo; of k8s VirtualRouter</p>
</td>
</tr>
<tr>
<td>
<code>listeners</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualRouterListener">
[]VirtualRouterListener
</a>
</em>
</td>
<td>
<p>The listeners that the virtual router is expected to receive inbound traffic from</p>
</td>
</tr>
<tr>
<td>
<code>routes</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.Route">
[]Route
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The routes associated with VirtualRouter</p>
</td>
</tr>
<tr>
<td>
<code>meshRef</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.MeshReference">
MeshReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>A reference to k8s Mesh CR that this VirtualRouter belongs to.
The admission controller populates it using Meshes&rsquo;s selector, and prevents users from setting this field.</p>
<p>Populated by the system.
Read-only.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.VirtualRouterStatus">VirtualRouterStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualRouter">VirtualRouter</a>)
</p>
<p>
<p>VirtualRouterStatus defines the observed state of VirtualRouter</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>virtualRouterARN</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>VirtualRouterARN is the AppMesh VirtualRouter object&rsquo;s Amazon Resource Name.</p>
</td>
</tr>
<tr>
<td>
<code>routeARNs</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>RouteARNs is a map of AppMesh Route objects&rsquo; Amazon Resource Names, indexed by route name.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualRouterCondition">
[]VirtualRouterCondition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The current VirtualRouter status.</p>
</td>
</tr>
<tr>
<td>
<code>observedGeneration</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>The generation observed by the VirtualRouter controller.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.VirtualServiceBackend">VirtualServiceBackend
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.Backend">Backend</a>)
</p>
<p>
<p>VirtualServiceBackend refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_VirtualServiceBackend.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_VirtualServiceBackend.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>virtualServiceRef</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualServiceReference">
VirtualServiceReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Reference to Kubernetes VirtualService CR in cluster that is acting as a virtual node backend. Exactly one of &lsquo;virtualServiceRef&rsquo; or &lsquo;virtualServiceARN&rsquo; must be specified.</p>
</td>
</tr>
<tr>
<td>
<code>virtualServiceARN</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Amazon Resource Name to AppMesh VirtualService object that is acting as a virtual node backend. Exactly one of &lsquo;virtualServiceRef&rsquo; or &lsquo;virtualServiceARN&rsquo; must be specified.</p>
</td>
</tr>
<tr>
<td>
<code>clientPolicy</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.ClientPolicy">
ClientPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>A reference to an object that represents the client policy for a backend.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.VirtualServiceCondition">VirtualServiceCondition
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualServiceStatus">VirtualServiceStatus</a>)
</p>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>type</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualServiceConditionType">
VirtualServiceConditionType
</a>
</em>
</td>
<td>
<p>Type of VirtualService condition.</p>
</td>
</tr>
<tr>
<td>
<code>status</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.16/#conditionstatus-v1-core">
Kubernetes core/v1.ConditionStatus
</a>
</em>
</td>
<td>
<p>Status of the condition, one of True, False, Unknown.</p>
</td>
</tr>
<tr>
<td>
<code>lastTransitionTime</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.16/#time-v1-meta">
Kubernetes meta/v1.Time
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Last time the condition transitioned from one status to another.</p>
</td>
</tr>
<tr>
<td>
<code>reason</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>The reason for the condition&rsquo;s last transition.</p>
</td>
</tr>
<tr>
<td>
<code>message</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>A human readable message indicating details about the transition.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.VirtualServiceConditionType">VirtualServiceConditionType
(<code>string</code> alias)</p></h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualServiceCondition">VirtualServiceCondition</a>)
</p>
<p>
</p>
<h3 id="appmesh.k8s.aws/v1beta2.VirtualServiceProvider">VirtualServiceProvider
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualServiceSpec">VirtualServiceSpec</a>)
</p>
<p>
<p>VirtualServiceProvider refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_VirtualServiceProvider.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_VirtualServiceProvider.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>virtualNode</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualNodeServiceProvider">
VirtualNodeServiceProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The virtual node associated with a virtual service.</p>
</td>
</tr>
<tr>
<td>
<code>virtualRouter</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualRouterServiceProvider">
VirtualRouterServiceProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The virtual router associated with a virtual service.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.VirtualServiceReference">VirtualServiceReference
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualServiceBackend">VirtualServiceBackend</a>)
</p>
<p>
<p>VirtualServiceReference holds a reference to VirtualService.appmesh.k8s.aws</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>namespace</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Namespace is the namespace of VirtualService CR.
If unspecified, defaults to the referencing object&rsquo;s namespace</p>
</td>
</tr>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of VirtualService CR</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.VirtualServiceSpec">VirtualServiceSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualService">VirtualService</a>)
</p>
<p>
<p>VirtualServiceSpec defines the desired state of VirtualService
refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_VirtualServiceSpec.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_VirtualServiceSpec.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>awsName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>AWSName is the AppMesh VirtualService object&rsquo;s name.
If unspecified or empty, it defaults to be &ldquo;${name}.${namespace}&rdquo; of k8s VirtualService</p>
</td>
</tr>
<tr>
<td>
<code>provider</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualServiceProvider">
VirtualServiceProvider
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The provider for virtual services. You can specify a single virtual node or virtual router.</p>
</td>
</tr>
<tr>
<td>
<code>meshRef</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.MeshReference">
MeshReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>A reference to k8s Mesh CR that this VirtualService belongs to.
The admission controller populates it using Meshes&rsquo;s selector, and prevents users from setting this field.</p>
<p>Populated by the system.
Read-only.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.VirtualServiceStatus">VirtualServiceStatus
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualService">VirtualService</a>)
</p>
<p>
<p>VirtualServiceStatus defines the observed state of VirtualService</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>virtualServiceARN</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>VirtualServiceARN is the AppMesh VirtualService object&rsquo;s Amazon Resource Name.</p>
</td>
</tr>
<tr>
<td>
<code>conditions</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualServiceCondition">
[]VirtualServiceCondition
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>The current VirtualService status.</p>
</td>
</tr>
<tr>
<td>
<code>observedGeneration</code></br>
<em>
int64
</em>
</td>
<td>
<em>(Optional)</em>
<p>The generation observed by the VirtualService controller.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="appmesh.k8s.aws/v1beta2.WeightedTarget">WeightedTarget
</h3>
<p>
(<em>Appears on:</em>
<a href="#appmesh.k8s.aws/v1beta2.GRPCRouteAction">GRPCRouteAction</a>, 
<a href="#appmesh.k8s.aws/v1beta2.HTTPRouteAction">HTTPRouteAction</a>, 
<a href="#appmesh.k8s.aws/v1beta2.TCPRouteAction">TCPRouteAction</a>)
</p>
<p>
<p>WeightedTarget refers to <a href="https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_WeightedTarget.html">https://docs.aws.amazon.com/app-mesh/latest/APIReference/API_WeightedTarget.html</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>virtualNodeRef</code></br>
<em>
<a href="#appmesh.k8s.aws/v1beta2.VirtualNodeReference">
VirtualNodeReference
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Reference to Kubernetes VirtualNode CR in cluster to associate with the weighted target. Exactly one of &lsquo;virtualNodeRef&rsquo; or &lsquo;virtualNodeARN&rsquo; must be specified.</p>
</td>
</tr>
<tr>
<td>
<code>virtualNodeARN</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Amazon Resource Name to AppMesh VirtualNode object to associate with the weighted target. Exactly one of &lsquo;virtualNodeRef&rsquo; or &lsquo;virtualNodeARN&rsquo; must be specified.</p>
</td>
</tr>
<tr>
<td>
<code>weight</code></br>
<em>
int64
</em>
</td>
<td>
<p>The relative weight of the weighted target.</p>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>
on git commit <code>0b14b71</code>.
</em></p>
