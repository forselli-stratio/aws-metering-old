package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
    RequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "aws_metering_requests_total",
            Help: "The total number of HTTP requests to the AWS metering service.",
        },
        []string{"status"}, // Label to indicate response status code
    )

)

func RegisterMetrics() {
    prometheus.MustRegister(RequestsTotal)
}
