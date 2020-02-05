package metrics

import (
	"fmt"
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	promdto "github.com/prometheus/client_model/go"
)

var stats *Recorder

func TestMain(m *testing.M) {
	stats = NewRecorder(true)
	defer stats.clearRegistry()
	code := m.Run()
	os.Exit(code)
}

func TestRecorder_SetMesh(t *testing.T) {
	stats.SetMeshActive("test-mesh")

	name := "appmesh_mesh_state"
	metric, err := lookupMetric(name, promdto.MetricType_GAUGE, "name", "test-mesh")
	if err != nil {
		t.Fatalf("Error collecting %s metric: %v", name, err)
	}

	if int(*metric.Gauge.Value) != 1 {
		t.Errorf("%s expected value %v got %v", name, 1, *metric.Gauge.Value)
	}

	stats.SetMeshInactive("test-mesh")
	metric, err = lookupMetric(name, promdto.MetricType_GAUGE, "name", "test-mesh")
	if err != nil {
		t.Fatalf("Error collecting %s metric: %v", name, err)
	}

	if int(*metric.Gauge.Value) != 0 {
		t.Errorf("%s expected value %v got %v", name, 0, *metric.Gauge.Value)
	}
}

func TestRecorder_SetVirtualNode(t *testing.T) {
	stats.SetVirtualNodeActive("test-vt", "test-mesh")

	name := "appmesh_virtual_node_state"
	metric, err := lookupMetric(name, promdto.MetricType_GAUGE, "name", "test-vt", "mesh", "test-mesh")
	if err != nil {
		t.Fatalf("Error collecting %s metric: %v", name, err)
	}

	if int(*metric.Gauge.Value) != 1 {
		t.Errorf("%s expected value %v got %v", name, 1, *metric.Gauge.Value)
	}

	stats.SetVirtualNodeInactive("test-vt", "test-mesh")
	metric, err = lookupMetric(name, promdto.MetricType_GAUGE, "name", "test-vt", "mesh", "test-mesh")
	if err != nil {
		t.Fatalf("Error collecting %s metric: %v", name, err)
	}

	if int(*metric.Gauge.Value) != 0 {
		t.Errorf("%s expected value %v got %v", name, 0, *metric.Gauge.Value)
	}
}

func TestRecorder_SetVirtualService(t *testing.T) {
	stats.SetVirtualServiceActive("test-vs", "test-mesh")

	name := "appmesh_virtual_service_state"
	metric, err := lookupMetric(name, promdto.MetricType_GAUGE, "name", "test-vs", "mesh", "test-mesh")
	if err != nil {
		t.Fatalf("Error collecting %s metric: %v", name, err)
	}

	if int(*metric.Gauge.Value) != 1 {
		t.Errorf("%s expected value %v got %v", name, 1, *metric.Gauge.Value)
	}

	stats.SetVirtualServiceInactive("test-vs", "test-mesh")
	metric, err = lookupMetric(name, promdto.MetricType_GAUGE, "name", "test-vs", "mesh", "test-mesh")
	if err != nil {
		t.Fatalf("Error collecting %s metric: %v", name, err)
	}

	if int(*metric.Gauge.Value) != 0 {
		t.Errorf("%s expected value %v got %v", name, 0, *metric.Gauge.Value)
	}
}

func TestRecorder_RecordAWSAPIRequestError(t *testing.T) {
	stats.RecordAWSAPIRequestError("test-svc", "test-op", "test-error-code")

	metric_name := "appmesh_aws_api_errors"
	metric, err := lookupMetric(
		metric_name,
		promdto.MetricType_COUNTER,
		"service", "test-svc",
		"operation", "test-op",
		"errorcode", "test-error-code",
	)
	if err != nil {
		t.Fatalf("Error collecting %s metric: %v", metric_name, err)
	}
	if int(*metric.Counter.Value) != 1 {
		t.Errorf("%s expected value %v got %v", metric_name, 1, *metric.Counter.Value)
	}
}

func lookupMetric(name string, metricType promdto.MetricType, labels ...string) (*promdto.Metric, error) {
	metricsRegistry := prometheus.DefaultRegisterer.(*prometheus.Registry)
	if metrics, err := metricsRegistry.Gather(); err == nil {
		for _, metricFamily := range metrics {
			if *metricFamily.Name == name {
				if *metricFamily.Type != metricType {
					return nil, fmt.Errorf("metric types for %v doesn't correpond: %v != %v", name, metricFamily.Type, metricType)
				}
				for _, metric := range metricFamily.Metric {
					if len(labels) != len(metric.Label)*2 {
						return nil, fmt.Errorf("metric labels length for %v doesn't correpond: %v != %v", name, len(labels)*2, len(metric.Label))
					}
					return metric, nil
				}
				return nil, fmt.Errorf("can't find metric %v with appropriate labels in registry", name)
			}
		}
		return nil, fmt.Errorf("can't find metric %v in registry", name)
	} else {
		return nil, fmt.Errorf("error reading metrics registry %v", err)
	}
}
