# Registro de Cambios (Changelog)

Todos los cambios notables en este proyecto se documentarán en este archivo.

El formato se basa en [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/)
y este proyecto se adhiere a [Semantic Versioning](https://semver.org/lang/es/).
## [0.12.0] - 2026-10-24

### Agregado
- **Trazabilidad Forense Enriquecida (Task 11):**
  - Enriquecimiento de `GET /v1/payments/{id}` en `payments-api` mediante `GetPaymentIntentByID` para retornar los datos financieros de la orden comercial (`order_reference`, `amount`, `currency`).
  - Consolidación de metadatos de auditoría forense (`from_status`, `to_status`, `code`, `message`, `latency_ms`, `acquirer_attempt`, `correlation_id`) en los eventos de `transaction_events` expuestos al consumidor.
- **Mejoras de Experiencia de Usuario (UX) en Web Dashboard:**
  - Componente `CopyButton` para copiado en 1 clic de IDs de transacción, claves de idempotencia, referencias de orden, correlation IDs y números de tarjeta enmascarados, con micro-feedback interactivo.
  - `EventTimeline` rediseñado con transiciones de estado explícitas (`from_status` → `to_status`), insignias cromáticas de diagnóstico, latencia de red bancaria en milisegundos y badges de correlación.
  - Banner de orden comercial en `/payments/[id]` y botón de acción rápida *"Probar en Sandbox con estos datos"* para precarga instantánea en el simulador mediante parámetros URL.
  - Alerta de precarga interactiva en `/sandbox` con opción de limpiar o probar de inmediato.
- **Síntesis de Documentación Maestra:**
  - `README.md`: Incorporación de diagrama de componentes de arquitectura del MVP (Sección 4), síntesis de supuestos y decisiones técnicas (Sección 4.2), compactación de tabla de campos especiales (Sección 4.3) y rediseño lineal del diagrama Mermaid de arquitectura objetivo (Sección 8).
  - Limpieza de documentación: consolidación final de deuda técnica y eliminación de backlog temporal al encontrarse 100% resuelto.

## [0.11.0] - 2026-10-24

### Agregado
- **Documentación Maestra Pitch-Ready (Task 10):** Redacción completa de `README.md` maestro explicando la arquitectura, decisiones técnicas, modelo de base de datos relacional y supuestos de ingeniería.
- **Supuestos Clave y Decisiones Técnicas:** Explicación del flujo de checkout síncrono e interactivo, seguridad de tarjetas exclusivamente en RAM volátil (PCI-DSS), validaciones tempranas anti-carding en umbral HTTP (Luhn/expiración en memoria), garantía de no doble cobro en 3 niveles, reintentos selectivos y reconciliación on-demand.
- **Modelo Relacional en PostgreSQL:** Justificación del desacople 1:N entre `payment_intents` (órdenes comerciales) y `transactions` (intentos técnicos de cobro), aislamiento multi-tenant con `merchants` y garantía física mediante índices parciales únicos (`uq_tx_one_approved`).
- **Plan de Escalabilidad y Evolución Asíncrona:** Visión a futuro que contempla balanceo stateless horizontal, réplicas de lectura, circuit breakers y evolución asíncrona mediante colas (RabbitMQ/Kafka) para facturación diferida y cobro recurrente de suscripciones por lotes (*batch*).
- **Pragmatismo de Monorepo y Trazabilidad AI:** Argumentación de la elección de monorepo para MVP y declaración transparente del uso de asistentes de IA en pair programming asistido, referenciando los artefactos de diseño en `docs/` (`prd.md`, `technical-design.md`, `tasks/`).

## [0.10.0] - 2026-10-24

### Agregado
- **Orquestación Docker Compose Unificada (Task 09):** Archivo `docker-compose.yml` que levanta la arquitectura completa con un solo comando (`postgres:16-alpine`, `acquirer-mock`, `payments-api` y `web-dashboard`), con red unificada (`haulmer-payments-net`), volúmenes persistentes (`haulmer-postgres-data`), dependencias condicionales (`service_healthy`) y plantilla de entorno en `.env.example`.
- **Automatización & Makefile Integral:** Interfaz de control estandarizada en `Makefile` con comandos limpios para ciclo de vida (`up`, `down`, `logs`, `status`), pruebas (`test`, `test-e2e`), análisis de calidad y complejidad (`complexity`), escaneo de vulnerabilidades (`security`), análisis estático (`lint`) y formateo (`format`).
- **Control de Complejidad Ciclomática y Cognitiva:** Verificación estricta de mantenibilidad de código con `gocyclo` (complejidad ciclomática <= 15) y `gocognit` (complejidad cognitiva <= 20) en servicios Go.
- **Pipeline de Integración Continua (CI):** Flujo de GitHub Actions en `.github/workflows/ci.yml` con trabajos paralelos optimizados (`backend-ci` y `frontend-ci`) que ejecutan tests con `-race`, chequeos de complejidad, linters y escaneo de vulnerabilidades sin sobrecosto de Docker.
- **Suite Automatizada de Pruebas E2E (`scripts/test-e2e.sh`):** Cobertura determinista de los 7 escenarios bancarios clave contra el entorno orquestado: pagos aprobados, rechazo por fondos insuficientes (HTTP 402), reintentos automáticos ante 503, replay idempotente idéntico (HTTP 200), colisión de clave de idempotencia (HTTP 409), garantía física anti-doble cobro en orden ya pagada (HTTP 409 `intent_already_approved`), y aislamiento multi-tenant.
- **Escaneo de Seguridad & CVEs:** Auditoría de vulnerabilidades con `govulncheck` en dependencias Go y `npm audit` en el panel web, garantizando 0 vulnerabilidades conocidas.
- **Documentación Operativa:** Actualización exhaustiva de `web-dashboard/README.md` documentando la arquitectura SSR, Server Actions, variables de entorno y catálogo exacto de tarjetas deterministas.

## [0.9.0] - 2026-10-24

### Agregado
- **Panel Web Operativo & Simulador (Task 08):** Aplicación web interactiva en `web-dashboard/` construida con Next.js 15, React 19, TypeScript y Tailwind CSS para evaluación y monitoreo de pagos.
- **Dashboard de Transacciones (`/`):** Visualización de métricas en tiempo real (total procesado, volumen, aprobadas, rechazadas) y tabla interactiva de pagos con filtrado por estado y paginación.
- **Línea de Tiempo de Auditoría (`/payments/[id]`):** Componente visual `EventTimeline` vertical que detalla el ciclo de vida inmutable de cada transacción (`TRANSACTION_CREATED`, `STATUS_CHANGED`, `ACQUIRER_ATTEMPT`) con latencia bancaria y códigos de respuesta.
- **Simulador Interactivo de Checkout (`/sandbox`):** Formulario de pagos con precarga en un clic de las 7 tarjetas deterministas de prueba (Aprobada, Fondos insuficientes, 503, Flaky, Timeouts, Luhn), bloqueo síncrono anti-doble envío y resultado en tiempo real.
- **Arquitectura Unificada de API (Server Actions):** Integración de cobros interactivos mediante Server Actions de Next.js, unificando toda la configuración en una sola variable de entorno obligatoria (`PAYMENTS_API_URL`), eliminando variables públicas (`NEXT_PUBLIC_*`), problemas de CORS en navegador y exposición de topología de red interna.
- **Contenerización Standalone:** Configuración `output: 'standalone'` y `Dockerfile` multi-stage optimizado para despliegue en producción.

## [0.8.0] - 2026-10-24

### Agregado
- **Adaptador Primario HTTP (Task 07):** Capa HTTP plana en `internal/http/` sobre `net/http.ServeMux` de Go 1.22+ sin dependencias externas (`POST /v1/payments`, `GET /v1/payments/{id}`, `GET /v1/payments`).
- **Manejo Centralizado de Errores:** Serializador unificado `respondError` mapeando el catálogo de dominio a formato RFC 9457 (`application/problem+json`) con trazabilidad `X-Correlation-ID`.
- **Middlewares Globales:** Inyección de correlación (`Correlation`), logging estructurado PCI-DSS (`StructuredLogger` con `slog`) y captura de pánicos (`Recovery`).
- **Configuración y Composition Root:** Módulo `internal/config` con validación estricta fail-fast para todas las variables de entorno obligatorias (`PORT`, `DATABASE_URL`, `ACQUIRER_BASE_URL`, `PAYMENT_MAX_AMOUNT_CLP`, `LOG_LEVEL`), eliminando valores por defecto mágicos; punto de entrada `cmd/api/main.go` con apagado ordenado (10s) y `Dockerfile` multi-stage.

### Cambiado
- **Capa de Aplicación Unificada:** Consolidación de los casos de uso fragmentados (`processpayment`, `getpayment`, `listpayments`) en un único paquete limpio `internal/payment/` con `PaymentService` cohesivo, simplificando la inyección en el handler HTTP y el Composition Root.
- **Estandarización de Identificadores:** Comercios semilla actualizados a IDs numéricos (`1001`, `1002`) y `order_reference` a formato de 6 caracteres sin prefijos (`KD0877`).

## [0.7.0] - 2026-10-24

### Agregado
- **Casos de Uso de Aplicación & Orquestación (Task 06):** Implementación de la capa de orquestación de aplicación en `payments-api/internal/app/` con paquetes colocalizados (`processpayment/`, `getpayment/`, `listpayments/`), desacoplada de HTTP y SQL.
- **Flujo Síncrono de Procesamiento (`ProcessPaymentUseCase`):** Validación temprana de comercios activos, verificación de monto máximo configurable (`PAYMENT_MAX_AMOUNT_CLP`), hash SHA-256 de integridad de parámetros e inserción atómica ACID de intención, transacción inicial y evento de auditoría.
- **Manejo Estricto de Idempotencia y Replay:** Detección de colisiones de clave con payload alterado (`domain.ErrIdempotencyConflict`), reproducción inmediata de transacciones terminales previas (`IsReplayed = true`) sin cobro duplicado, y recuperación automática de transacciones huérfanas en `PROCESSING` con más de 5 minutos de antigüedad.
- **Protección Anti-Desconexión:** Invocación bancaria y persistencia desacopladas de cancelaciones de cliente mediante `context.WithoutCancel(ctx)` para prevenir transacciones en el limbo ante cierres de ventana del comprador.
- **Consulta con Aislamiento Multi-Tenant (`GetPaymentUseCase`):** Consulta protegida de transacciones y línea de tiempo completa de eventos de auditoría, previniendo fuga de información entre diferentes comercios.
- **Listado y Paginación Filtrada (`ListPaymentsUseCase`):** Soporte de filtrado por comercio y estado (`domain.TransactionStatus`) con paginación robusta acotada (`LIMIT` / `OFFSET`).

## [0.6.0] - 2026-10-23

### Agregado
- **Cliente HTTP del Adquirente Bancario & Resiliencia (Task 05):** Implementación del adaptador secundario en `payments-api/internal/acquirer/` para comunicación HTTP resiliente con `acquirer-mock`.
- **Política Estricta de Reintentos:** Reintentos automáticos (hasta 3 intentos totales) con backoff exponencial y jitter aleatorio exclusivamente ante caídas transitorias (`503 Service Unavailable` o caídas de red), garantizando cero reintentos ante decisiones deterministas de negocio (`APPROVED`, `DECLINED` ej. fondos insuficientes, tarjeta expirada, rechazo emisor) y errores de cliente (`400 Bad Request`, `422 Unprocessable`).
- **Reconciliación On-Demand ante Timeouts:** Mitigación de cobros huérfanos mediante consulta síncrona `GET /authorizations/{reference_id}` tras superar el tiempo límite de lectura/escritura, reconociendo cobros aprobados para evitar doble cobro o retornando falla segura (`FAILED`) ante `404 Not Found`.
- **Propagación de Trazabilidad (`X-Correlation-ID`):** Inyección obligatoria de cabeceras de correlación en todas las peticiones externas para unificar los logs entre la API y el simulador bancario.

## [0.5.0] - 2026-10-23

### Agregado
- **Adaptadores de Persistencia PostgreSQL (Task 04):** Implementación del adaptador secundario de base de datos en `payments-api/internal/postgres/` utilizando `jackc/pgx/v5` con pool de conexiones afinado (`pgxpool`) y consultas SQL explícitas.
- **Mapeo de Invariantes de Concurrencia (23505):** Detección y traducción transparente de violaciones de restricciones únicas de PostgreSQL a errores tipados de dominio (`ErrIdempotencyConflict`, `ErrIntentAlreadyApproved`, `ErrActiveTransactionExists`).
- **Transacciones Atómicas ACID:** Gestor transaccional `WithTx` para inserciones atómicas de intenciones, transacciones y eventos, manteniendo las consultas de lectura directamente en el pool.
- **Control de Concurrencia Optimista:** Actualización de estado condicional (`WHERE id = $1 AND status = $2`) para evitar sobrescritura de estados ante respuestas desfasadas.
- **Fórmula Determinista de Idempotencia:** Generador `ComputePayloadHash` (SHA-256) sobre parámetros canónicos de cobro cumpliendo con blindaje PCI-DSS.
- **Consultas y Paginación:** Métodos estandarizados `GetMerchantByID`, `GetPaymentIntentByOrderRef`, `GetTransactionByID`, `GetTransactionByIdempotencyKey`, `ListEventsByTransactionID` y `ListTransactions` con filtros y paginación.

## [0.4.0] - 2026-10-23

### Agregado
- **Núcleo de Dominio Puro (Task 03):** Implementación del hexágono interior en Go puro (`payments-api/internal/domain/`) consolidado en 5 archivos altamente cohesionados (`card.go`, `amount.go`, `merchant.go`, `payment.go`, `errors.go`) sin dependencias externas.
- **Value Objects y Blindaje PCI-DSS:** Validación del algoritmo de Luhn (Módulo 10) en `CardNumber`, expiración temporal en `CardExpiry`, `CVV` redactado y auto-enmascaramiento obligatorio mediante `slog.LogValuer` y `fmt.Stringer`.
- **Máquina de Estados Finita e Inmutable:** Definición de `TransactionStatus` como única fuente de verdad del sistema con control de transiciones legales (`PENDING` -> `PROCESSING` -> `APPROVED`/`DECLINED`/`FAILED`) e inmutabilidad estricta en estados terminales.
- **Catálogo de Errores Técnicos para Desarrolladores:** Estructura `DomainError` con códigos semánticos para frontend/control de flujo, descripciones técnicas concisas para desarrolladores/integradores y mapeo a códigos HTTP RFC 9457.
- **Entidades y Agregados:** Modelado de `Merchant`, `PaymentIntent`, `Transaction` (con métodos de mutación `MarkApproved`, `MarkDeclined`, `MarkFailed`) y `TransactionEvent` para auditoría append-only.

## [0.3.0] - 2026-10-23

### Agregado
- **Servicio Acquirer Mock Independiente (Task 02):** Implementación de simulador HTTP bancario autónomo en Go nativo (`net/http`) con su propio `go.mod`, reproduciendo 6 comportamientos deterministas de tarjeta (aprobaciones `00`, rechazos por fondos `51` / `insufficient_funds`, caídas 503 persistentes e intermitentes, y timeouts de red con y sin cobro previo).
- **Endpoint de Reconciliación On-Demand:** Soporte para consulta de transacciones vía `GET /authorizations/{reference_id}` para recuperar el estado ante pérdidas de respuesta por timeout de red y prevenir doble cobro.
- **Almacén en Memoria Thread-Safe:** Implementación de `MemoryStore` concurrente con `sync.RWMutex` para conteo de reintentos y consulta de autorizaciones.
- **Protección PCI-DSS:** Enmascaramiento automático de credenciales sensibles (PAN y CVV) mediante `slog.LogValuer` y `fmt.Stringer`.
- **Documentación y Guías Operativas:** Creación de `acquirer-mock/README.md`, `payments-api/README.md` y `AGENTS.md` con las directrices de arquitectura, estándares de ingeniería y flujo de desarrollo para agentes y colaboradores.

## [0.2.0] - 2026-10-23

### Agregado
- **Migraciones Iniciales y DDL PostgreSQL (Task 01):** Creación de scripts SQL versionados (`000001_init_schema` y `000002_seed_merchants`) gestionados con `golang-migrate`, integrando `payload_hash`, `updated_at`, `correlation_id`, `ON DELETE CASCADE` y suite de tests de integración para invariantes de concurrencia.
- **Refinamiento de Estructura de Paquetes:** Adopción de paquete plano `domain/` (entidades y value objects sin subcarpetas innecesarias) y colocalización de casos de uso en `internal/app/` (`processpayment/`, `getpayment/`, `listpayments/`).

## [0.1.0] - 2026-10-23

### Agregado
- **Documento de Requisitos de Producto (PRD):** Formalización de requerimientos funcionales (R1–R20), reglas de negocio (BR-01–BR-11), máquina de estados y criterios de aceptación en formato Gherkin en `docs/prd.md`.
- **Diseño Técnico:** Especificación de arquitectura hexagonal, esquema PostgreSQL de 4 tablas con restricciones físicas de concurrencia (`uq_tx_idempotency`, `uq_tx_one_approved`, `uq_tx_one_active`) y protocolo de trazabilidad de correlación de punto a punto en `docs/technical-design.md`.
- **Decisiones de Arquitectura:** Justificación técnica del procesamiento síncrono para checkout interactivo, recuperación on-demand sin procesos en segundo plano, hash de idempotencia SHA-256 y protección de datos PCI mediante tipos seguros en Go (`slog.LogValuer` / `fmt.Stringer`).
- **Framework de Ejecución de Tareas:** Especificación granular de tareas (Tareas 01 a 10) en `docs/tasks/` siguiendo las metodologías OpenSpec y Superpowers (TDD).
