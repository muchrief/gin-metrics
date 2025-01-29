package go_apitor

import (
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

type MetricHandlerFunc func(metric *Metric) error

var PromptTypeHandler = map[MetricType]MetricHandlerFunc{
	Counter:   counterHandler,
	Gauge:     gaugeHandler,
	Histogram: histogramHandler,
	Summary:   summaryHandler,
}

func counterHandler(metric *Metric) error {
	metric.vec = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: string(metric.Name), Help: metric.Description},
		metric.Labels,
	)

	return nil
}

func gaugeHandler(metric *Metric) error {
	metric.vec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: string(metric.Name), Help: metric.Description},
		metric.Labels,
	)

	return nil
}

func histogramHandler(metric *Metric) error {
	var err error

	if len(metric.Buckets) == 0 {
		err = errors.Errorf("metric '%s' is histogram type, cannot lose bucket param.", metric.Name)
		return err
	}

	metric.vec = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: string(metric.Name), Help: metric.Description, Buckets: metric.Buckets},
		metric.Labels,
	)

	return err
}

func summaryHandler(metric *Metric) error {
	var err error

	if len(metric.Objectives) == 0 {
		err = errors.Errorf("metric '%s' is summary type, cannot lose objectives param.", metric.Name)
		return err
	}

	metric.vec = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{Name: string(metric.Name), Help: metric.Description, Objectives: metric.Objectives},
		metric.Labels,
	)

	return err
}
