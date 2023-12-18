package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	awscli "github.com/forselli-stratio/aws-metering/pkg/aws"
	"github.com/forselli-stratio/aws-metering/pkg/metrics"
	promcli "github.com/forselli-stratio/aws-metering/pkg/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	// Register the metrics endpoint
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Fatal(http.ListenAndServe(":8080", nil))
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

func run() {
	promAPI, err := promcli.InitPrometheusAPI(prometheusURL)
	if err != nil {
		log.Printf("Error creating Prometheus client: %v", err)
		return
	}
	cpuMetricValue, cpuMetricTimestamp, err := promcli.RunPromQuery(promAPI, cpuCapacity.promQuery)
	if err != nil {
		log.Printf("Error getting CPU capacity: %v", err)
		return
	}

	memMetricValue, memMetricTimestamp, err := promcli.RunPromQuery(promAPI, memCapacity.promQuery)
	if err != nil {
		log.Printf("Error getting MEM capacity: %v", err)
		return
	}

	storageMetricValue, storageMetricTimestamp, err := promcli.RunPromQuery(promAPI, storageCapacity.promQuery)
	if err != nil {
		log.Printf("Error getting STORAGE capacity: %v", err)
		return
	}

	fmt.Println(awscli.CreateMeteringRecords(productCode, customerIdentifier, cpuMetricValue, memMetricValue, storageMetricValue, cpuMetricTimestamp, memMetricTimestamp, storageMetricTimestamp))
	awscli.SendMeteringRecords(awscli.CreateMeteringRecords(productCode, customerIdentifier, cpuMetricValue, memMetricValue, storageMetricValue, cpuMetricTimestamp, memMetricTimestamp, storageMetricTimestamp))
}
