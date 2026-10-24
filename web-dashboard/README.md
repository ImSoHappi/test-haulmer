# Haulmer Payment - Panel Web Operativo & Sandbox

Panel de control interactivo para evaluación, monitoreo y simulación de transacciones de la pasarela de pagos Haulmer. Construido con **Next.js 15 (App Router)**, **React 19**, **TypeScript** y **Tailwind CSS**.

---

## 1. Características Principales

1. **Dashboard en Tiempo Real (`/`):**
   - Resumen de métricas operativas (Total procesado, transacciones Aprobadas, Rechazadas y Fallidas).
   - Tabla paginada de transacciones con filtrado por estado (`APPROVED`, `DECLINED`, `FAILED`, `PROCESSING`).
   - Soporte multi-tenant con cambio de comercio en la cabecera (`1001` vs `1002`).
   - Renderizado en el servidor (**Server-Side Rendering - SSR**) con manejo resiliente ante fallas de red.

2. **Línea de Tiempo de Auditoría (`/payments/[id]`):**
   - Inspección forense del ciclo de vida inmutable de cada transacción (`TRANSACTION_CREATED`, `STATUS_CHANGED`, `ACQUIRER_ATTEMPT`).
   - Desglose de latencia bancaria en milisegundos y códigos de retorno del adquirente.

3. **Simulador Interactivo de Checkout (`/sandbox`):**
   - Checkout con precarga en un clic de las 7 tarjetas de prueba deterministas.
   - Generación automática de orden y clave de idempotencia con opción de repetición de clave para validar la protección anti-doble cobro.
   - Arquitectura **Backend-for-Frontend (BFF)** mediante Next.js **Server Actions** (`src/app/sandbox/actions.ts`), eliminando problemas de CORS y evitando la exposición de la red interna al navegador.

---

## 2. Variables de Entorno

El servicio utiliza una única variable obligatoria para conectarse a la API central:

| Variable | Descripción | Valor por Defecto Local |
| :--- | :--- | :--- |
| `PAYMENTS_API_URL` | URL base del servicio `payments-api` | `http://localhost:8080` (Docker: `http://payments-api:8080`) |

Para desarrollo local, copia el archivo de ejemplo:
```bash
cp .env.example .env.local
```

---

## 3. Tarjetas de Prueba Deterministas

El simulador (`/sandbox`) cuenta con 7 escenarios bancarios predefinidos:

| Escenario | PAN | HTTP / Estado | Comportamiento Bancario Esperado |
| :--- | :--- | :--- | :--- |
| **Aprobada** | `4242 4242 4242 4242` | `201` / `APPROVED` | Flujo exitoso estándar. El banco autoriza inmediatamente. |
| **Fondos Insuficientes** | `4000 0000 0000 9995` | `402` / `DECLINED` | Rechazada deterministamente por el emisor con código `51` (Fondos insuficientes). |
| **Falla Intermitente (503)** | `4000 0000 0000 0341` | `201` / `APPROVED` | Simula falla 503 en intentos 1 y 2, se recupera con éxito en el 3er intento. |
| **Caída Bancaria Sostenida** | `4000 0000 0000 0119` | `503` / `FAILED` | El adquirente simula caída persistente tras agotar los 3 reintentos con backoff. |
| **Timeout sin Autorizar** | `4000 0000 0000 0101` | `504` / `FAILED` | Timeout de lectura; reconciliación on-demand consulta al adquirente y recibe 404. |
| **Timeout con Autorización**| `4000 0000 0000 0259` | `201` / `APPROVED` | Timeout de lectura; reconciliación on-demand detecta cargo previo y aprueba. |
| **Anti-Carding (Luhn Inválido)** | `4242 4242 4242 4241` | `422 Unprocessable` | Dígito verificador Luhn incorrecto; rechazado en memoria sin tocar base de datos. |

---

## 4. Ejecución en Desarrollo

```bash
# Instalar dependencias
npm ci

# Ejecutar servidor de desarrollo con Turbopack
npm run dev

# Verificar linter de código
npm run lint

# Construir bundle de producción (Standalone)
npm run build
```

---

## 5. Contenerización en Producción

El proyecto incluye un `Dockerfile` multi-stage optimizado que utiliza `output: 'standalone'` en Next.js, reduciendo el tamaño de la imagen final y corriendo sin requerir la suite completa de Node modules de desarrollo.
