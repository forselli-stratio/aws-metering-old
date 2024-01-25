# aws-metering

This Go application periodically queries Prometheus for resource usage metrics, constructs metering records, and uploads them to an AWS DynamoDB table.

## Configuration

| Environment Variable               | Description                                                | Example Value                     |
|-------------------------------------|------------------------------------------------------------|-----------------------------------|
| AWS_METERING_PROMETHEUS_URL         | Prometheus server URL                                      | http://your-prometheus-server:9090 |
| AWS_METERING_CUSTOMER_IDENTIFIER   | Customer identifier for AWS Marketplace Metering Service    | your-customer-identifier          |
| AWS_METERING_INTERVAL              | Time interval for data collection (e.g., "10m" for 10 minutes) | 10m                             |

Optional Environment Variables:

| Environment Variable               | Description                                              | Default Value  |
|------------------------------------|----------------------------------------------------------|----------------|
| AWS_METERING_METRICS_ENDPOINT      | Metrics endpoint path for Prometheus metrics             | /metrics       |
| AWS_METERING_LISTEN_ADDRESS        | Address for the HTTP server for Prometheus metrics         | :8080          |
