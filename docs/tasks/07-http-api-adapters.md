# Task 07: HTTP API Presentation Layer & Server

- **Servicio:** `payments-api`
- **Dependencias:** `06-application-usecases`
- **Estado:** PENDING
- **Trazabilidad:** PRD §R1–R3, §R10–R12, §R16, §R17 | Technical Design §5.1, §9.1, §9.2, §9.3, §12.1

---

## 1. Contexto y Objetivo
Exponer la interfaz REST pública del procesador de pagos utilizando la biblioteca estándar `net/http` de Go (con enrutamiento nativo por método y patrón). Configurar middlewares globales de trazabilidad (`correlation_id`), logs estructurados JSON con `log/slog` que garanticen cero fugas de datos sensibles, manejo de pánicos con recuperación ordenada y formato estándar de errores RFC 9457 (`application/problem+json`).

---

## 2. Archivos Afectados
- `crear`: `payments-api/internal/adapters/httpapi/server.go`
- `crear`: `payments-api/internal/adapters/httpapi/router.go`
- `crear`: `payments-api/internal/adapters/httpapi/handler/payment_handler.go`
- `crear`: `payments-api/internal/adapters/httpapi/handler/health_handler.go`
- `crear`: `payments-api/internal/adapters/httpapi/middleware/correlation.go`
- `crear`: `payments-api/internal/adapters/httpapi/middleware/logger.go`
- `crear`: `payments-api/internal/adapters/httpapi/middleware/recovery.go`
- `crear`: `payments-api/internal/adapters/httpapi/dto/error_response.go` (RFC 9457)
- `crear`: `payments-api/internal/adapters/httpapi/dto/request.go`
- `crear`: `payments-api/internal/adapters/httpapi/dto/response.go`
- `crear`: `payments-api/internal/adapters/httpapi/handler/payment_handler_test.go`
- `crear`: `payments-api/cmd/api/main.go` (Composition Root)
- `crear`: `payments-api/Dockerfile`

---

## 3. Especificación Técnica Detallada

### 3.1 Endpoints y Códigos HTTP
- `POST /v1/payments`:
  - Encabezado obligatorio: `Idempotency-Key` (si falta $\rightarrow$ `400 Bad Request`).
  - Éxito normal: `201 Created` con el resultado del pago.
  - Reproducción idempotente: `200 OK` con encabezado `Idempotent-Replayed: true`.
  - Resultado desconocido (timeout): `202 Accepted` con `Retry-After: 5`.
  - Petición duplicada en curso: `409 Conflict` (`request_in_progress`).
  - Clave reutilizada con parámetros distintos: `422 Unprocessable Entity` / `409` (`idempotency_key_reused`).
  - Sobrecarga: `503 Service Unavailable` con `Retry-After: 2`.
- `GET /v1/payments/{id}`:
  - Devuelve la transacción y su arreglo completo de `events` ordenados cronológicamente (`200 OK`). Si no existe $\rightarrow$ `404 Not Found`.
- `GET /v1/payments?merchant_id=&status=&limit=&cursor=`:
  - Filtros opcionales con paginación por cursor. Sin resultados devuelve `200 OK` con `"data": []`.
- `GET /healthz` y `GET /readyz`.

### 3.2 Formato de Errores RFC 9457
Todas las respuestas de error retornan `Content-Type: application/problem+json`:
```json
{
  "type": "https://errors.haulmer.com/order-attempt-in-progress",
  "title": "Conflict",
  "status": 409,
  "code": "order_attempt_in_progress",
  "detail": "Another payment attempt for this order is currently in progress",
  "correlation_id": "c9d1e0f2-...",
  "instance": "/v1/payments"
}
```

### 3.3 Trazabilidad y Línea Conductora de Punto a Punto
Para garantizar la observabilidad total y debugging inmediato:
1. **Inyección y Propagación de `correlation_id`:**
   - El middleware `Correlation` extrae `X-Correlation-ID` del request o genera un UUID v7 si no viene.
   - Lo inyecta en el `context.Context` de Go y lo devuelve siempre en el encabezado de respuesta `X-Correlation-ID`.
   - Se incluye en el struct de error RFC 9457 y en el payload de respuesta de la transacción.
2. **Logs Estructurados en JSON (`log/slog`):**
   - Cada línea de log contiene obligatoriamente: `timestamp`, `level`, `correlation_id`, `merchant_id`, `duration_ms` y `step`.
   - Permite rastrear la historia completa en Datadog / CloudWatch / stdout filtrando únicamente por `correlation_id = "..."`.
   - **Regla estricta:** Cero fuga de PAN y CVV (los structs de tarjeta se redactan mediante `slog.LogValuer`).

---

## 4. Protocolo TDD (Superpowers)
1. **Fase Roja:** Escribir pruebas HTTP en `payment_handler_test.go` usando `httptest.NewRecorder()` probando validación de header `Idempotency-Key`, formato RFC 9457, inyección de `X-Correlation-ID` y serialización de respuestas.
2. **Fase Verde:** Implementar handlers, router y middlewares en `httpapi`.
3. **Fase Verificación:** Ejecutar `go test -v ./internal/adapters/httpapi/...`.

---

## 5. Criterios de Aceptación (Definition of Done)
- [ ] Endpoints implementados con `net/http` estándar de Go.
- [ ] Respuestas de error siguen estrictamente el estándar RFC 9457.
- [ ] Logs estructurados con `log/slog` en JSON: ninguna traza contiene PAN ni CVV.
- [ ] Apagado ordenado (*graceful shutdown*) en `main.go` esperando las peticiones activas.

---

## 6. Convención de Git Commit
`feat(http-api): implement REST handlers, correlation middleware, RFC 9457 errors, and main entrypoint`
