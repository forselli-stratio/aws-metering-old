// Package main provides a Go application for metering resource usage and sending metering records to AWS Marketplace Metering Service.
package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	awscli "github.com/forselli-stratio/aws-metering/pkg/aws"
	"github.com/forselli-stratio/aws-metering/pkg/metrics"
	promcli "github.com/forselli-stratio/aws-metering/pkg/prometheus"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Configurations are moved to a struct to avoid global state and improve testability.
type Config struct {
	PrometheusURL      string
	ProductCode        string
	CustomerIdentifier string
	MetricsEndpoint    string
	ListenAddress      string
}

// Dimension represents a metric dimension and its corresponding Prometheus query.
type Dimension struct {
	Name      string
	PromQuery string
}

func main() {
	// Configuration is now read from a function, making it easier to manage.
	config := loadConfig()
	log.Printf("Configuration loaded: %+v", config)

	// Parse command-line arguments.
	interval := flag.Duration("interval", time.Hour, "Execution interval duration (e.g., 1h)")
	flag.Parse()

	// Initialize DynamoDB client.
	awscli.InitDynamoDB()

	// Register the metrics endpoint for Prometheus.
	http.Handle(config.MetricsEndpoint, promhttp.Handler())
	go startServer(config.ListenAddress)

	// Register and collect metrics.
	metrics.RegisterMetrics()

	// Schedule repeated execution at the specified interval.
	schedule(interval, config)
}

// loadConfig loads the configurations from environment variables or default values.
func loadConfig() Config {
	return Config{
		PrometheusURL:      "http://localhost:9090",
		ProductCode:        "STRATIO",
		CustomerIdentifier: "CUSTOMER",
		MetricsEndpoint:    "/metrics",
		ListenAddress:      ":8080",
	}
}

// startServer starts the HTTP server for Prometheus metrics.
func startServer(listenAddress string) {
    log.Printf("HTTP server listening on %s", listenAddress)
    if err := http.ListenAndServe(listenAddress, nil); err != nil {
        log.Fatalf("HTTP server failed to start: %v", err)
    }
}

func schedule(interval *time.Duration, config Config) {
	ticker := time.NewTicker(*interval)
	log.Printf("Data will be uploaded to DynamoDB every %v", *interval)
    for range ticker.C {
        run(config)
    }
}

// run performs the main execution logic, fetching metrics and sending metering records to AWS.
func run(config Config) {
	currentTimestamp := time.Now().Unix()
	promAPI, err := promcli.InitPrometheusAPI(config.PrometheusURL)
	if err != nil {
		log.Printf("Error creating Prometheus client: %v", err)
		return
	}

	// Predefined dimensions are now initialized here to avoid global state.
	dimensions := []Dimension{
		{"cpu", "billing:cpu_capacity:last1h"},
		{"memory", "billing:mem_capacity:last1h"},
		{"storage", "billing:storage_capacity:last1h"},
	}

	meteringRecord := createMeteringRecord(currentTimestamp, config.CustomerIdentifier, dimensions, promAPI)

	// Insert metering record into DynamoDB.
	if err := awscli.InsertMeteringRecord(meteringRecord); err != nil {
		log.Printf("Error inserting metering record into DynamoDB: %v", err)
	}
}

// createMeteringRecord constructs a metering record directly from the Prometheus query results.
func createMeteringRecord(timestamp int64, customerIdentifier string, dimensions []Dimension, promAPI v1.API) *awscli.MeteringRecord {
	var dimensionUsage []struct{ Dimension string; Value int64 }

	for _, dimension := range dimensions {
		metricValue, _, err := promcli.RunPromQuery(promAPI, dimension.PromQuery, timestamp)
		if err != nil {
			log.Printf("Error querying %s with %s: %v", dimension.Name, dimension.PromQuery, err)
			continue
		}

		dimensionUsage = append(dimensionUsage, struct{ Dimension string; Value int64 }{
			Dimension: dimension.Name,
			Value:     metricValue,
		})
	}
	return &awscli.MeteringRecord{
		CreateTimestamp:    timestamp,
		CustomerIdentifier: customerIdentifier,
		DimensionUsage:     dimensionUsage,
		MeteringPending:    "true",
	}
}
