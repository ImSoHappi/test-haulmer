# Task 03: Payments API Pure Domain Core

- **Servicio:** `payments-api`
- **Dependencias:** Ninguna (Dominio puro)
- **Estado:** PENDING
- **Trazabilidad:** PRD §BR-01, §BR-02, §BR-04, §BR-05, §BR-06, §BR-10 | Technical Design §5.1, §12.2

---

## 1. Contexto y Objetivo
Modelar el hexágono interior (núcleo del dominio) en Go puro, completamente desacoplado de HTTP, SQL o librerías de terceros. Contiene las entidades de negocio, value objects con validación de Luhn y vencimiento de tarjetas, blindaje PCI mediante `slog.LogValuer` y `fmt.Stringer`, y la máquina de estados de transacciones.

---

## 2. Archivos Afectados
- `crear`: `payments-api/go.mod`
- `crear`: `payments-api/internal/domain/entity/payment_intent.go`
- `crear`: `payments-api/internal/domain/entity/transaction.go`
- `crear`: `payments-api/internal/domain/entity/transaction_event.go`
- `crear`: `payments-api/internal/domain/entity/merchant.go`
- `crear`: `payments-api/internal/domain/valueobject/amount.go`
- `crear`: `payments-api/internal/domain/valueobject/card.go`
- `crear`: `payments-api/internal/domain/valueobject/status.go`
- `crear`: `payments-api/internal/domain/errors.go`
- `crear`: `payments-api/internal/domain/valueobject/card_test.go`
- `crear`: `payments-api/internal/domain/valueobject/amount_test.go`
- `crear`: `payments-api/internal/domain/valueobject/status_test.go`

---

## 3. Especificación Técnica Detallada

### 3.1 Value Objects
1. **`CardNumber`**:
   - `ValidateLuhn(number string) bool` (algoritmo estándar de Luhn).
   - `Mask() string` $\rightarrow$ formato `BIN + ****** + LAST4` (ej. `411111******1111`).
   - `BIN() string` (primeros 6 dígitos), `Last4() string` (últimos 4 dígitos).
   - Implementar `fmt.Stringer`: retorna siempre el string enmascarado.
   - Implementar `slog.LogValuer`: retorna `slog.StringValue(c.Mask())`.
2. **`CVV`**:
   - Validación: 3 o 4 dígitos numéricos.
   - Implementar `fmt.Stringer` y `slog.LogValuer`: retorna siempre `"[REDACTED]"`.
3. **`CardExpiry`**:
   - Validación de mes (1 a 12) y año ($\ge$ año actual).
   - `IsExpired(now time.Time) bool`.
4. **`Amount`**:
   - `int64` en pesos chilenos (CLP), estricto $> 0$, sin decimales.
   - `ValidateMax(maxLimit int64) error`.
5. **`TransactionStatus` & Máquina de Estados**:
   - Estados: `PENDING`, `PROCESSING`, `APPROVED`, `DECLINED`, `FAILED`.
   - Transiciones legales:
     - `PENDING` $\rightarrow$ `PROCESSING`, `DECLINED`, `FAILED`
     - `PROCESSING` $\rightarrow$ `APPROVED`, `DECLINED`, `FAILED`
     - `APPROVED`, `DECLINED`, `FAILED` son **estados terminales** (cualquier transición desde ellos retorna `ErrTerminalStateTransition`).

---

## 4. Protocolo TDD (Superpowers)
1. **Fase Roja:**
   - Escribir tests para Luhn con PANs válidos (Visa, Mastercard) e inválidos en `card_test.go`.
   - Escribir tests para `slog.LogValuer` y `fmt.Sprintf("%v", card)` asegurando que el PAN completo y el CVV jamás aparezcan en el output.
   - Escribir tests de transiciones de estado en `status_test.go` probando transiciones válidas y el rechazo de transiciones ilegales.
2. **Fase Verde:** Implementar la lógica mínima en los paquetes de dominio.
3. **Fase Verificación:** Ejecutar `go test -v -race ./internal/domain/...` con 100% de cobertura.

---

## 5. Criterios de Aceptación (Definition of Done)
- [ ] Cero dependencias externas en `internal/domain` (solo biblioteca estándar de Go).
- [ ] Algoritmo de Luhn y validación de expiración cubiertos por pruebas exhaustivas.
- [ ] Máquina de estados impide reabrir o mutar transacciones en estados terminales.
- [ ] Blindaje PCI validado: `CardNumber` y `CVV` jamás exponen datos sensibles al ser formateados o logueados.

---

## 6. Convención de Git Commit
`feat(domain): implement core entities, value objects, PCI redactors, and state machine`
