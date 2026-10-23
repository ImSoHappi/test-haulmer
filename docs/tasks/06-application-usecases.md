# Task 06: Application Use Cases & Orchestration

- **Servicio:** `payments-api`
- **Dependencias:** `03-domain-core`, `04-persistence-adapters`, `05-acquirer-client-resilience`
- **Estado:** PENDING
- **Trazabilidad:** PRD §R1–R6, §R10–R15 | Technical Design §5.2, §5.3, §7.1, §7.2, §10.1

---

## 1. Contexto y Objetivo
Implementar la capa de aplicación que orquesta el flujo completo de cobro síncrono (`ProcessPaymentUseCase`), la consulta detallada con historial de eventos (`GetPaymentUseCase`) y el listado filtrado con paginación (`ListPaymentsUseCase`). Garantiza el cálculo del hash SHA-256 de idempotencia, el TTL de 5 minutos para transacciones huérfanas y el desacople de cancelación (`context.WithoutCancel`).

---

## 2. Archivos Afectados
- `crear`: `payments-api/internal/app/dto/payment.go`
- `crear`: `payments-api/internal/app/usecase/process_payment.go`
- `crear`: `payments-api/internal/app/usecase/get_payment.go`
- `crear`: `payments-api/internal/app/usecase/list_payments.go`
- `crear`: `payments-api/internal/app/usecase/process_payment_test.go`
- `crear`: `payments-api/internal/app/usecase/get_payment_test.go`

---

## 3. Especificación Técnica Detallada

### 3.1 Flujo de `ProcessPaymentUseCase`
1. **Validación del Comercio:** Consultar `MerchantRepository`. Si el comercio no existe o está inactivo (`status != 'ACTIVE'`), persistir intento como `DECLINED` con `decline_source = 'INTERNAL'` y motivo `merchant_not_allowed`, retornando error de negocio.
2. **Protección de Sobrecarga:** Verificar `ConcurrencyLimiter.TryAcquire()`. Si está lleno, retornar `ErrServiceOverloaded` con sugerencia `Retry-After: 2`.
3. **Cálculo de Integridad:** Calcular `request_hash = SHA256(merchant_id + order_reference + amount + currency + card_bin + card_last4)`.
4. **Verificación de Idempotencia:**
   - Si la clave ya existe para el comercio:
     - Si `stored_hash != incoming_hash` $\rightarrow$ Retornar `ErrIdempotencyKeyReused` (código HTTP 422/409).
     - Si la transacción está en estado terminal (`APPROVED`, `DECLINED`, `FAILED`) $\rightarrow$ Retornar la transacción existente con flag `IsReplayed = true`.
     - Si está en `PROCESSING` y han pasado $< 5$ minutos $\rightarrow$ Retornar `ErrRequestInProgress` (HTTP 409).
     - Si está en `PROCESSING` y han pasado $\ge 5$ minutos $\rightarrow$ Transicionar a `FAILED` (`PROCESSING_TIMEOUT`) y permitir nuevo intento.
5. **Creación Atómica en BD:**
   - En una transacción SQL: crear/recuperar `PaymentIntent`, insertar `Transaction` en `PENDING` y registrar evento `TRANSACTION_CREATED`.
6. **Reglas de Negocio Internas:** Validar monto máximo (`PAYMENT_MAX_AMOUNT_CLP`). Si excede $\rightarrow$ transicionar a `DECLINED` (`amount_limit_exceeded`), registrar evento y retornar sin llamar al adquirente.
7. **Transición a `PROCESSING`:** Transicionar estado en BD y registrar evento `STATUS_CHANGED`.
8. **Llamada Externa al Adquirente:**
   - Ejecutar con `context.WithoutCancel(ctx)` para evitar estados inconsistentes si el cliente HTTP desconecta.
   - Invocar `AcquirerPort.Authorize(...)`.
9. **Cierre Atómico y Eventos:**
   - Registrar evento `ACQUIRER_ATTEMPT` con latencia y respuesta.
   - Transicionar la transacción a su estado final (`APPROVED`, `DECLINED` o `FAILED`). Si el resultado quedó desconocido tras reintentos y consulta final, mantener en `PROCESSING` con respuesta `202`.

---

## 4. Protocolo TDD (Superpowers)
1. **Fase Roja:**
   - Crear mocks para `MerchantRepository`, `PaymentIntentRepository`, `TransactionRepository`, `EventRepository` y `AcquirerPort`.
   - Escribir tests para:
     - Flujo exitoso de aprobación (`APPROVED`).
     - Rechazo por fondos insuficientes (`DECLINED`).
     - Rechazo interno por monto máximo superado (`DECLINED` sin llamar al adquirente).
     - Reintento idempotente que devuelva el resultado previo sin volver a llamar al adquirente.
     - Detección de colisión de clave con payload alterado (`ErrIdempotencyKeyReused`).
2. **Fase Verde:** Implementar `ProcessPaymentUseCase` conectando todos los componentes.
3. **Fase Verificación:** Ejecutar `go test -v -race ./internal/app/usecase/...`.

---

## 5. Criterios de Aceptación (Definition of Done)
- [ ] Orquestación desacoplada: el caso de uso no conoce SQL ni HTTP.
- [ ] La clave de idempotencia con diferente payload es detectada y rechazada.
- [ ] Si el cliente aborta la conexión, la transacción concluye su ciclo sin quedar en el limbo.
- [ ] Cobertura de pruebas unitarias sobre casos de uso $\ge 90\%$.

---

## 6. Convención de Git Commit
`feat(application): implement payment processing, idempotency hash checks, and query use cases`
