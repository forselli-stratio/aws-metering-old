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

// Constants defining configuration values
const (
	prometheusURL      = "http://localhost:9090"
	productCode        = "STRATIO"
	customerIdentifier = "CUSTOMER"
	metricsEndpoint    = "/metrics"
	listenAddress      = ":8080"
)

// Dimension represents a metric dimension and its corresponding Prometheus query.
type Dimension struct {
	Name      string
	PromQuery string
}

// Predefined dimensions for CPU, memory, and storage capacity.
var (
	cpuCapacity    = Dimension{"cpu", "billing:cpu_lala:last1h"}
	memCapacity    = Dimension{"memory", "billing:mem_capacity:last1h"}
	storageCapacity = Dimension{"storage", "billing:storage_capacity:last1h"}
)

// main function sets up and runs the metering application.
func main() {
	// Parse command-line arguments
	interval := flag.Duration("interval", time.Hour, "Execution interval duration (e.g., 1h)")
	flag.Parse()

	// Register the metrics endpoint for Prometheus
	http.Handle(metricsEndpoint, promhttp.Handler())
	go func() {
		log.Fatal(http.ListenAndServe(listenAddress, nil))
	}()
	metrics.RegisterMetrics()

	// Run the initial execution
	run()

	// Schedule execution at the specified interval
	ticker := time.NewTicker(*interval)
	for range ticker.C {
		run()
	}
}

// run function performs the main execution logic, fetching metrics and sending metering records to AWS.
func run() {
	// Initialize Prometheus API client
	promAPI, err := promcli.InitPrometheusAPI(prometheusURL)
	if err != nil {
		log.Printf("Error creating Prometheus client: %v", err)
		return
	}

	// Fetch usage records for different dimensions
	usageRecords := getCapacityRecords(promAPI, cpuCapacity, memCapacity, storageCapacity)

	// Create metering input
	meterUsageInput := awscli.CreateBatchMeterUsageInput(productCode, customerIdentifier, usageRecords...)
	log.Println("Meter Usage Input:", meterUsageInput)

	// Send metering records to AWS
	_, err = awscli.SendBatchMeterUsageRequest(meterUsageInput)
	if err != nil {
		log.Printf("Error sending metering records: %v", err)
	}
}

// getCapacityRecords function retrieves usage records for specified dimensions from Prometheus.
func getCapacityRecords(promAPI v1.API, dimensions ...Dimension) []*marketplacemetering.UsageRecord {
	var usageRecords []*marketplacemetering.UsageRecord

	for _, dimension := range dimensions {
		// Fetch metric value and timestamp from Prometheus
		metricValue, metricTimestamp, err := promcli.RunPromQuery(promAPI, dimension.PromQuery)
		if err != nil {
			log.Printf("Error getting %s capacity: %v", dimension.Name, err)
			continue
		}

		// Create UsageRecord and add it to the list
		usageRecords = append(usageRecords, &marketplacemetering.UsageRecord{
			Dimension: aws.String(dimension.Name),
			Quantity:  aws.Int64(metricValue),
			Timestamp: aws.Time(metricTimestamp),
		})
	}

	return usageRecords
}