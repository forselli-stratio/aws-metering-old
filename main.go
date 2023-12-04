package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/marketplacemetering"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

const (
    prometheusURL       = "http://localhost:9090"
	productCode     = "STRATIO"
	customerIdentifier = "YOUR_CUSTOMER_IDENTIFIER"
	dimensionName    = "CPU"
)

func main() {

    // Initialize Prometheus client
    promClient, err := api.NewClient(api.Config{
        Address: prometheusURL,
    })
    if err != nil {
        log.Fatal("Error creating Prometheus client: ", err)
    }

    // Create Prometheus API client
    promAPI := v1.NewAPI(promClient)

    // Get CPU capacity metric from Prometheus
    cpuCapacity, err := getCPUCapacity(promAPI)
    if err != nil {
        log.Fatal("Error getting CPU capacity: ", err)
    }

	metricValue := float64(cpuCapacity[0].Value)
	metricTimestamp := cpuCapacity[0].Timestamp.Time()


    sendMeteringRecords(metricValue, metricTimestamp)
}

func getCPUCapacity(promAPI v1.API) (model.Vector, error) {
    // Query Prometheus for cpu_usage in the last hour
    query := `billing:cpu_usage:last1h`
    result, warnings, err := promAPI.Query(context.Background(), query, time.Now())
    if err != nil {
        fmt.Printf("Error querying Prometheus: %v\n", err)
        os.Exit(1)
    }
    if len(warnings) > 0 {
        fmt.Printf("Warnings: %v\n", warnings)
    }

	cpuCapacity := result.(model.Vector)
    return cpuCapacity, nil
}

func sendMeteringRecords(value float64, timestamp time.Time) {
	meteringRecords := &marketplacemetering.BatchMeterUsageInput{
		ProductCode: aws.String(productCode), // Required
		UsageRecords: []*marketplacemetering.UsageRecord{ // Required
			{ // Required
				CustomerIdentifier: aws.String("CustomerIdentifier"), // Required
				Dimension:          aws.String("UsageDimension"),     // Required
				Quantity:           aws.Int64(1),                     // Required
				Timestamp:          aws.Time(timestamp),             // Required
			},
		},
	}

    fmt.Println(meteringRecords)

}