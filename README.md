# aws-metering

https://awsmp-loadforms.s3.amazonaws.com/AWS+Marketplace+-+SaaS+Integration+Guide.pdf

Funcionamiento:

1. Aplicación arranca y envía registros del momento actual y de las ultimas 6 horas obteniendo los datos con query a prometheus (si existen)
 No existe problema en reenviar los datos de las ultimas 6 horas ya que

2. Comprobamos en la respuesta si existen UnprocessedRecords

3. Si existen unprocessedrecords se reintenta su envío.

4. Comprobamos en la respuesta siguen existiendo UnprocessedRecords y se actualiza metrica tras reintentos.

5. Se ejecuta el mismo ciclo cada x minutos

## Pendiente

- Gestion de retries y errores

- Obtener ultimo registro enviado


## Requisitos para probar con AWS Marketplace Metering Service API

- Producto estado limited
https://docs.aws.amazon.com/marketplace/latest/userguide/saas-integration-metering-and-entitlement-apis.html

- Precio $0.01
https://docs.aws.amazon.com/marketplace/latest/userguide/saas-create-product-page.html

- Usuario con una politica con permisos sobre la API de metering y entitlement del AWS Marketplace
(https://docs.aws.amazon.com/marketplace/latest/userguide/iam-user-policy-for-aws-marketplace-actions.html)

- Una cuenta whitelisteada en el producto en estado limited con la que probar suscribirnos al producto para disponer de un CustomerIdentifier.
(Lastly, applications start off as “limited” (meaning they are not visible publicly) to allow for development. As such, the “Accounts to allow-list” must be populated with AWS Account IDs you plan to “test subscribe” to the application. Only these accounts will be able to access the application while the application is in “limited” state:)

##

- Recomendado enviar metricas una vez por hora (es posible enviar en intervalos mayores, pero habria que continuar enviando un valor de 0 el resto de horas)

- Se pueden enviar de batches de a 25.

- La responsabilidad de de asegurarse de que los records son enviados y recibidos es nuestra. Se puede usar AWS Cloudtrail. https://docs.aws.amazon.com/marketplace/latest/userguide/cloudtrail-logging.html

- Se deduplican las metering requests de cada hora per product/customer/hour/dimension

- Se puede reintentar toda request, pero si se modifica la cantidad, se mantiene la original

- Si se envian multiples requests del mismo customer/dimension/hour, los records no se agregan

- Se pueden enviar records con un timestamp de hasta 6 horas hacia el pasado.

- Una vez el cliente se da de baja, hay 1 hora para enviar cualquier record pendiente.

- El payload de la request no puede superar 1MB
