package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/marketplacemetering"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

const (
	prometheusURL      = "http://localhost:9090"
	productCode        = "STRATIO"
	customerIdentifier = "CUSTOMER"
	cpuDimension       = "CPU"
	memDimension       = "MEM"
	storageDimension   = "STORAGE"
	cpuQuery           = "billing:cpu_usage:last1h"
	memQuery           = "billing:mem_usage:last1h"
	storageQuery       = "billing:storage_usage:last1h"
)

func main() {
	interval := flag.Duration("interval", time.Hour, "Execution interval duration (e.g., 1h)")
	flag.Parse()
	// Run the initial execution
	run()

	// Schedule execution at the specified interval
	ticker := time.NewTicker(*interval)

    for range ticker.C {
        run()
    }

}

func run() {
	promAPI, err := initPrometheusAPI(prometheusURL)
	if err != nil {
		log.Printf("Error creating Prometheus client: %v", err)
		return
	}

	cpuMetricValue, cpuMetricTimestamp, err := getMetric(promAPI, cpuQuery)
	if err != nil {
		log.Printf("Error getting CPU capacity: %v", err)
		return
	}

	memMetricValue, memMetricTimestamp, err := getMetric(promAPI, memQuery)
	if err != nil {
		log.Printf("Error getting MEM capacity: %v", err)
		return
	}

	storageMetricValue, storageMetricTimestamp, err := getMetric(promAPI, storageQuery)
	if err != nil {
		log.Printf("Error getting STORAGE capacity: %v", err)
		return
	}

	fmt.Println(createMeteringRecords(cpuMetricValue, memMetricValue, storageMetricValue, cpuMetricTimestamp, memMetricTimestamp, storageMetricTimestamp))
}

func initPrometheusAPI(prometheusURL string) (v1.API, error) {
	promClient, err := api.NewClient(api.Config{
		Address: prometheusURL,
	})
	if err != nil {
		return nil, err
	}
	return v1.NewAPI(promClient), nil
}

func getMetric(promAPI v1.API, query string) (int64, time.Time, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, warnings, err := promAPI.Query(ctx, query, time.Now(), v1.WithTimeout(5*time.Second))
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("error querying prometheus: %v", err)
	}
	if len(warnings) > 0 {
		fmt.Printf("Warnings: %v\n", warnings)
	}

    vector, ok := result.(model.Vector)
    if !ok || len(vector) == 0 {
        return 0, time.Time{}, fmt.Errorf("metric not found or result is empty")
    }

	metricValue := int64(vector[0].Value)
	metricTimestamp := vector[0].Timestamp.Time()
	return metricValue, metricTimestamp, nil
}

func createMeteringRecords(cpuValue, memValue, storageValue int64, cpuTimestamp, memTimestamp, storageTimestamp time.Time) (marketplacemetering.BatchMeterUsageInput) {
	timezone, _ := time.LoadLocation("UTC")
	meteringRecords := &marketplacemetering.BatchMeterUsageInput{
		ProductCode: aws.String(productCode),
		UsageRecords: []*marketplacemetering.UsageRecord{
			{
				CustomerIdentifier: aws.String(customerIdentifier),
				Dimension:          aws.String(cpuDimension),
				Quantity:           aws.Int64(cpuValue),
				Timestamp:          aws.Time(cpuTimestamp.In(timezone)),
			},
			{
				CustomerIdentifier: aws.String(customerIdentifier),
				Dimension:          aws.String(memDimension),
				Quantity:           aws.Int64(memValue),
				Timestamp:          aws.Time(memTimestamp.In(timezone)),
			},
			{
				CustomerIdentifier: aws.String(customerIdentifier),
				Dimension:          aws.String(storageDimension),
				Quantity:           aws.Int64(storageValue),
				Timestamp:          aws.Time(storageTimestamp.In(timezone)),
			},
		},
	}

	return *meteringRecords
}
