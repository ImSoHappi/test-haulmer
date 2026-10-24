# Task 01: Database Migrations & Initial Schema

- **Servicio:** `payments-api`
- **Dependencias:** Ninguna
- **Estado:** COMPLETED (v0.2.0)
- **Trazabilidad:** PRD §BR-01, §BR-10, §BR-11 | Technical Design §8.1, §8.2, §8.3, §8.4

---

## 1. Contexto y Objetivo
Definir y versionar el esquema DDL completo de PostgreSQL que actúa como la fuente de verdad inmutable y garante de la consistencia transaccional del procesador de pagos. La base de datos debe impedir dobles cobros y violaciones de idempotencia a nivel de restricciones físicas (`UNIQUE INDEX`).

---

## 2. Archivos Afectados
- `creado`: `payments-api/migrations/000001_init_schema.up.sql`
- `creado`: `payments-api/migrations/000001_init_schema.down.sql`
- `creado`: `payments-api/migrations/000002_seed_merchants.up.sql`
- `creado`: `payments-api/migrations/000002_seed_merchants.down.sql`
- `creado`: `payments-api/migrations/migrations.go` (FS embebido con `embed.FS`)
- `creado`: `payments-api/internal/postgres/migrator.go` (Runner basado en `golang-migrate`)
- `creado`: `payments-api/internal/postgres/migrator_test.go` (Suite de tests de concurrencia e invariantes DDL)

---

## 3. Especificación Técnica y Decisiones de Arquitectura

### 3.1 Tablas y Restricciones
1. **`merchants`**:
   - `id TEXT PRIMARY KEY`
   - `name TEXT NOT NULL`
   - `status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'INACTIVE'))`
2. **`payment_intents`**:
   - `id UUID PRIMARY KEY`
   - `merchant_id TEXT NOT NULL REFERENCES merchants (id) ON DELETE CASCADE`
   - `order_reference TEXT NOT NULL`
   - `amount BIGINT NOT NULL CHECK (amount > 0)` (CLP enteros, sin decimales)
   - `currency TEXT NOT NULL CHECK (currency = 'CLP')`
   - `metadata JSONB NOT NULL DEFAULT '{}'::jsonb`
   - `correlation_id TEXT NOT NULL`
   - `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
   - `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`
   - `UNIQUE (merchant_id, order_reference)`
3. **`transactions`**:
   - `id UUID PRIMARY KEY`
   - `payment_intent_id UUID NOT NULL REFERENCES payment_intents (id) ON DELETE CASCADE`
   - `merchant_id TEXT NOT NULL REFERENCES merchants (id) ON DELETE CASCADE`
   - `idempotency_key TEXT NOT NULL`
   - `payload_hash TEXT NOT NULL` (SHA-256 de los parámetros canónicos de cobro)
   - `status TEXT NOT NULL CHECK (status IN ('PENDING','PROCESSING','APPROVED','DECLINED','FAILED'))`
   - `card_bin TEXT NOT NULL` (primeros 6 dígitos)
   - `card_last4 TEXT NOT NULL` (últimos 4 dígitos)
   - `decline_source TEXT CHECK (decline_source IN ('INTERNAL','ACQUIRER'))`
   - `result_code TEXT`
   - `result_message TEXT`
   - `authorization_code TEXT`
   - `correlation_id TEXT NOT NULL`
   - `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
   - `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`
   - `completed_at TIMESTAMPTZ`
4. **Índices Físicos de Concurrencia en `transactions`**:
   - `CREATE UNIQUE INDEX uq_tx_idempotency ON transactions (merchant_id, idempotency_key);` (evita doble cargo por clave)
   - `CREATE UNIQUE INDEX uq_tx_one_approved ON transactions (payment_intent_id) WHERE status = 'APPROVED';` (a lo sumo 1 cobro aprobado por intención)
   - `CREATE UNIQUE INDEX uq_tx_one_active ON transactions (payment_intent_id) WHERE status IN ('PENDING', 'PROCESSING');` (a lo sumo 1 intento en curso simultáneo, cerrando carreras paralelas)
   - `CREATE INDEX ix_tx_list ON transactions (merchant_id, status, created_at DESC, id DESC);`
5. **`transaction_events` (Append-Only / Solo Inserción)**:
   - `id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY`
   - `transaction_id UUID NOT NULL REFERENCES transactions (id) ON DELETE CASCADE`
   - `event_type TEXT NOT NULL CHECK (event_type IN ('TRANSACTION_CREATED','STATUS_CHANGED','ACQUIRER_ATTEMPT'))`
   - `from_status TEXT`
   - `to_status TEXT`
   - `acquirer_attempt INT`
   - `latency_ms INT`
   - `code TEXT`
   - `message TEXT`
   - `details JSONB`
   - `correlation_id TEXT NOT NULL`
   - `occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()`
   - `CREATE INDEX ix_events_tx ON transaction_events (transaction_id, id);`

### 3.2 Decisiones Clave Incorporadas
- **Regla de Oro de Persistencia (Anti-Spam / Anti-Carding):** Rechazos tempranos de validación (Luhn fallido, tarjeta vencida, comercio inexistente/inactivo, montos fuera de rango) se manejan en memoria en el umbral HTTP (400/422) con logs estructurados en JSON; la base de datos almacena exclusivamente intentos legítimos enviados al adquirente.
- **Refinamiento de Layout de Paquetes:** Adopción de paquete plano `domain/` y adaptadores directos en `internal/postgres/` (sin carpetas intermedias ni cajones de sastre como `platform/`).
- **Nombres y Trazabilidad:** `payload_hash TEXT NOT NULL` en `transactions` y campos `updated_at` y `correlation_id` en `payment_intents`.

### 3.3 Datos Semilla (`merchants`)
- `1001`: "Comercio Activo Principal" (`status = 'ACTIVE'`)
- `1002`: "Comercio Inactivo Test" (`status = 'INACTIVE'`)

---

## 4. Protocolo TDD (Superpowers)
1. **Fase Roja:** Prueba de integración en `internal/postgres/migrator_test.go` verificando conexión y migración limpia.
2. **Fase Verde:** Implementación de migrator sobre `golang-migrate` con `migrations.FS` embebido.
3. **Fase Verificación:** Inserción concurrente verificando rechazo físico con error de PostgreSQL `23505` ante colisiones en `uq_tx_idempotency`, `uq_tx_one_approved` y `uq_tx_one_active`.

---

## 5. Criterios de Aceptación (Definition of Done)
- [x] Los scripts SQL se ejecutan limpiamente en orden (`up.sql`).
- [x] La migración inversa (`down.sql`) elimina las tablas limpiamente sin errores de dependencias.
- [x] Los índices parciales `uq_tx_idempotency`, `uq_tx_one_approved` y `uq_tx_one_active` están creados.
- [x] Los comercios semilla `1001` y `1002` quedan insertados.
- [x] Layout limpio en `internal/postgres/` sin capas artificiales.

---

## 6. Convención de Git Commit
`feat(database): define initial PostgreSQL schema, concurrency indexes and merchant seeds`
