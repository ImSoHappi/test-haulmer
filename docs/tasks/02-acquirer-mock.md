# Task 02: Independent Acquirer Mock Service

- **Servicio:** `acquirer-mock`
- **Dependencias:** Ninguna
- **Estado:** COMPLETED (v0.3.0)
- **Trazabilidad:** PRD §R18, §R19 | Technical Design §11.2, §11.3, §13.1

---

## 1. Contexto y Objetivo
Implementar el simulador HTTP del adquirente bancario como un servicio independiente en Go con su propio `go.mod`. Este servicio reproduce con exactitud milimétrica el comportamiento determinista de un banco adquirente (aprobaciones, rechazos, errores 503 temporales, reintentos y timeouts) basándose en 6 tarjetas de prueba predefinidas y un endpoint de consulta para reconciliación post-timeout.

---

## 2. Archivos Afectados
- `creado`: `acquirer-mock/go.mod`
- `creado`: `acquirer-mock/cmd/mock/main.go`
- `creado`: `acquirer-mock/internal/domain/card_behavior.go`
- `creado`: `acquirer-mock/internal/domain/authorization.go`
- `creado`: `acquirer-mock/internal/handler/authorization_handler.go`
- `creado`: `acquirer-mock/internal/handler/authorization_handler_test.go`
- `creado`: `acquirer-mock/internal/storage/memory_store.go`
- `creado`: `acquirer-mock/internal/storage/memory_store_test.go`
- `creado`: `acquirer-mock/Dockerfile`
- `creado`: `acquirer-mock/README.md`

*(Nota: Se descartó `health_handler.go` durante el diseño para mantener el componente enfocado y minimalista).*

---

## 3. Especificación Técnica y Decisiones Incorporadas

### 3.1 Endpoints HTTP
1. **`POST /authorizations`**:
   - Headers: `Content-Type: application/json`, `X-Correlation-ID: <uuid>`
   - Body:
     ```json
     {
       "reference_id": "0192f3a4-8b65-7a91-95a2-c12e8f1b9a21",
       "amount": 25990,
       "currency": "CLP",
       "card": {
         "number": "4242424242424242",
         "expiry_month": 12,
         "expiry_year": 2028,
         "cvv": "123",
         "holder_name": "Ana Pérez"
       }
     }
     ```
   - Respuestas alineadas con Stripe y red bancaria ISO 8583:
     - `200 OK`: `{"status": "APPROVED", "code": "00", "authorization_code": "AUT-..."}`
     - `200 OK`: `{"status": "DECLINED", "code": "51", "decline_code": "insufficient_funds", "message": "Your card has insufficient funds."}`
     - `503 Service Unavailable`: `{"error": "service_unavailable", "code": "acquirer_down", "message": "..."}`
     - Timeout simulado: duerme durante `TIMEOUT_DELAY_MS` (3.5s en producción, 10ms en tests) antes de responder `504 Gateway Timeout`.
2. **`GET /authorizations/{reference_id}`**:
   - Permite a la API principal reconciliar on-demand el estado de un cobro tras sufrir un timeout de red con el adquirente:
     - `200 OK`: Autorización registrada previamente (`{"reference_id": "...", "status": "APPROVED", ...}`). Evita doble cobro al comprador.
     - `404 Not Found`: Si no existe autorización para la referencia. Garantiza a la API que el adquirente jamás cobró y es 100% seguro marcar `FAILED`.

### 3.2 Tabla Determinista de Tarjetas
| PAN de Prueba | Comportamiento | Simulación HTTP | Almacén en Memoria |
| :--- | :--- | :--- | :--- |
| `4242424242424242` | **Aprobación directa** | `200 OK`, `APPROVED`, code `00` | Registra autorización |
| `4000000000009995` | **Rechazo por saldo** | `200 OK`, `DECLINED`, code `51`, `insufficient_funds` | Registra rechazo |
| `4000000000000119` | **Falla persistente** | `503 Service Unavailable` (`acquirer_down`) | No registra |
| `4000000000000341` | **Falla temporal / Flaky** | Intentos 1 y 2: `503 Service Unavailable`<br>Intento 3+: `200 OK` (`APPROVED`, code `00`) | Cuenta llamadas por `reference_id` |
| `4000000000000101` | **Timeout sin cobro** | Duerme 3.5s, no registra en memoria. `GET` devuelve `404` | **NO registra** (seguro marcar `FAILED`) |
| `4000000000000259` | **Timeout con cobro previo** | Registra `APPROVED` en memoria y luego duerme 3.5s. `GET` devuelve `200` | **SÍ registra** (salva de doble cobro) |
| Cualquier otro válido | **Aprobación por defecto** | `200 OK`, `APPROVED`, code `00` | Registra autorización |

### 3.3 Almacenamiento en Memoria Minimalista (`MemoryStore`)
- Implementación thread-safe de escasas 45 líneas con `sync.RWMutex` exclusivamente para:
  1. Contar los reintentos de la tarjeta intermitente `...0341`.
  2. Guardar autorizaciones para resolver el `GET /authorizations/{reference_id}` tras timeouts.

### 3.4 Protección de Datos PCI-DSS
- Enmascaramiento automático de credenciales sensibles mediante `slog.LogValuer` y `fmt.Stringer`: el PAN completo y el CVV nunca aparecen en stdout/stderr.

---

## 4. Protocolo TDD (Superpowers)
1. **Fase Roja:** Pruebas unitarias en `acquirer-mock/internal/handler/authorization_handler_test.go` y de concurrencia en `acquirer-mock/internal/storage/memory_store_test.go`.
2. **Fase Verde:** Implementación del router `net/http` nativo, catálogo de comportamiento y almacén en memoria.
3. **Fase Verificación:** Ejecución de `go test -v -race ./...` asegurando 100% de éxito con cero carreras de datos.

---

## 5. Criterios de Aceptación (Definition of Done)
- [x] Módulo Go independiente configurado en `acquirer-mock/go.mod`.
- [x] Las 6 tarjetas deterministas responden exactamente según la especificación (con códigos ISO/Stripe).
- [x] El endpoint `GET /authorizations/{ref}` permite consultar el estado posterior ante timeouts.
- [x] MemoryStore thread-safe con pruebas de concurrencia bajo `-race`.
- [x] `Dockerfile` compila y ejecuta el servicio en el puerto `8081` (configurable vía `HTTP_PORT`).
- [x] `README.md` documenta arquitectura, variables y tabla de tarjetas.

---

## 6. Convención de Git Commit
`feat(acquirer-mock): implement deterministic standalone acquirer simulator and tests`
