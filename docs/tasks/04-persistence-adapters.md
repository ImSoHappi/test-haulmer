# Task 04: PostgreSQL Persistence Adapters

- **Servicio:** `payments-api`
- **Dependencias:** `01-database-migrations`, `03-domain-core`
- **Estado:** COMPLETED (v0.5.0)
- **Trazabilidad:** PRD §BR-07, §BR-11 | Technical Design §5.1, §8.3, §10.2

---

## 1. Contexto y Objetivo
Implementar los adaptadores de salida para PostgreSQL utilizando `pgx/v5` con SQL explícito. Los repositorios respetan transacciones ACID para operaciones compuestas (crear intento + transacción + evento) y traducen limpiamente los errores de restricción única de PostgreSQL (`23505`) a errores semánticos de dominio.

---

## 2. Archivos Implementados
- `payments-api/internal/postgres/pool.go` (conexión pool con `pgxpool`)
- `payments-api/internal/postgres/repository.go` (repositorio consolidado, QueryRunner, transacciones atómicas `WithTx`, mapeo 23505 y bloqueo optimista)
- `payments-api/internal/postgres/repository_test.go` (suite de pruebas de integración con `-race` contra PostgreSQL real)

---

## 3. Especificación Técnica Detallada

### 3.1 Puertos (Interfaces en `internal/app/ports.go`)
```go
type MerchantRepository interface {
    FindByID(ctx context.Context, id string) (*entity.Merchant, error)
}

type PaymentIntentRepository interface {
    CreateOrGet(ctx context.Context, tx pgx.Tx, pi *entity.PaymentIntent) (*entity.PaymentIntent, error)
    FindByOrderReference(ctx context.Context, merchantID, orderRef string) (*entity.PaymentIntent, error)
}

type TransactionRepository interface {
    Create(ctx context.Context, tx pgx.Tx, t *entity.Transaction) error
    UpdateStatus(ctx context.Context, tx pgx.Tx, t *entity.Transaction, fromStatus valueobject.TransactionStatus) (bool, error)
    FindByIdempotencyKey(ctx context.Context, merchantID, key string) (*entity.Transaction, error)
    FindByID(ctx context.Context, id uuid.UUID) (*entity.Transaction, []*entity.TransactionEvent, error)
    List(ctx context.Context, filter ListFilter) ([]*entity.Transaction, *Cursor, error)
}

type EventRepository interface {
    Append(ctx context.Context, tx pgx.Tx, event *entity.TransactionEvent) error
}
```

### 3.2 Mapeo de Errores de Concurrencia PostgreSQL
Al ejecutar `INSERT INTO transactions ...`:
Si el error retornado por Postgres tiene código `23505` (Unique Violation):
- Si el constraint es `uq_tx_idempotency` $\rightarrow$ Retornar `domain.ErrDuplicateIdempotencyKey`.
- Si el constraint es `uq_tx_one_approved` $\rightarrow$ Retornar `domain.ErrOrderAlreadyApproved`.
- Si el constraint es `uq_tx_one_active` $\rightarrow$ Retornar `domain.ErrOrderAttemptInProgress`.

---

## 4. Protocolo TDD (Superpowers)
1. **Fase Roja:**
   - Escribir tests de integración en `transaction_repo_test.go` que ejecuten dos inserts concurrentes con la misma `idempotency_key` y verifiquen que el segundo falle con `ErrDuplicateIdempotencyKey`.
   - Escribir test que verifique que intentar insertar dos transacciones con status `APPROVED` para la misma `payment_intent_id` falle con `ErrOrderAlreadyApproved`.
2. **Fase Verde:** Implementar los queries SQL y el extractor de errores `pgconn.PgError`.
3. **Fase Verificación:** Ejecutar la suite contra la base de datos de test (`go test -v -tags=integration ./internal/adapters/postgres/...`).

---

## 5. Criterios de Aceptación (Definition of Done)
- [x] Operaciones de base de datos utilizan `pgx/v5` con sentencias parametrizadas (cero riesgo de SQL Injection).
- [x] Transacciones atómicas: `payment_intent` + `transaction` + `event` se crean en una sola transacción SQL.
- [x] Mapeo exacto de los índices `uq_tx_idempotency`, `uq_tx_one_approved` y `uq_tx_one_active` a errores de dominio.
- [x] Paginación estable basada en `LIMIT` y `OFFSET`.

---

## 6. Convención de Git Commit
`feat(postgres): implement repositories, transaction manager, and concurrency error mapping`
