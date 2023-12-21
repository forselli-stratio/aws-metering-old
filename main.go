// Package main provides a Go application for metering resource usage and sending metering records to AWS Marketplace Metering Service.
package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/marketplacemetering"
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

	// Run the initial execution before scheduling it at regular intervals.
	run(config)

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
	log.Fatal(http.ListenAndServe(listenAddress, nil))
}

func schedule(interval *time.Duration, config Config) {
	ticker := time.NewTicker(*interval)
	for {
		select {
		case <-ticker.C:
			run(config)
		}
	}
}

// run performs the main execution logic, fetching metrics and sending metering records to AWS.
func run(config Config) {
	// Simplified error handling and improved readability.
	currentTimestamp := time.Now().Unix()
	promAPI, err := promcli.InitPrometheusAPI(config.PrometheusURL)
	if err != nil {
		log.Printf("Error creating Prometheus client: %v", err)
		return
	}

	// Predefined dimensions are now initialized here to avoid global state.
	dimensions := []Dimension{
		{"cpu", "billing:cpu_lala:last1h"},
		{"memory", "billing:mem_capacity:last1h"},
		{"storage", "billing:storage_capacity:last1h"},
	}

	usageRecords := getCapacityRecords(promAPI, currentTimestamp, dimensions...)
	meteringRecord := createMeteringRecord(currentTimestamp, config.CustomerIdentifier, usageRecords)

	// Insert metering record into DynamoDB.
	if err := awscli.InsertMeteringRecord(meteringRecord); err != nil {
		log.Printf("Error inserting metering record into DynamoDB: %v", err)
	}
}

// createMeteringRecord constructs a metering record from the usage records.
func createMeteringRecord(timestamp int64, customerIdentifier string, usageRecords []*marketplacemetering.UsageRecord) *awscli.MeteringRecord {
	dimensionUsage := make([]struct{ Dimension string; Value int64 }, len(usageRecords))
	for i, usageRecord := range usageRecords {
		dimensionUsage[i] = struct{ Dimension string; Value int64 }{
			Dimension: aws.StringValue(usageRecord.Dimension),
			Value:     aws.Int64Value(usageRecord.Quantity),
		}
	}
	return &awscli.MeteringRecord{
		CreateTimestamp:   timestamp,
		CustomerIdentifier: customerIdentifier,
		DimensionUsage:     dimensionUsage,
		MeteringPending:    "true",
	}
}

// getCapacityRecords retrieves usage records for specified dimensions from Prometheus.
func getCapacityRecords(promAPI v1.API, timestamp int64, dimensions ...Dimension) []*marketplacemetering.UsageRecord {
	var usageRecords []*marketplacemetering.UsageRecord
	for _, dimension := range dimensions {
		metricValue, _, err := promcli.RunPromQuery(promAPI, dimension.PromQuery, timestamp)
		if err != nil {
			log.Printf("Error getting %s capacity: %v", dimension.Name, err)
			continue
		}

		usageRecords = append(usageRecords, &marketplacemetering.UsageRecord{
			Dimension: aws.String(dimension.Name),
			Quantity:  aws.Int64(metricValue),
			Timestamp: aws.Time(time.Unix(timestamp, 0)),
		})
	}
	return usageRecords
}
