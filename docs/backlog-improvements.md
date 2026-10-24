# Backlog de Mejoras Técnicas y Refactorización

Este documento registra oportunidades de mejora identificadas durante el desarrollo para ser abordadas en una fase posterior de refactorización y pulido.

---

## 1. Refactorización de `processpayment.Service.ProcessPayment`

- **Contexto:** Actualmente `ProcessPayment` orquesta en un solo método lineal la validación temprana, cálculo de hash SHA-256, verificación de idempotencia y expiración de 5 min, persistencia atómica con `pgx.Tx`, reglas de límite interno, invocación con contexto desacoplado al adquirente y registro de múltiples eventos de auditoría.
- **Mejora:**
  - Extraer sub-funciones privadas con nombres claros que cuenten la historia del caso de uso en 30–40 líneas de alto nivel:
    - `validateRequest(req Request) (*merchant, *amount, *currency, *card, error)`
    - `resolveIdempotency(ctx, req, hash) (*Response, bool, error)`
    - `createPendingTransaction(ctx, req, hash, card) (*Transaction, error)`
    - `executeAcquirerCall(ctx, req, tx) (*acquirer.AuthorizeResponse, error)`
    - `finalizeTransaction(ctx, tx, resp, err)`
  - Mantener los contratos y tests unitarios intactos (los tests de `service_test.go` servirán como arnés de regresión).

---

## 2. Fail-Fast Estricto en Variables de Entorno (Sin Valores Mágicos Ocultos)

- **Contexto:** Los servicios adoptan el enfoque minimalista idiomático de Go en sus constructores (`NewService`), limitándose a la inyección y asignación directa de dependencias y parámetros sin lógica ni fallbacks mágicos ocultos.
- **Responsabilidad de Configuración (Tarea 07):**
  - La validación del entorno corresponde 100% a `internal/config/config.go`.
  - En la Tarea 07, asegurar que `PAYMENT_MAX_AMOUNT_CLP` y demás variables mandatorias se carguen con validación estricta al iniciar el servidor en `cmd/api/main.go`. Si falta una variable crítica o tiene un formato inválido (`<= 0`), el proceso debe abortar de inmediato (`log.Fatal`) indicando con precisión qué variable falta, garantizando que el servicio nunca reciba valores espurios.
