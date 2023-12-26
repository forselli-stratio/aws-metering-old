package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
    PrometheusQuerySuccessesTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "aws_metering_prometheus_query_successes_total",
            Help: "The total number of succeded querys to Prometheus.",
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
	DynamoDBErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "aws_metering_dynamodb_errors_total",
			Help: "The total number of errors in DynamoDB operations.",
		},
	)
	DynamoDBOperationsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "aws_metering_dynamodb_operations_total",
			Help: "The total number of operations in DynamoDB.",
		},
	)
)

func RegisterMetrics() {
    prometheus.MustRegister(PrometheusQuerySuccessesTotal)
    prometheus.MustRegister(PrometheusQueryErrorsTotal)
	prometheus.MustRegister(DynamoDBErrorsTotal)
	prometheus.MustRegister(DynamoDBOperationsTotal)
}
