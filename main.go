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
)

type Dimension struct {
	name string
	promQuery string
}

var (
	cpuCapacity    = Dimension{"cpu", "billing:cpu_usage:last1h"}
	memCapacity    = Dimension{"memory", "billing:mem_usage:last1h"}
	storageCapacity = Dimension{"storage", "billing:storage_usage:last1h"}
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
	cpuMetricValue, cpuMetricTimestamp, err := prometheus.RunPromQuery(promAPI, cpuCapacity.promQuery)
	if err != nil {
		log.Printf("Error getting CPU capacity: %v", err)
		return
	}

	memMetricValue, memMetricTimestamp, err := prometheus.RunPromQuery(promAPI, memCapacity.promQuery)
	if err != nil {
		log.Printf("Error getting MEM capacity: %v", err)
		return
	}

	storageMetricValue, storageMetricTimestamp, err := prometheus.RunPromQuery(promAPI, storageCapacity.promQuery)
	if err != nil {
		log.Printf("Error getting STORAGE capacity: %v", err)
		return
	}

	fmt.Println(aws.CreateMeteringRecords(productCode, customerIdentifier, cpuMetricValue, memMetricValue, storageMetricValue, cpuMetricTimestamp, memMetricTimestamp, storageMetricTimestamp))
	aws.SendMeteringRecords(aws.CreateMeteringRecords(productCode, customerIdentifier, cpuMetricValue, memMetricValue, storageMetricValue, cpuMetricTimestamp, memMetricTimestamp, storageMetricTimestamp))
}
