# Task 03: Payments API Pure Domain Core

- **Servicio:** `payments-api`
- **Dependencias:** Ninguna (Dominio puro)
- **Estado:** COMPLETED (v0.4.0)
- **Trazabilidad:** PRD §BR-01, §BR-02, §BR-04, §BR-05, §BR-06, §BR-10 | Technical Design §5.1, §12.2

---

## 1. Contexto y Objetivo
Modelar el hexágono interior (núcleo del dominio) en Go puro, completamente desacoplado de HTTP, SQL o librerías de infraestructura. Contiene las entidades de negocio, value objects con validación de Luhn y vencimiento de tarjetas, blindaje PCI mediante `slog.LogValuer` y `fmt.Stringer`, la máquina de estados de transacciones como única fuente de verdad, y el catálogo centralizado de errores técnicos de dominio.

---

## 2. Archivos Afectados (Consolidación Cohesiva)
- `creado`: `payments-api/internal/domain/card.go` (Value Objects: `CardNumber`, `CVV`, `CardExpiry`, `Card`)
- `creado`: `payments-api/internal/domain/card_test.go`
- `creado`: `payments-api/internal/domain/amount.go` (Value Object: `Amount` en CLP, `Currency`)
- `creado`: `payments-api/internal/domain/amount_test.go`
- `creado`: `payments-api/internal/domain/merchant.go` (Entidad `Merchant`, `MerchantStatus`)
- `creado`: `payments-api/internal/domain/payment.go` (`PaymentIntent`, `Transaction`, `TransactionEvent`, `TransactionStatus` y máquina de estados)
- `creado`: `payments-api/internal/domain/payment_test.go`
- `creado`: `payments-api/internal/domain/errors.go` (Catálogo estructurado de errores con `Code`, `Message`, `HTTPStatus`)
- `creado`: `payments-api/internal/domain/errors_test.go`

---

## 3. Especificación Técnica y Decisiones de Arquitectura

### 3.1 Value Objects de Tarjeta y Blindaje PCI (`card.go`)
1. **`CardNumber`**:
   - `NewCardNumber(raw string) (CardNumber, error)`: saneo de espacios/guiones, validación de 13 a 19 dígitos y algoritmo de Luhn (Módulo 10).
   - `BIN()` (primeros 6 dígitos), `Last4()` (últimos 4 dígitos).
   - `Mask()`: formato `BIN + ****** + LAST4` (ej. `411111******1111`).
   - `fmt.Stringer`: retorna exclusivamente el string enmascarado.
   - `slog.LogValuer`: retorna `slog.StringValue(c.Mask())`.
2. **`CVV`**:
   - Validación de 3 o 4 dígitos numéricos.
   - `fmt.Stringer` y `slog.LogValuer`: retorna siempre `"[REDACTED]"`.
   - `Raw()`: método seguro de acceso exclusivo para serialización hacia el cliente del adquirente.
3. **`CardExpiry`**:
   - Validación de mes (1 a 12) y año (soporte de 2 y 4 dígitos).
   - `IsExpired(now time.Time) bool`: válido hasta el último segundo del mes de expiración.
4. **`Card`**:
   - Agregado inmutable que agrupa número, fecha, CVV y titular, implementando enmascaramiento seguro en logs.

### 3.2 Value Object de Montos (`amount.go`)
- `Amount int64`: entero positivo estricto (`> 0`) en pesos chilenos (sin centavos ni decimales).
- `ValidateMax(limit int64) error`: control de tope máximo por transacción.
- `Currency`: constante canónica `"CLP"`.

### 3.3 Entidad de Comercios (`merchant.go`)
- `Merchant`: identificador, razón social y estado (`ACTIVE`, `INACTIVE`).
- `IsActive()`: determina si el comercio puede procesar cobros.

### 3.4 Agregado de Pagos y Máquina de Estados (`payment.go`)
- **Única fuente de verdad** del ciclo de vida de los pagos:
  - `PENDING` $\rightarrow$ `PROCESSING`, `DECLINED`, `FAILED`
  - `PROCESSING` $\rightarrow$ `APPROVED`, `DECLINED`, `FAILED`
  - `APPROVED`, `DECLINED`, `FAILED` $\rightarrow$ Estados terminales e inmutables (`ErrTerminalStateTransition`).
- Mutaciones seguras en `Transaction`: `MarkApproved`, `MarkDeclined`, `MarkFailed`.
- Registro inmutable de eventos `TransactionEvent` para auditoría y trazabilidad.

### 3.5 Catálogo Técnico de Errores (`errors.go`)
- Cada error expone:
  - `Code`: string semántico para consumo del frontend y control de flujo programático.
  - `Message`: descripción técnica concisa para desarrolladores e integradores.
  - `HTTPStatus`: código RFC 9457 correspondiente (400, 404, 409, 422, 502, 503, 504).

---

## 4. Protocolo TDD (Superpowers)
1. **Fase Roja:** Creación de suites de prueba unitarias exhaustivas en `card_test.go`, `amount_test.go`, `payment_test.go` y `errors_test.go`.
2. **Fase Verde:** Implementación del código de dominio puro.
3. **Fase Verificación:** Ejecución de `go test -v -race ./internal/domain/...` con 100% de éxito.

---

## 5. Criterios de Aceptación (Definition of Done)
- [x] Cero dependencias externas en `internal/domain` (solo biblioteca estándar de Go y `google/uuid`).
- [x] Consolidación temática en 5 archivos altamente cohesionados.
- [x] Algoritmo de Luhn y validación de expiración cubiertos por pruebas exhaustivas.
- [x] Máquina de estados impide reabrir o mutar transacciones en estados terminales.
- [x] Blindaje PCI validado: `CardNumber` y `CVV` jamás exponen datos sensibles al ser formateados o logueados.
- [x] Catálogo de errores técnicos estructurado para integradores con mapping RFC 9457.
- [x] Pruebas en verde con `-race` sin carreras de datos.

---

## 6. Convención de Git Commit
`feat(domain): implement core entities, value objects, PCI redactors, and state machine`
