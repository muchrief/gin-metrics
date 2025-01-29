package go_apitor

import (
	"fmt"
	"slices"
	"strconv"
	"time"
)

var (
	metricRequestTotal    = "request_total"
	metricRequestUVTotal  = "request_uv_total"
	metricURIRequestTotal = "uri_request_total"
	metricRequestBody     = "request_body_total"
	metricResponseBody    = "response_body_total"
	metricRequestDuration = "request_duration"
	metricSlowRequest     = "slow_request_total"
)

func (m *Monitor) RegisterDefaultMetrics() error {
	var err error

	err = m.AddMetric(&Metric{
		Type:        Counter,
		Name:        metricRequestTotal,
		Description: "all the server received request num.",
		Labels:      m.getMetricLabelsIncludingMetadata(metricRequestTotal),
	})
	if err != nil {
		return err
	}

	err = m.AddMetric(&Metric{
		Type:        Counter,
		Name:        metricRequestUVTotal,
		Description: "all the server received ip num.",
		Labels:      m.getMetricLabelsIncludingMetadata(metricRequestUVTotal),
	})
	if err != nil {
		return err
	}

	err = m.AddMetric(&Metric{
		Type:        Counter,
		Name:        metricURIRequestTotal,
		Description: "all the server received request num with every uri.",
		Labels:      m.getMetricLabelsIncludingMetadata(metricURIRequestTotal),
	})
	if err != nil {
		return err
	}

	err = m.AddMetric(&Metric{
		Type:        Counter,
		Name:        metricRequestBody,
		Description: "the server received request body size, unit byte",
		Labels:      m.getMetricLabelsIncludingMetadata(metricRequestBody),
	})
	if err != nil {
		return err
	}

	err = m.AddMetric(&Metric{
		Type:        Counter,
		Name:        metricResponseBody,
		Description: "the server send response body size, unit byte",
		Labels:      m.getMetricLabelsIncludingMetadata(metricResponseBody),
	})
	if err != nil {
		return err
	}

	err = m.AddMetric(&Metric{
		Type:        Histogram,
		Name:        metricRequestDuration,
		Description: "the time server took to handle the request.",
		Labels:      m.getMetricLabelsIncludingMetadata(metricRequestDuration),
		Buckets:     m.reqDuration,
	})
	if err != nil {
		return err
	}

	err = m.AddMetric(&Metric{
		Type:        Counter,
		Name:        metricSlowRequest,
		Description: fmt.Sprintf("the server handled slow requests counter, t=%d.", m.slowTime),
		Labels:      m.getMetricLabelsIncludingMetadata(metricSlowRequest),
	})
	if err != nil {
		return err
	}

	return err
}

func (m *Monitor) includesMetadata() bool {
	return len(m.metadata) > 0
}

func (m *Monitor) getMetadata() ([]string, []string) {
	metadata_labels := []string{}
	metadata_values := []string{}

	for v := range m.metadata {
		metadata_labels = append(metadata_labels, v)
		metadata_values = append(metadata_values, m.metadata[v])
	}

	return metadata_labels, metadata_values
}

func (m *Monitor) getMetricLabelsIncludingMetadata(metricName string) []string {
	includes_metadata := m.includesMetadata()
	metadata_labels, _ := m.getMetadata()

	switch metricName {
	case metricRequestDuration:
		metric_labels := []string{"uri"}
		if includes_metadata {
			metric_labels = append(metric_labels, metadata_labels...)
		}
		return metric_labels

	case metricURIRequestTotal:
		metric_labels := []string{"uri", "method", "code"}
		if includes_metadata {
			metric_labels = append(metric_labels, metadata_labels...)
		}
		return metric_labels

	case metricSlowRequest:
		metric_labels := []string{"uri", "method", "code"}
		if includes_metadata {
			metric_labels = append(metric_labels, metadata_labels...)
		}
		return metric_labels

	default:
		var metric_labels []string = nil
		if includes_metadata {
			metric_labels = metadata_labels
		}
		return metric_labels
	}
}

func (m *Monitor) InterceptorHandler(
	start time.Time,
	clientIP,
	method,
	path string,
	status,
	contentLength,
	responseSize int,
) error {
	var err error

	if path == m.metricPath || slices.Contains(m.excludePaths, path) {
		return err
	}

	var metric_values []string = []string{}
	p := NewParalelAction()

	// set request total
	p.Add(func() error {
		metric, err := m.GetMetric(metricRequestTotal)
		if err != nil {
			return err
		}

		return metric.Inc(m.getMetricValues(metric_values))
	})

	// set uv
	if !m.bloomFilter.Contains(clientIP) {
		p.Add(func() error {
			m.bloomFilter.Add(clientIP)
			metric_values = nil
			metric, err := m.GetMetric(metricRequestUVTotal)
			if err != nil {
				return err
			}

			return metric.Inc(m.getMetricValues(metric_values))
		})
	}

	metric_values = []string{path, method, strconv.Itoa(status)}

	// set uri request total
	p.Add(func() error {
		metric, err := m.GetMetric(metricURIRequestTotal)
		if err != nil {
			return err
		}

		return metric.Inc(m.getMetricValues(metric_values))
	})

	// set request body size
	// since r.ContentLength can be negative (in some occasions) guard the operation
	if contentLength >= 0 {
		p.Add(func() error {
			metric_values = nil
			metric, err := m.GetMetric(metricRequestBody)
			if err != nil {
				return err
			}

			return metric.Add(m.getMetricValues(metric_values), float64(contentLength))
		})
	}

	// set slow request
	latency := time.Since(start)
	if int32(latency.Seconds()) > m.slowTime {
		p.Add(func() error {
			metric_values = []string{path, method, strconv.Itoa(status)}
			metric, err := m.GetMetric(metricSlowRequest)
			if err != nil {
				return err
			}

			return metric.Inc(m.getMetricValues(metric_values))
		})
	}

	// set request duration
	metric_values = []string{path}
	p.Add(func() error {
		metric, err := m.GetMetric(metricRequestDuration)
		if err != nil {
			return err
		}

		return metric.Observe(m.getMetricValues(metric_values), latency.Seconds())
	})

	// set response size
	if responseSize > 0 {
		p.Add(func() error {
			metric_values = nil
			metric, err := m.GetMetric(metricResponseBody)
			if err != nil {
				return err
			}

			return metric.Add(m.getMetricValues(metric_values), float64(responseSize))
		})
	}

	err = p.Wait()
	if err != nil {
		return err
	}

	return err
}

func (m *Monitor) getMetricValues(metric_values []string) []string {
	includes_metadata := m.includesMetadata()
	_, metadata_values := m.getMetadata()
	if includes_metadata {
		metric_values = append(metric_values, metadata_values...)
	}

	return metric_values
}
