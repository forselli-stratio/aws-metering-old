// Package prometheus provides utility functions for interacting with Prometheus API and executing queries.
package prometheus

import (
	"context"
	"fmt"
	"time"

	"github.com/forselli-stratio/aws-metering/pkg/metrics"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// InitPrometheusAPI initializes a Prometheus API client using the specified Prometheus server URL.
// It returns a Prometheus API client and an error if the initialization fails.
func InitPrometheusAPI(prometheusURL string) (v1.API, error) {
	promClient, err := api.NewClient(api.Config{
		Address: prometheusURL,
	})
	if err != nil {
		return nil, err
	}
	return v1.NewAPI(promClient), nil
}

// RunPromQuery executes a Prometheus query using the provided Prometheus API client and query string.
// It returns the metric value, timestamp, and an error if the query execution encounters any issues.
func RunPromQuery(promAPI v1.API, query string, timestamp int64) (int64, time.Time, error) {
	// Set up a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Execute Prometheus query with the provided timestamp
	result, warnings, err := promAPI.Query(ctx, query, time.Unix(timestamp, 0), v1.WithTimeout(5*time.Second))
	// Increment Prometheus query success metric
	metrics.PrometheusQueryOperationsTotal.WithLabelValues(query).Inc()
	if err != nil {
		// Increment Prometheus query error metric
		metrics.PrometheusQueryErrorsTotal.WithLabelValues(query).Inc()
		return 0, time.Time{}, fmt.Errorf("error querying Prometheus: %v", err)
	}

	// Print any warnings
	if len(warnings) > 0 {
		fmt.Printf("Warnings: %v\n", warnings)
	}

	// Parse the result as a vector
	vector, ok := result.(model.Vector)
	if !ok || len(vector) == 0 {
		// Increment Prometheus query error metric for metric not found or empty result
		metrics.PrometheusQueryErrorsTotal.WithLabelValues(query).Inc()
		return 0, time.Time{}, fmt.Errorf("metric not found or result is empty")
	}

	// Extract metric value and timestamp
	metricValue := int64(vector[0].Value)
	metricTimestamp := vector[0].Timestamp.Time()

	return metricValue, metricTimestamp, nil
}