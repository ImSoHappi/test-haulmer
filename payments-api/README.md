# Payments API

Servicio central de procesamiento de pagos con tarjeta de crédito/débito para comercios chilenos en CLP, construido en Go bajo Arquitectura Hexagonal.

---

## 1. Arquitectura y Estructura del Código

El proyecto separa estrictamente la lógica pura de negocio de la infraestructura y entrega casos de uso colocalizados:

```
payments-api/
├── cmd/
│   └── api/
│       └── main.go                         # Punto de entrada, inyección de dependencias y graceful shutdown
├── internal/
│   ├── domain/                             # Núcleo de dominio (sin dependencias externas)
│   │   ├── amount.go                       # Value Object: montos en CLP (enteros BIGINT)
│   │   ├── card.go                         # Value Object: PAN, BIN, Last4, enmascaramiento PCI
│   │   ├── merchant.go                     # Entidad de comercio y validaciones
│   │   ├── payment_intent.go               # Agregado de intención de pago
│   │   ├── transaction.go                  # Entidad de intento de cobro con adquirente
│   │   ├── transaction_event.go            # Registro inmutable de eventos de auditoría
│   │   ├── status.go                       # Máquina de estados (PENDING, PROCESSING, APPROVED, etc.)
│   │   └── errors.go                       # Errores de dominio tipados (RFC 9457)
│   ├── app/                                # Casos de uso (Vertical Slices)
│   │   ├── processpayment/                 # Caso de uso: Procesar Pago
│   │   │   ├── usecase.go                  # Orquestación: idempotencia, bloqueo y llamada a adquirente
│   │   │   ├── dto.go                      # DTOs de entrada y salida
│   │   │   ├── ports.go                    # Interfaces requeridas (Repository, AcquirerClient)
│   │   │   └── usecase_test.go             # Pruebas unitarias de flujo
│   │   ├── getpayment/                     # Caso de uso: Consultar Pago por ID
│   │   └── listpayments/                   # Caso de uso: Listar Pagos con filtros y paginación
│   ├── http/                               # Adaptador de entrada: router, handlers HTTP, middleware
│   ├── postgres/                           # Adaptador de salida: BD (pgx/v5, sqlc, migraciones)
│   │   ├── migrator.go                     # Runner de migraciones con golang-migrate
│   │   └── migrator_test.go                # Tests de DDL y restricciones de concurrencia
│   └── acquirer/                           # Adaptador de salida: cliente HTTP hacia acquirer-mock
├── migrations/                             # Scripts SQL DDL versionados (Up y Down)
│   ├── 000001_init_schema.up.sql           # Tablas e índices físicos de concurrencia
│   ├── 000002_seed_merchants.up.sql        # Comercios de prueba (m_1001, m_1002)
│   └── migrations.go                       # Sistema de archivos embebido (embed.FS)
├── Dockerfile                              # Multi-stage build estático
├── README.md                               # Documentación del servicio
└── go.mod / go.sum                         # Módulo Go 1.23
```

---

## 2. Decisiones de Arquitectura Clave

1. **Garantía Física de No Doble Cobro:**
   - La base de datos impone índices únicos parciales:
     - `uq_tx_idempotency (merchant_id, idempotency_key)`: Idempotencia a nivel de transacción.
     - `uq_tx_one_approved`: A lo sumo un cobro aprobado por cada intención de pago.
     - `uq_tx_one_active`: A lo sumo un cobro activo (`PENDING`, `PROCESSING`) en curso simultáneamente.
2. **Regla de Oro de Persistencia (Anti-Spam / Anti-Carding):**
   - Las peticiones con errores tempranos (formato JSON inválido, fallo Luhn, tarjeta vencida, comercio inexistente o inactivo) se rechazan **en memoria en el umbral HTTP (400/422)** con logs estructurados. La base de datos almacena únicamente intentos legítimos enviados al adquirente.
3. **Protección de Datos PCI-DSS:**
   - Nunca se almacena ni se registra en logs el PAN completo ni el CVV.
   - En base de datos solo se guardan `card_bin` (primeros 6 dígitos) y `card_last4` (últimos 4 dígitos).
   - Los tipos de tarjeta implementan `slog.LogValuer` y `fmt.Stringer` para auto-enmascararse automáticamente.
4. **Recuperación On-Demand (Sin Workers Asíncronos):**
   - En un checkout interactivo, la API resuelve de forma síncrona. Si ocurre un timeout con el adquirente, la API reconcilia en la misma petición o cuando el comercio reintenta con la misma `Idempotency-Key` vía `GET /authorizations/{ref}`.

---

## 3. Variables de Entorno

| Variable | Descripción | Valor por Defecto |
| :--- | :--- | :--- |
| `HTTP_PORT` | Puerto HTTP de la API | `8080` |
| `DATABASE_URL` | String de conexión PostgreSQL | `postgres://postgres:postgres@localhost:5432/payments?sslmode=disable` |
| `ACQUIRER_URL` | URL base del Acquirer Mock | `http://localhost:8081` |
| `ACQUIRER_TIMEOUT_SEC` | Timeout por intento HTTP con adquirente | `2` |
| `ACQUIRER_MAX_RETRIES` | Número máximo de reintentos con backoff | `3` |

---

## 4. Ejecución y Pruebas

### Pruebas de Integración y Migraciones
Requiere instancia local de PostgreSQL en el puerto `5432`:
```bash
cd payments-api
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/payments_test?sslmode=disable" go test -v ./internal/postgres/...
```

### Ejecutar la API Localmente
```bash
cd payments-api
go run ./cmd/api
```
