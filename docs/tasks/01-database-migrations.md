# Task 01: Database Migrations & Initial Schema

- **Servicio:** `payments-api`
- **Dependencias:** Ninguna
- **Estado:** PENDING
- **Trazabilidad:** PRD §BR-01, §BR-10, §BR-11 | Technical Design §8.1, §8.2, §8.3, §8.4

---

## 1. Contexto y Objetivo
Definir y versionar el esquema DDL completo de PostgreSQL que actúa como la fuente de verdad inmutable y garante de la consistencia transaccional del procesador de pagos. La base de datos debe impedir dobles cobros y violaciones de idempotencia a nivel de restricciones físicas (`UNIQUE INDEX`).

---

## 2. Archivos Afectados
- `crear`: `payments-api/migrations/000001_init_schema.up.sql`
- `crear`: `payments-api/migrations/000001_init_schema.down.sql`
- `crear`: `payments-api/migrations/000002_seed_merchants.up.sql`
- `crear`: `payments-api/migrations/000002_seed_merchants.down.sql`
- `crear`: `payments-api/internal/platform/postgres/migrator.go` (o runner de migraciones)

---

## 3. Especificación Técnica Detallada

### 3.1 Tablas y Restricciones
1. **`merchants`**:
   - `id TEXT PRIMARY KEY`
   - `name TEXT NOT NULL`
   - `status TEXT NOT NULL CHECK (status IN ('ACTIVE', 'INACTIVE'))`
2. **`payment_intents`**:
   - `id UUID PRIMARY KEY`
   - `merchant_id TEXT NOT NULL` (sin FK para permitir auditoría de comercios no registrados)
   - `order_reference TEXT NOT NULL`
   - `amount BIGINT NOT NULL CHECK (amount > 0)` (CLP enteros, sin decimales)
   - `currency TEXT NOT NULL CHECK (currency = 'CLP')`
   - `metadata JSONB NOT NULL DEFAULT '{}'::jsonb`
   - `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
   - `UNIQUE (merchant_id, order_reference)`
3. **`transactions`**:
   - `id UUID PRIMARY KEY`
   - `payment_intent_id UUID NOT NULL REFERENCES payment_intents (id)`
   - `merchant_id TEXT NOT NULL`
   - `idempotency_key TEXT NOT NULL`
   - `request_hash TEXT NOT NULL` (SHA-256)
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
4. **Índices Críticos de Concurrencia en `transactions`**:
   - `CREATE UNIQUE INDEX uq_tx_idempotency ON transactions (merchant_id, idempotency_key);`
   - `CREATE UNIQUE INDEX uq_tx_one_approved ON transactions (payment_intent_id) WHERE status = 'APPROVED';`
   - `CREATE UNIQUE INDEX uq_tx_one_active ON transactions (payment_intent_id) WHERE status IN ('PENDING', 'PROCESSING');`
   - `CREATE INDEX ix_tx_list ON transactions (merchant_id, status, created_at DESC, id DESC);`
5. **`transaction_events` (Append-Only)**:
   - `id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY`
   - `transaction_id UUID NOT NULL REFERENCES transactions (id)`
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

### 3.2 Datos Semilla (`merchants`)
- `m_1001`: "Comercio Activo Principal" (`status = 'ACTIVE'`)
- `m_1002`: "Comercio Inactivo Test" (`status = 'INACTIVE'`)

---

## 4. Protocolo TDD (Superpowers)
1. **Fase Roja:** Escribir prueba de integración en Go (`payments-api/internal/platform/postgres/migrator_test.go`) que intente conectar a una base de datos PostgreSQL de prueba y aplicar las migraciones.
2. **Fase Verde:** Crear los archivos `.sql` y ejecutar el runner de migración hasta que todas las tablas, índices y seeds existan.
3. **Fase Verificación:** Comprobar en PostgreSQL que intentar insertar dos registros duplicados en `uq_tx_idempotency` o dos registros `APPROVED` en `uq_tx_one_approved` retorne violación de clave única (`23505`).

---

## 5. Criterios de Aceptación (Definition of Done)
- [ ] Los scripts SQL se ejecutan limpiamente en orden (`up.sql`).
- [ ] La migración inversa (`down.sql`) elimina las tablas limpiamente sin errores de dependencias.
- [ ] Los índices parciales `uq_tx_idempotency`, `uq_tx_one_approved` y `uq_tx_one_active` están creados.
- [ ] Los comercios semilla `m_1001` y `m_1002` quedan insertados.

---

## 6. Convención de Git Commit
`feat(database): define initial PostgreSQL schema, concurrency indexes and merchant seeds`
