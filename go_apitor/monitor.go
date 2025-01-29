package go_apitor

import (
	"time"

	"github.com/muchrief/go-apitor/bloom"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

type InterceptorHandlerFunc func(start time.Time, clientIP, method, path string, status int) error

// Monitor is an object that uses to set gin server monitor.
type Monitor struct {
	slowTime     int32
	metricPath   string
	excludePaths []string
	reqDuration  []float64
	metrics      map[string]*Metric
	metadata     map[string]string
	bloomFilter  *bloom.BloomFilter

	interceptors []InterceptorHandlerFunc
}

// GetMonitor used to get global Monitor object,
// this function returns a singleton object.
func NewDefaultMonitor() *Monitor {
	monitor := &Monitor{
		metricPath:   defaultMetricPath,
		slowTime:     defaultSlowTime,
		reqDuration:  defaultDuration,
		bloomFilter:  bloom.NewBloomFilter(),
		excludePaths: []string{},
		metrics:      make(map[string]*Metric),
		metadata:     make(map[string]string),
		interceptors: []InterceptorHandlerFunc{},
	}

	return monitor
}

// GetMonitor used to get global Monitor object,
// this function returns a singleton object.
func NewMonitor(metricPath string, slowTime int32, duration []float64) *Monitor {
	monitor := &Monitor{
		metricPath:   metricPath,
		slowTime:     slowTime,
		reqDuration:  duration,
		bloomFilter:  bloom.NewBloomFilter(),
		excludePaths: []string{},
		metrics:      make(map[string]*Metric),
		metadata:     make(map[string]string),
		interceptors: []InterceptorHandlerFunc{},
	}

	return monitor
}

func (m *Monitor) AddInterceptors(interceptor InterceptorHandlerFunc) *Monitor {
	if m.interceptors == nil {
		m.interceptors = []InterceptorHandlerFunc{}
	}

	m.interceptors = append(m.interceptors, interceptor)

	return m
}

// GetMetric used to get metric object by metric_name.
func (m *Monitor) GetMetric(name string) (*Metric, error) {
	metric, ok := m.metrics[name]
	if !ok {
		return nil, ErrMetricNotFound
	}

	return metric, nil
}

// SetMetricPath set metricPath property. metricPath is used for Prometheus
// to get gin server monitoring data.
func (m *Monitor) SetMetricPath(path string) *Monitor {
	m.metricPath = path
	return m
}

// SetExcludePaths set exclude paths which should not be reported (e.g. /ping /healthz...)
func (m *Monitor) SetExcludePaths(paths []string) *Monitor {
	m.excludePaths = paths
	return m
}

// SetSlowTime set slowTime property. slowTime is used to determine whether
// the request is slow. For "gin_slow_request_total" metric.
func (m *Monitor) SetSlowTime(slowTime int32) *Monitor {
	m.slowTime = slowTime
	return m
}

// SetDuration set reqDuration property. reqDuration is used to ginRequestDuration
// metric buckets.
func (m *Monitor) SetDuration(duration []float64) *Monitor {
	m.reqDuration = duration
	return m
}

func (m *Monitor) SetMetricPrefix(prefix string) {
	for _, metric := range m.metrics {
		metric.Name = prefix + metric.Name
	}
}

func (m *Monitor) SetMetricSuffix(suffix string) {
	for _, metric := range m.metrics {
		metric.Name += suffix
	}
}

// AddMetric add custom monitor metric.
func (m *Monitor) AddMetric(metric *Metric) error {
	var err error

	if metric.Name == "" {
		err = errors.New("metric name cannot be empty")
		return err
	}

	_, ok := m.metrics[metric.Name]
	if ok {
		err = errors.Errorf("metric '%s' is existed", metric.Name)
		return err
	}

	setVec, ok := PromptTypeHandler[metric.Type]
	if !ok {
		err = errors.Errorf("metric type '%s' not existed.", metric.Type)
		return err
	}

	err = setVec(metric)
	if err != nil {
		return err
	}

	prometheus.MustRegister(metric.vec)
	m.metrics[metric.Name] = metric

	return err
}
