package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

// Subsystem represents the Prometheus metrics prefix
const Subsystem = "appmesh"

// Recorder exports mesh stats as Prometheus metrics
type Recorder struct {
	meshState           *prometheus.GaugeVec
	virtualNodeState    *prometheus.GaugeVec
	virtualServiceState *prometheus.GaugeVec
	apiRequestDuration  *prometheus.HistogramVec
	operationDuration   *prometheus.HistogramVec
	awsAPIRequestError  *prometheus.CounterVec
	awsAPIRequestCount  *prometheus.CounterVec
}

// NewRecorder registers the App Mesh metrics
func NewRecorder(register bool) *Recorder {
	meshState := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: Subsystem,
		Name:      "mesh_state",
		Help:      "Mesh state.",
	}, []string{"name"})

	virtualNodeState := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: Subsystem,
		Name:      "virtual_node_state",
		Help:      "Virtual node state.",
	}, []string{"name", "mesh"})

	virtualServiceState := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: Subsystem,
		Name:      "virtual_service_state",
		Help:      "Virtual service state.",
	}, []string{"name", "mesh"})

	apiRequestDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Subsystem: Subsystem,
		Name:      "api_request_duration_seconds",
		Help:      "Seconds spent performing App Mesh API calls.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"kind", "object", "operation"})

	operationDuration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Subsystem: Subsystem,
		Name:      "operation_duration_seconds",
		Help:      "Seconds spent performing operation.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"kind", "object", "operation"})

  awsAPIRequestError := prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: Subsystem,
		Name:      "aws_api_errors",
		Help:      "Cumulative number of errors from the AWS API",
	}, []string{"service", "operation", "errorcode"})

	awsAPIRequestCount := prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: Subsystem,
		Name:      "aws_api_requests",
		Help:      "Cumulative number of requests made to the AWS API",
	}, []string{"service", "operation"})

	if register {
		prometheus.MustRegister(meshState)
		prometheus.MustRegister(virtualNodeState)
		prometheus.MustRegister(virtualServiceState)
		prometheus.MustRegister(apiRequestDuration)
		prometheus.MustRegister(operationDuration)
		prometheus.MustRegister(awsAPIRequestError)
		prometheus.MustRegister(awsAPIRequestCount)
	}

	return &Recorder{
		meshState:           meshState,
		virtualNodeState:    virtualNodeState,
		virtualServiceState: virtualServiceState,
		apiRequestDuration:  apiRequestDuration,
		operationDuration:   operationDuration,
		awsAPIRequestError:  awsAPIRequestError,
		awsAPIRequestCount:  awsAPIRequestCount,
	}
}

func (r *Recorder) clearRegistry() {
	prometheus.Unregister(r.meshState)
	prometheus.Unregister(r.virtualNodeState)
	prometheus.Unregister(r.virtualServiceState)
	prometheus.Unregister(r.apiRequestDuration)
	prometheus.Unregister(r.operationDuration)
	prometheus.Unregister(r.awsAPIRequestError)
	prometheus.Unregister(r.awsAPIRequestCount)
}

// SetMeshActive sets the mesh gauge to 1
func (r *Recorder) SetMeshActive(name string) {
	r.meshState.WithLabelValues(name).Set(1)
}

// SetMeshActive sets the mesh gauge to 0 indicating that the object was deleted
func (r *Recorder) SetMeshInactive(name string) {
	r.meshState.WithLabelValues(name).Set(0)
}

// SetVirtualNodeActive sets the virtual node gauge to 1
func (r *Recorder) SetVirtualNodeActive(name string, mesh string) {
	r.virtualNodeState.WithLabelValues(name, mesh).Set(1)
}

// SetVirtualNodeInactive sets the mesh gauge to 0 indicating that the object was deleted
func (r *Recorder) SetVirtualNodeInactive(name string, mesh string) {
	r.virtualNodeState.WithLabelValues(name, mesh).Set(0)
}

// SetVirtualServiceActive sets the virtual service gauge to 1
func (r *Recorder) SetVirtualServiceActive(name string, mesh string) {
	r.virtualServiceState.WithLabelValues(name, mesh).Set(1)
}

// SetVirtualServiceInactive sets the mesh gauge to 0 indicating that the object was deleted
func (r *Recorder) SetVirtualServiceInactive(name string, mesh string) {
	r.virtualServiceState.WithLabelValues(name, mesh).Set(0)
}

// SetRequestDuration records the duration of App Mesh API calls based on object kind, name and operation type
// The operation type can be get, create, update, delete
func (r *Recorder) SetRequestDuration(kind string, object string, operation string, duration time.Duration) {
	r.apiRequestDuration.WithLabelValues(kind, object, operation).Observe(duration.Seconds())
}

// RecordOperationDuration records the duration of operation (e.g. API Call, Function exection)
// based on object kind, name and operation type. Operation can be any string based on the context, e.g. get, create
func (r *Recorder) RecordOperationDuration(kind string, object string, operation string, duration time.Duration) {
	r.operationDuration.WithLabelValues(kind, object, operation).Observe(duration.Seconds())
}

// RecordAWSAPIRequestError records error count of AWS API calls
func (r *Recorder) RecordAWSAPIRequestError(service string, operation string, errorcode string) {
	r.awsAPIRequestError.WithLabelValues(service, operation, errorcode).Inc()
}

// RecordAWSAPIRequestCount records count of AWS API call attempts
func (r *Recorder) RecordAWSAPIRequestCount(service string, operation string) {
	r.awsAPIRequestCount.WithLabelValues(service, operation).Inc()
}
