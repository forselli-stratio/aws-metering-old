package prometheus

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)


func InitPrometheusAPI(prometheusURL string) (v1.API, error) {
	promClient, err := api.NewClient(api.Config{
		Address: prometheusURL,
	})
	if err != nil {
		return nil, err
	}
	return v1.NewAPI(promClient), nil
}

func GetMetric(promAPI v1.API, query string) (int64, time.Time, error) {
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