package conversions

import (
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/ptypes/duration"
	"testing"

	_ "github.com/gogo/protobuf/gogoproto"
)

func Test_scratch(t *testing.T) {

	const jsonStr = `
		{
			"@type": "type.googleapis.com/aws.lattice.configuration.ClusterDefault",
			"cluster": {
				"@type": "type.googleapis.com/envoy.config.cluster.v3.Cluster",
				"type": "EDS",
				"edsClusterConfig": {
					"edsConfig": {
						"ads": {},
						"resourceApiVersion": "V3"
					}
				},
				"connect_timeout": "10s",
				"circuitBreakers": {
					"thresholds": [
						{
							"maxConnections": 2147483647,
							"maxPendingRequests": 2147483647,
							"maxRequests": 2147483647,
							"maxRetries": 2147483647
						}
					]
				},
				"typedExtensionProtocolOptions": {
					"envoy.extensions.upstreams.http.v3.HttpProtocolOptions": {
						"@type": "type.googleapis.com/envoy.extensions.upstreams.http.v3.HttpProtocolOptions",
						"explicitHttpConfig": {
							"http2ProtocolOptions": {
								"maxConcurrentStreams": 1073741824
							}
						}
					}
				},
				"outlierDetection": {
					"consecutive5xx": 5,
					"interval": "10s",
					"baseEjectionTime": "30s",
					"splitExternalLocalOriginErrors": true,
					"consecutiveLocalOriginFailure": 5,
					"maxEjectionTime": "300s"
				}        
			}
		}
	`

	const jsonStr2 = `
		{
           "name": "hello",
			"cluster": {
				"connect_timeout": "10s"
             }
		}
	`

	buf := json.RawMessage([]byte(jsonStr2))
	var clusterD ClusterDefault
	fmt.Println("Generating ClusterDefault...")
	err := json.Unmarshal(buf, &clusterD)
	if err != nil {
		panic(err)
	}
	fmt.Printf("ClusterDefault: %+v\n", clusterD)
	fmt.Printf("connect timeout: %+v\n", clusterD.Cluster.ConnectTimeout.Seconds)

}

type ClusterDefault struct {
	Name string `json:"name,omitempty"`
	// The configuration data for cluster.
	Cluster Cluster `json:"cluster,omitempty"`
}

type Cluster struct {
	Name           string            `protobuf:"json:"name,omitempty"`
	ConnectTimeout duration.Duration `protobuf:"json:"connect_timeout,omitempty"`
}
