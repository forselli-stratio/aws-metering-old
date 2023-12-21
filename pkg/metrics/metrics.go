package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
    RequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "aws_metering_batchmeterusage_requests_total",
            Help: "The total number of HTTP requests to the AWS metering service.",
        },
        []string{"status"}, // Label to indicate response status code
    )
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
)

func RegisterMetrics() {
    prometheus.MustRegister(RequestsTotal)
    prometheus.MustRegister(PrometheusQuerySuccessesTotal)
    prometheus.MustRegister(PrometheusQueryErrorsTotal)
}
