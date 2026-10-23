# Task 02: Independent Acquirer Mock Service

- **Servicio:** `acquirer-mock`
- **Dependencias:** Ninguna
- **Estado:** PENDING
- **Trazabilidad:** PRD §R18, §R19 | Technical Design §13.1

---

## 1. Contexto y Objetivo
Implementar el simulador HTTP del adquirente bancario como un servicio independiente en Go con su propio `go.mod`. Este servicio reproduce con exactitud milimétrica el comportamiento determinista de un banco adquirente (aprobaciones, rechazos, errores 503 temporales, reintentos y timeouts) basándose en 6 tarjetas de prueba predefinidas.

---

## 2. Archivos Afectados
- `crear`: `acquirer-mock/go.mod`
- `crear`: `acquirer-mock/cmd/mock/main.go`
- `crear`: `acquirer-mock/internal/domain/card_behavior.go`
- `crear`: `acquirer-mock/internal/handler/authorization_handler.go`
- `crear`: `acquirer-mock/internal/storage/memory_store.go`
- `crear`: `acquirer-mock/internal/handler/authorization_handler_test.go`
- `crear`: `acquirer-mock/Dockerfile`

---

## 3. Especificación Técnica Detallada

### 3.1 Endpoints HTTP
1. **`POST /authorizations`**:
   - Headers: `Content-Type: application/json`, `X-Correlation-ID: <uuid>`
   - Body:
     ```json
     {
       "reference_id": "0192f3a4-...",
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
   - Respuestas:
     - `200 OK`: `{"status": "APPROVED", "code": "00", "message": "Approved", "authorization_code": "AUT-..."}`
     - `200 OK`: `{"status": "DECLINED", "code": "51", "message": "Insufficient funds"}`
     - `503 Service Unavailable`: `{"error": "service_unavailable", "message": "Acquirer system error"}`
     - Timeout simulado: duerme durante 3.5 segundos antes de responder o cortar la conexión.
2. **`GET /authorizations/{reference_id}`**:
   - Devuelve el estado de la autorización previamente registrada por la referencia:
     - `200 OK`: `{"reference_id": "...", "status": "APPROVED", "authorization_code": "..."}`
     - `404 Not Found`: si la referencia no existe (usado para comprobar que un timeout no generó cobro).

### 3.2 Tabla Determinista de Tarjetas
| PAN de Prueba | Comportamiento | Simulación |
| :--- | :--- | :--- |
| `4242424242424242` | Aprueba | `200 OK`, `APPROVED`, code `00` |
| `4000000000009995` | Rechaza por saldo | `200 OK`, `DECLINED`, code `51` |
| `4000000000000119` | Falla persistente | `503 Service Unavailable` |
| `4000000000000341` | Falla 2 veces con 503, aprueba al 3er intento | Cuenta llamadas por `reference_id` en memoria |
| `4000000000000101` | Timeout sin registro | Duerme 3.5s, no registra en memoria. `GET` devuelve `404` |
| `4000000000000259` | Timeout con autorización | Duerme 3.5s, pero registra `APPROVED` en memoria. `GET` devuelve `200` |
| Cualquier otro válido | Aprueba por defecto | `200 OK`, `APPROVED` |

---

## 4. Protocolo TDD (Superpowers)
1. **Fase Roja:** Escribir pruebas unitarias en `acquirer-mock/internal/handler/authorization_handler_test.go` para cada una de las 6 tarjetas de prueba y el endpoint `GET /authorizations/{ref}`.
2. **Fase Verde:** Implementar el store en memoria (`memory_store.go`) y el router `net/http` con los comportamientos exactos de sleep y status code.
3. **Fase Verificación:** Ejecutar `go test -v ./...` en `acquirer-mock/` verificando 100% de éxito en los 6 casos.

---

## 5. Criterios de Aceptación (Definition of Done)
- [ ] Módulo Go independiente configurado en `acquirer-mock/go.mod`.
- [ ] Las 6 tarjetas deterministas responden exactamente según la especificación.
- [ ] El endpoint `GET /authorizations/{ref}` permite consultar el estado posterior ante timeouts.
- [ ] `Dockerfile` compila y ejecuta el servicio en el puerto `8081` (o configurable vía `HTTP_PORT`).

---

## 6. Convención de Git Commit
`feat(acquirer-mock): implement deterministic standalone acquirer simulator and tests`
