# aws-metering

https://awsmp-loadforms.s3.amazonaws.com/AWS+Marketplace+-+SaaS+Integration+Guide.pdf

Funcionamiento:

1. Aplicación arranca y envía registros del momento actual obteniendo los datos con query a prometheus
last_prometheus_query_success
last_report_success

2. Comprobamos en la respuesta si existen UnprocessedRecords
unprocessed_records

3. Si existen unprocessedrecords se reintenta su envío.
unprocessed_records_retried

4. Comprobamos en la respuesta siguen existiendo UnprocessedRecords y se actualiza metrica tras reintentos.
unprocessed_records

4. Se ejecuta el mismo ciclo cada x minutos