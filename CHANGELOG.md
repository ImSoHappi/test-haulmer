# Registro de Cambios (Changelog)

Todos los cambios notables en este proyecto se documentarán en este archivo.

El formato se basa en [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/)
y este proyecto se adhiere a [Semantic Versioning](https://semver.org/lang/es/).
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
