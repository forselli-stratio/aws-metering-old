package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
    PrometheusQueryOperationsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "aws_metering_prometheus_query_operations_total",
            Help: "The total number of query operations to Prometheus.",
        },
        []string{"query"}, // Labels to indicate response status code and query performed
    )
    PrometheusQueryErrorsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "aws_metering_prometheus_query_errors_total",
            Help: "The total number of failed querys to Prometheus.",
        },
        []string{"query"}, // Labels to indicate response status code and query performed
    )
	// New Prometheus metrics for DynamoDB operations
	DynamoDBErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aws_metering_dynamodb_errors_total",
			Help: "The total number of errors in DynamoDB operations.",
		},
        []string{"operation"}, // Labels to indicate response status code and query performed
	)
	DynamoDBOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aws_metering_dynamodb_operations_total",
			Help: "The total number of operations in DynamoDB.",
		},
        []string{"operation"}, // Labels to indicate response status code and query performed
	)
)

func RegisterMetrics() {
    prometheus.MustRegister(PrometheusQueryOperationsTotal)
    prometheus.MustRegister(PrometheusQueryErrorsTotal)
	prometheus.MustRegister(DynamoDBErrorsTotal)
	prometheus.MustRegister(DynamoDBOperationsTotal)
}
