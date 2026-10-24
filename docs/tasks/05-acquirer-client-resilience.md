# Task 05: Acquirer Resilient HTTP Client & On-Demand Reconciliation

- **Servicio:** `payments-api`
- **Dependencias:** `02-acquirer-mock`, `03-domain-core`
- **Estado:** COMPLETED (v0.6.0)
- **Trazabilidad:** PRD §R7, §R8, §R9 | Technical Design §11.1, §11.2, §11.3

---

## 1. Contexto y Objetivo
Implementar el cliente HTTP hacia el adquirente bancario con tolerancia a fallos transitorios: hasta 3 reintentos con backoff exponencial y jitter ante respuestas `503`, política de no-reintento ante decisiones de negocio (`APPROVED`, `DECLINED`) y errores de cliente (`4xx`), y consulta de estado on-demand ante timeouts para evitar cobros huérfanos.

---

## 2. Archivos Implementados
- `payments-api/internal/acquirer/types.go` (DTOs y puerto `AcquirerPort`)
- `payments-api/internal/acquirer/client.go` (cliente HTTP resiliente, reintentos con backoff/jitter, reconciliación on-demand)
- `payments-api/internal/acquirer/client_test.go` (suite exhaustiva TDD con `httptest.Server` y `-race`)

---

## 3. Especificación Técnica Detallada

### 3.1 Puerto del Adquirente
```go
type AcquirerPort interface {
    Authorize(ctx context.Context, req AuthorizeRequest) (*AcquirerResponse, error)
    GetAuthorization(ctx context.Context, referenceID string, correlationID string) (*AcquirerResponse, error)
}
```

### 3.2 Política de Reintentos y Resiliencia
1. **Regla de Cero Reintentos (Deterministas):**
   - Si la respuesta es `APPROVED` (código `00`), `DECLINED` (ej. `51` saldo insuficiente, `05` rechazo emisor, `54` tarjeta expirada) o error de cliente (`400 Bad Request`, `422 Unprocessable`) $\rightarrow$ **NO se reintenta**, se retorna inmediatamente.
2. **Reintentos por Fallas Transitorias:**
   - Si la respuesta es `503 Service Unavailable` o fallo temporal de conexión $\rightarrow$ **Se reintenta hasta 3 veces** con backoff exponencial (base 200ms con jitter aleatorio).
3. **Resolución On-Demand de Timeouts:**
   - Si una llamada da timeout (> 2s), se invoca de forma síncrona `GetAuthorization(ctx, referenceID, correlationID)`.
   - Si `GetAuthorization` confirma que existe la autorización $\rightarrow$ se retorna `APPROVED` (evita doble cobro).
   - Si `GetAuthorization` devuelve 404 $\rightarrow$ se confirma que no hubo cobro, retornando `FAILED`.
   - Si `GetAuthorization` también falla o no responde $\rightarrow$ se retorna `ErrAcquirerUnknownResult` (la transacción quedará en `PROCESSING` con respuesta `202`).
4. **Propagación de Correlación hacia el Adquirente:**
   - El cliente HTTP inyecta obligatoriamente el encabezado `X-Correlation-ID` en cada llamada externa saliente hacia `acquirer-mock`.

---

## 4. Protocolo TDD (Superpowers)
1. **Fase Roja:**
   - Test con `httptest.Server` simulando dos respuestas `503` seguidas de un `200 OK APPROVED`, verificando que el cliente reintente y retorne éxito al tercer intento.
   - Tests verificando que respuestas `200 OK DECLINED` (51, 05, 54) y `400 Bad Request` jamás se reintenten.
   - Tests de timeout con reconciliación confirmada, 404 y fallida.
2. **Fase Verde:** Implementar la lógica del cliente HTTP con reintentos y reconciliación on-demand.
3. **Fase Verificación:** Ejecutar `go test -v -race ./internal/acquirer/...`.

---

## 5. Criterios de Aceptación (Definition of Done)
- [x] El cliente no reintenta errores lógicos de negocio (`DECLINED`) ni de cliente (`4xx`).
- [x] Los reintentos ante `503` respetan backoff exponencial con jitter.
- [x] Reconciliación on-demand ante timeouts evita doble cobro.
- [x] El cliente HTTP propaga el `X-Correlation-ID` en cada llamada externa.

---

## 6. Convención de Git Commit
`feat(acquirer): implement resilient HTTP client, retries, and on-demand reconciliation`
