package go_apitor

import (
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

var ErrMetricNotFound = errors.New("metric not found")
var ErrInvalidMetricName = errors.New("invalid metric name")
var ErrInvalidMetricType = errors.New("invalid metric type")

type MetricType string

const (
	Counter   MetricType = "counter"
	Gauge     MetricType = "gauge"
	Histogram MetricType = "histogram"
	Summary   MetricType = "summary"
)

func (t MetricType) EnumList() []string {
	return []string{
		"counter",
		"gauge",
		"histogram",
		"summary",
	}
}

func (t MetricType) IsValid() bool {
	values := t.EnumList()
	for _, value := range values {
		if string(t) == value {
			return true
		}
	}

	return false
}

// Metric defines a metric object. Users can use it to save
// metric data. Every metric should be globally unique by name.
type Metric struct {
	Type        MetricType
	Name        string
	Description string
	Labels      []string
	Buckets     []float64
	Objectives  map[float64]float64

	vec prometheus.Collector
}

func NewMetric(metricType MetricType, name string) *Metric {
	return &Metric{
		Type: metricType,
		Name: name,
	}
}

func (m *Metric) SetDescription(desc string) *Metric {
	m.Description = desc
	return m
}

func (m *Metric) SetLabels(labels ...string) *Metric {
	if m.Labels == nil {
		m.Labels = []string{}
	}

	m.Labels = append(m.Labels, labels...)

	return m
}

func (m *Metric) SetObjectives(objectives map[float64]float64) *Metric {
	m.Objectives = objectives
	return m
}

// SetGaugeValue set data for Gauge type Metric.
func (m *Metric) SetGaugeValue(labelValues []string, value float64) error {
	var err error

	if m.Type.IsValid() {
		return ErrInvalidMetricType
	}

	if m.Type != Gauge {
		err = errors.Errorf("metric '%s' not Gauge type", m.Name)
		return err
	}

	vec, ok := m.vec.(*prometheus.GaugeVec)
	if !ok {
		return ErrInvalidMetricType
	}

	vec.WithLabelValues(labelValues...).Set(value)
	return nil
}

// Inc increases value for Counter/Gauge type metric, increments
// the counter by 1
func (m *Metric) Inc(labelValues []string) error {
	var err error

	if m.Type.IsValid() {
		return ErrInvalidMetricType
	}

	switch m.Type {
	case Counter:
		vec, ok := m.vec.(*prometheus.CounterVec)
		if !ok {
			return ErrInvalidMetricType
		}

		vec.WithLabelValues(labelValues...).Inc()

	case Gauge:
		vec, ok := m.vec.(*prometheus.GaugeVec)
		if !ok {
			return ErrInvalidMetricType
		}

		vec.WithLabelValues(labelValues...).Inc()

	default:
		err = errors.Errorf("metric '%s' not Gauge or Counter type", m.Name)
		return err
	}

	return err
}

// Add adds the given value to the Metric object. Only
// for Counter/Gauge type metric.
func (m *Metric) Add(labelValues []string, value float64) error {
	var err error

	if m.Type.IsValid() {
		return ErrInvalidMetricType
	}

	switch m.Type {
	case Counter:
		vec, ok := m.vec.(*prometheus.CounterVec)
		if !ok {
			return ErrInvalidMetricType
		}

		vec.WithLabelValues(labelValues...).Add(value)

	case Gauge:
		vec, ok := m.vec.(*prometheus.GaugeVec)
		if !ok {
			return ErrInvalidMetricType
		}

		vec.WithLabelValues(labelValues...).Add(value)

	default:
		err = errors.Errorf("metric '%s' not Gauge or Counter type", m.Name)
		return err
	}

	return err
}

// Observe is used by Histogram and Summary type metric to
// add observations.
func (m *Metric) Observe(labelValues []string, value float64) error {
	var err error

	if m.Type.IsValid() {
		return ErrInvalidMetricType
	}

	switch m.Type {
	case Histogram:
		vec, ok := m.vec.(*prometheus.HistogramVec)
		if !ok {
			return ErrInvalidMetricType
		}

		vec.WithLabelValues(labelValues...).Observe(value)
	case Summary:
		vec, ok := m.vec.(*prometheus.SummaryVec)
		if !ok {
			return ErrInvalidMetricType
		}

		vec.WithLabelValues(labelValues...).Observe(value)
	default:
		err = errors.Errorf("metric '%s' not Histogram or Summary type", m.Name)
		return err
	}

	return err
}
