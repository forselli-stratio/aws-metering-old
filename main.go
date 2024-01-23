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
	go startServer(config.ListenAddress)

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
    prometheusURL := os.Getenv("PROMETHEUS_URL")
    customerIdentifier := os.Getenv("CUSTOMER_IDENTIFIER")
    interval := os.Getenv("INTERVAL")

    // Check if mandatory environment variables are set
    if prometheusURL == "" || customerIdentifier == "" || interval == "" {
        return Config{}, fmt.Errorf("missing required environment variables 'PROMETHEUS_URL', 'CUSTOMER_IDENTIFIER', or 'INTERVAL'")
    }

	// Schedule repeated execution at the specified interval.
	timeInterval, err := time.ParseDuration(interval)
	if err != nil {
        return Config{}, fmt.Errorf("error parsing interval: %v", err)
	}

    // Set defaults for optional parameters if they are not provided
    metricsEndpoint := getEnv("METRICS_ENDPOINT", "/metrics")
    listenAddress := getEnv("LISTEN_ADDRESS", ":8080")

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
func startServer(listenAddress string) {
    log.Printf("HTTP server listening on %s", listenAddress)
    if err := http.ListenAndServe(listenAddress, nil); err != nil {
        log.Fatalf("HTTP server failed to start: %v", err)
    }
}

func schedule(interval *time.Duration, config Config) {
    // Execute once immediately.
    log.Printf("Uploading data to DynamoDB.")
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
    // Round down to the nearest hour
    roundedCurrentTime := currentTime.Truncate(time.Hour)

    for i := 1; i <= 6; i++ {
        // Calculate the start of each past natural hour
        naturalHourStart := roundedCurrentTime.Add(-time.Duration(i) * time.Hour)
        naturalHourEnd := naturalHourStart.Add(time.Hour).Unix()

        // Check if there is already a record for this interval in DynamoDB
        if !awscli.CheckIfRecordExists(config.CustomerIdentifier, naturalHourStart.Unix(), naturalHourEnd) {
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
            meteringRecord := createMeteringRecord(naturalHourEnd, config.CustomerIdentifier, dimensions, promAPI)
            fmt.Println(meteringRecord)
            // Check if dimension_usage is empty or has less than 3 items
            if len(meteringRecord.DimensionUsage) < len(dimensions) {
                log.Printf("Skipping metering record upload to DynamoDB as dimension_usage is empty or incomplete.")
            }

            // Insert metering record into DynamoDB
            if err := awscli.InsertMeteringRecord(meteringRecord); err != nil {
                metrics.DynamoDBErrorsTotal.Inc()
                log.Printf("Error inserting metering record into DynamoDB: %v", err)
                return
            }
            log.Printf("Successfully uploaded record for customer %s to DynamoDB from date: %d", config.CustomerIdentifier, naturalHourEnd)
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
