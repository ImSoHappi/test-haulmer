# Task 05: Acquirer HTTP Client & In-Flight Concurrency Limiter

- **Servicio:** `payments-api`
- **Dependencias:** `02-acquirer-mock`, `03-domain-core`
- **Estado:** PENDING
- **Trazabilidad:** PRD §R7, §R8, §R9 | Technical Design §11.1, §11.2, §11.3

---

## 1. Contexto y Objetivo
Implementar el cliente HTTP hacia el adquirente bancario con tolerancia a fallos transitorios: hasta 3 reintentos con backoff exponencial ante respuestas `503`, consulta de estado on-demand ante timeouts para evitar cobros huérfanos, y un semáforo de concurrencia en vuelo que proteja al adquirente rechazando peticiones excedentes con `503 Service Unavailable` y cabecera `Retry-After`.

---

## 2. Archivos Afectados
- `crear`: `payments-api/internal/app/acquirer_port.go`
- `crear`: `payments-api/internal/adapters/acquirer/http_client.go`
- `crear`: `payments-api/internal/adapters/acquirer/limiter.go` (semáforo de concurrencia)
- `crear`: `payments-api/internal/adapters/acquirer/http_client_test.go`

---

## 3. Especificación Técnica Detallada

### 3.1 Puerto del Adquirente
```go
type AcquirerPort interface {
    Authorize(ctx context.Context, req AuthorizeRequest) (*AcquirerResponse, error)
    GetAuthorization(ctx context.Context, referenceID string) (*AcquirerResponse, error)
}
```

### 3.2 Política de Reintentos y Resiliencia
1. **Regla de reintentos:**
   - Si la respuesta es `APPROVED` (código `00`) o `DECLINED` (ej. código `51` saldo insuficiente) $\rightarrow$ **NO se reintenta**, se retorna inmediatamente.
   - Si la respuesta es `503 Service Unavailable` o fallo temporal de conexión $\rightarrow$ **Se reintenta hasta 3 veces** con backoff exponencial (base 200ms con jitter aleatorio).
2. **Resolución On-Demand de Timeouts:**
   - Si una llamada da timeout (> 2s), se invoca `GetAuthorization(ctx, referenceID)`.
   - Si `GetAuthorization` confirma que existe la autorización $\rightarrow$ se retorna `APPROVED`.
   - Si `GetAuthorization` devuelve 404 $\rightarrow$ se confirma que no hubo cobro, retornando `FAILED`.
   - Si `GetAuthorization` también falla o no responde $\rightarrow$ se retorna `ErrAcquirerUnknownResult` (la transacción quedará en `PROCESSING` con respuesta `202`).
3. **Semáforo de Concurrencia en Vuelo (`ConcurrencyLimiter`):**
   - Canal con buffer de tamaño `ACQUIRER_MAX_INFLIGHT` (default 50).
   - `TryAcquire() bool`: si el canal está lleno, retorna `false` de inmediato $\rightarrow$ la API aborta con error de sobrecarga y encabezado `Retry-After: 2` antes de llamar al adquirente.
4. **Propagación de Correlación hacia el Adquirente:**
   - El cliente HTTP inyecta obligatoriamente el encabezado `X-Correlation-ID` en cada llamada externa saliente hacia `acquirer-mock`.
   - Permite que los logs del simulador del adquirente y los logs de la API compartan la misma traza de ejecución unificada.

---

## 4. Protocolo TDD (Superpowers)
1. **Fase Roja:**
   - Test con `httptest.Server` simulando dos respuestas `503` seguidas de un `200 OK APPROVED`, verificando que el cliente reintente y retorne éxito al tercer intento.
   - Test verificando que un `200 OK DECLINED` jamás se reintente.
   - Test saturando el semáforo concurrente para verificar que la petición 51 sea rechazada al instante.
2. **Fase Verde:** Implementar la lógica del cliente HTTP con reintentos y el semáforo.
3. **Fase Verificación:** Ejecutar `go test -v -race ./internal/adapters/acquirer/...`.

---

## 5. Criterios de Aceptación (Definition of Done)
- [ ] El cliente no reintenta errores lógicos de negocio (`DECLINED`).
- [ ] Los reintentos ante `503` respetan backoff exponencial con jitter.
- [ ] El semáforo de concurrencia protege la saturación de conexiones.
- [ ] El cliente HTTP propaga el `X-Correlation-ID` en cada llamada externa.

---

## 6. Convención de Git Commit
`feat(acquirer-client): implement resilient HTTP client, retries, and in-flight concurrency limiter`
