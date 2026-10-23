# Acquirer Mock Service

Simulador bancario independiente en Go (`net/http` nativo) que reproduce con fidelidad los comportamientos deterministas de un adquirente financiero (aprobaciones, rechazos por fondos, sobrecargas 503, reintentos y timeouts de red con o sin autorización previa).

---

## 1. Arquitectura y Estructura

El servicio opera con un módulo Go independiente (`go 1.23`) desacoplado del API principal:

```
acquirer-mock/
├── cmd/
│   └── mock/
│       └── main.go                         # Punto de entrada HTTP y graceful shutdown
├── internal/
│   ├── domain/
│   │   ├── card_behavior.go                # Catálogo determinista de PANs y clasificación
│   │   └── authorization.go                # Modelos de autorización y enmascaramiento PCI
│   ├── handler/
│   │   ├── authorization_handler.go        # Rutas POST /authorizations y GET /authorizations/{id}
│   │   └── authorization_handler_test.go   # Suite de pruebas TDD para todos los escenarios
│   └── storage/
│       ├── memory_store.go                 # Almacén en memoria thread-safe (sync.RWMutex)
│       └── memory_store_test.go            # Pruebas de concurrencia y -race
├── Dockerfile                              # Multi-stage build estático (Alpine 3.20)
├── README.md                               # Documentación del servicio
└── go.mod / go.sum                         # Dependencias mínimas (uuid, testify)
```

---

## 2. Catálogo Determinista de Tarjetas

El comportamiento ante una autorización se determina según el número de tarjeta (PAN) recibido:

| PAN de Prueba | Escenario | Código HTTP | Código ISO / Decline Code | Memoria Interna |
| :--- | :--- | :---: | :--- | :--- |
| `4242424242424242` | **Aprobado** | `200 OK` | `00` / `APPROVED` | Registra autorización |
| `4000000000009995` | **Saldo insuficiente** | `200 OK` | `51` / `insufficient_funds` | Registra rechazo |
| `4000000000000119` | **Caída persistente** | `503 Service Unavailable` | `acquirer_down` | No registra |
| `4000000000000341` | **Falla temporal / Flaky** | `503` (intentos 1 y 2)<br>`200` (intento 3+) | `temporary_system_overload`<br>`00` al 3er intento | Cuenta llamadas por `reference_id` |
| `4000000000000101` | **Timeout sin cobro** | `504` tras delay (3.5s) | `gateway_timeout` | **No registra** (`GET` da `404`) |
| `4000000000000259` | **Timeout con cobro previo** | `504` tras delay (3.5s) | `gateway_timeout` | **Sí registra** (`GET` da `200`) |
| Cualquier otro válido | **Aprobado por defecto** | `200 OK` | `00` / `APPROVED` | Registra autorización |

---

## 3. Endpoints HTTP

### `POST /authorizations`
Procesa una solicitud de autorización de pago.

- **Headers:** `Content-Type: application/json`, `X-Correlation-ID: <uuid>` (opcional).
- **Body:**
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
- **Respuesta Exitosa (`200 OK`):**
  ```json
  {
    "reference_id": "0192f3a4-8b65-7a91-95a2-c12e8f1b9a21",
    "status": "APPROVED",
    "code": "00",
    "authorization_code": "AUT-7F3B1A8C",
    "amount": 25990,
    "currency": "CLP",
    "created_at": "2026-10-23T18:57:17Z"
  }
  ```

### `GET /authorizations/{reference_id}`
Permite a la pasarela reconciliar el estado real de un cobro tras un timeout de red:
- **`200 OK`**: Transacción encontrada y autorizada (evita doble cobro al comprador).
- **`404 Not Found`**: Ninguna autorización fue procesada para esa referencia (seguro marcar `FAILED`).

---

## 4. Variables de Entorno

| Variable | Descripción | Valor por Defecto |
| :--- | :--- | :--- |
| `HTTP_PORT` | Puerto de escucha HTTP | `8081` |
| `TIMEOUT_DELAY_MS` | Duración del sleep para simular timeouts | `3500` (milisegundos) |

---

## 5. Ejecución y Pruebas

### Ejecutar Localmente
```bash
cd acquirer-mock
go run ./cmd/mock
```

### Ejecutar Pruebas Unitarias y de Concurrencia (-race)
```bash
cd acquirer-mock
go test -v -race ./...
```

### Construir Imagen Docker
```bash
docker build -t acquirer-mock:latest ./acquirer-mock
```
