// Package main provides a Go application for metering resource usage and sending metering records to AWS Marketplace Metering Service.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
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
	CustomerIdentifier string
	MetricsEndpoint    string
	ListenAddress      string
	Interval		   time.Duration
}

// Dimension represents a metric dimension and its corresponding Prometheus query.
type Dimension struct {
	Name      string
	PromQuery string
}

func main() {
	// Load configuration from environment variables.
	config, err := loadConfig()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }
    log.Printf("Configuration loaded: %+v", config)

	// Register the metrics endpoint for Prometheus.
	http.Handle(config.MetricsEndpoint, promhttp.Handler())
	go startServer(config.ListenAddress, config.MetricsEndpoint)

	// Initialize DynamoDB client.
	awscli.InitDynamoDB()

	// Register and collect metrics.
	metrics.RegisterMetrics()

    // Schedule repeated execution at the specified interval
	schedule(&config.Interval, config)
}

// loadConfig loads the configurations from environment variables or default values.
func loadConfig() (Config, error) {
    // Retrieve mandatory configuration parameters from environment variables
    prometheusURL := os.Getenv("AWS_METERING_PROMETHEUS_URL")
    customerIdentifier := os.Getenv("AWS_METERING_CUSTOMER_IDENTIFIER")
    interval := os.Getenv("AWS_METERING_INTERVAL")

    // Check if mandatory environment variables are set
    if prometheusURL == "" || customerIdentifier == "" || interval == "" {
        return Config{}, fmt.Errorf("missing required environment variables 'AWS_METERING_PROMETHEUS_URL', 'AWS_METERING_CUSTOMER_IDENTIFIER', or 'AWS_METERING_INTERVAL'")
    }

	// Schedule repeated execution at the specified interval.
	timeInterval, err := time.ParseDuration(interval)
	if err != nil {
        return Config{}, fmt.Errorf("error parsing interval: %v", err)
	}

    // Set defaults for optional parameters if they are not provided
    metricsEndpoint := getEnv("AWS_METERING_METRICS_ENDPOINT", "/metrics")
    listenAddress := getEnv("AWS_METERING_LISTEN_ADDRESS", ":8080")

    config := Config{
        PrometheusURL: prometheusURL,
        CustomerIdentifier: customerIdentifier,
        MetricsEndpoint: metricsEndpoint,
        ListenAddress: listenAddress,
        Interval: timeInterval,
    }

    return config, nil
}

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, fallback string) string {
    if value, exists := os.LookupEnv(key); exists {
        return value
    }
    return fallback
}

// startServer starts the HTTP server for Prometheus metrics.
func startServer(listenAddress, metricsEndpoint string) {
    log.Printf("HTTP server for Prometheus metrics listening on port %s and path %s", listenAddress, metricsEndpoint)
    if err := http.ListenAndServe(listenAddress, nil); err != nil {
        log.Fatalf("HTTP server for Prometheus metrics failed to start: %v", err)
    }
}

func schedule(interval *time.Duration, config Config) {
    // Execute once immediately.
    run(config)

	// Continue with scheduled execution.
	ticker := time.NewTicker(*interval)
	log.Printf("Data will be uploaded to DynamoDB every %v", *interval)
    for range ticker.C {
        run(config)
    }
}

// run performs the main execution logic, fetching metrics and sending metering records to AWS.
func run(config Config) {
	currentTime := time.Now()

	for h := 0; h < 6; h++ { // iterate over the past 6 hours
		// Round down to the nearest 10 minutes for each hour
		roundedCurrentTime := currentTime.Add(-time.Duration(h) * time.Hour).Truncate(10 * time.Minute)

		for i := 1; i < 7; i++ { // 6 intervals per hour excluding the current interval
			// Calculate the start of each 10-minute interval
			intervalStart := roundedCurrentTime.Add(-time.Duration(i) * 10 * time.Minute)
			intervalEnd := intervalStart.Add(10 * time.Minute)

            // Check if there is already a record for this interval in DynamoDB
            if !awscli.CheckIfRecordExists(config.CustomerIdentifier, intervalStart.Unix(), intervalEnd.Unix()) {
                // Run the Prometheus query for this specific hour
                promAPI, err := promcli.InitPrometheusAPI(config.PrometheusURL)
                if err != nil {
                    log.Printf("Error creating Prometheus client: %v", err)
                    continue
                }

                dimensions := []Dimension{
                    {"cpu", "billing:cpu_capacity:last1h"},
                    {"memory", "billing:mem_capacity:last1h"},
                    {"storage", "billing:storage_capacity:last1h"},
                }

                // Using the end time of the natural hour as the timestamp for the query
                meteringRecord := createMeteringRecord(intervalEnd.Unix(), config.CustomerIdentifier, dimensions, promAPI)
                fmt.Println(meteringRecord)
                // Check if dimension_usage is empty or has less than 3 items
                if len(meteringRecord.DimensionUsage) < len(dimensions) {
                    log.Printf("Skipping metering record upload to DynamoDB as dimension_usage field is empty or incomplete.")
                    continue
                }

                // Check if any of the dimension_usage values is 0 (prometheus query returns 0 on error)
                isNullDimension := false
                for _, dimension := range meteringRecord.DimensionUsage {
                    if dimension.Value == 0 {
                        isNullDimension=true
                        break
                    }
                }

                if isNullDimension {
                    log.Printf("Skipping metering record upload to DynamoDB as one or more dimension values are 0.")
                    continue
                }

                // Insert metering record into DynamoDB
                if err := awscli.InsertMeteringRecord(meteringRecord); err != nil {
                    log.Printf("Error inserting metering record into DynamoDB: %v", err)
                    continue
                }
                humanReadableIntervalEnd := intervalEnd.Format("2006-01-02 15:04:05")
                log.Printf("Successfully uploaded record for customer %s to DynamoDB from date: %s", config.CustomerIdentifier, humanReadableIntervalEnd)
            }
		}
	}
}

// createMeteringRecord constructs a metering record directly from the Prometheus query results.
func createMeteringRecord(timestamp int64, customerIdentifier string, dimensions []Dimension, promAPI v1.API) *awscli.MeteringRecord {
	var dimensionUsage []struct{ Dimension string; Value int64 }

	for _, dimension := range dimensions {
		// metricValue, _, err := promcli.RunPromQuery(promAPI, dimension.PromQuery, timestamp)
        _, _, err := promcli.RunPromQuery(promAPI, dimension.PromQuery, timestamp)
		if err != nil {
            humanReadableTimestamp := time.Unix(timestamp, 0).Format("2006-01-02 15:04:05")
			log.Printf("Error querying %s with expression %s and timestamp %s: %v", dimension.Name, dimension.PromQuery, humanReadableTimestamp, err)
		}

		dimensionUsage = append(dimensionUsage, struct{ Dimension string; Value int64 }{
			Dimension: dimension.Name,
            Value:     1,
			// Value:     metricValue,
		})
	}

	return &awscli.MeteringRecord{
		CreateTimestamp:    timestamp,
		CustomerIdentifier: customerIdentifier,
		DimensionUsage:     dimensionUsage,
		MeteringPending:    "true",
	}
}
