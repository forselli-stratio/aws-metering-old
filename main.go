package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	aws "github.com/forselli-stratio/aws-metering/pkg/aws"
	prometheus "github.com/forselli-stratio/aws-metering/pkg/prometheus"
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
	promAPI, err := prometheus.InitPrometheusAPI(prometheusURL)
	if err != nil {
		log.Printf("Error creating Prometheus client: %v", err)
		return
	}

	cpuMetricValue, cpuMetricTimestamp, err := prometheus.GetMetric(promAPI, cpuQuery)
	if err != nil {
		log.Printf("Error getting CPU capacity: %v", err)
		return
	}

	memMetricValue, memMetricTimestamp, err := prometheus.GetMetric(promAPI, memQuery)
	if err != nil {
		log.Printf("Error getting MEM capacity: %v", err)
		return
	}

	storageMetricValue, storageMetricTimestamp, err := prometheus.GetMetric(promAPI, storageQuery)
	if err != nil {
		log.Printf("Error getting STORAGE capacity: %v", err)
		return
	}

	fmt.Println(aws.CreateMeteringRecords(productCode, customerIdentifier, cpuMetricValue, memMetricValue, storageMetricValue, cpuMetricTimestamp, memMetricTimestamp, storageMetricTimestamp))
}
