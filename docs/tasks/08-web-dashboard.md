# Task 08: Web Dashboard (Next.js 15 & TypeScript)

- **Servicio:** `web-dashboard`
- **Dependencias:** `07-http-api-adapters`
- **Estado:** PENDING
- **Trazabilidad:** PRD §R16, §R17 | Technical Design §13.2

---

## 1. Contexto y Objetivo
Desarrollar la interfaz web moderna para operadores y evaluadores utilizando Next.js 15 (App Router) y TypeScript. La aplicación debe ofrecer una visualización clara del estado de las transacciones, filtros ágiles, una línea de tiempo (*timeline*) visual que detalle cada evento técnico de la transacción y un simulador interactivo de pagos (*Sandbox*) precargado con las tarjetas de prueba.

---

## 2. Archivos Afectados
- `crear`: `web-dashboard/package.json`
- `crear`: `web-dashboard/src/app/layout.tsx`
- `crear`: `web-dashboard/src/app/page.tsx` (Lista de pagos con filtros y resumen)
- `crear`: `web-dashboard/src/app/payments/[id]/page.tsx` (Detalle y timeline de eventos)
- `crear`: `web-dashboard/src/app/sandbox/page.tsx` (Simulador de checkout interactivo)
- `crear`: `web-dashboard/src/components/StatusBadge.tsx`
- `crear`: `web-dashboard/src/components/EventTimeline.tsx`
- `crear`: `web-dashboard/src/components/PaymentsTable.tsx`
- `crear`: `web-dashboard/src/lib/api-client.ts`
- `crear`: `web-dashboard/Dockerfile`

---

## 3. Especificación Técnica Detallada

### 3.1 Vistas Principales
1. **Listado de Pagos (`/`):**
   - Métricas clave en cabecera: Total de transacciones, % Aprobación, Tasa de fallos.
   - Tabla interactiva con badges de estado:
     - `APPROVED`: Verde esmeralda.
     - `DECLINED`: Ámbar/Naranja.
     - `FAILED`: Rojo/Rosa.
     - `PROCESSING`: Azul con micro-animación de carga.
   - Filtros por: `merchant_id`, `status` y paginación por cursor.
2. **Detalle de Transacción y Timeline (`/payments/[id]`):**
   - Tarjeta resumen: Monto formateado en CLP (`$25.990`), moneda, tarjeta enmascarada (`411111******1111`), código de autorización, `correlation_id`.
   - **Timeline Visual de Eventos (`EventTimeline`):**
     - Representación gráfica vertical de cada paso en `transaction_events`.
     - Despliega: `TRANSACTION_CREATED`, `STATUS_CHANGED`, `ACQUIRER_ATTEMPT` (con latencia en milisegundos y código de respuesta devuelto por el banco).
3. **Simulador de Pagos / Sandbox (`/sandbox`):**
   - Formulario de checkout rápido para que los evaluadores puedan probar la API en vivo sin usar cURL.
   - Selector rápido para rellenar con las 6 tarjetas de prueba (aprobación, fondos insuficientes, error 503, timeouts).
   - Botón *"Pagar"* con deshabilitación síncrona inmediata (`isSubmitting = true`) para evitar dobles clics.

### 3.2 Arquitectura Frontend
- Consumo mediante **Server Components** de Next.js conectándose a la variable de entorno `API_INTERNAL_URL` (la URL interna de Docker no se expone al navegador).
- Diseño sobrio, profesional y responsivo con soporte de modo oscuro por defecto.

---

## 4. Protocolo TDD y Verificación (Superpowers)
1. **Fase Roja:** Crear pruebas de componentes con React Testing Library / Jest verificando que `EventTimeline` renderice correctamente los pasos y que `StatusBadge` asigne los colores adecuados según el estado.
2. **Fase Verde:** Implementar los componentes y páginas en `src/`.
3. **Fase Verificación:** Ejecutar `npm run build` y `npm run lint` asegurando cero errores de compilación de TypeScript.

---

## 5. Criterios de Aceptación (Definition of Done)
- [ ] Aplicación Next.js 15 configurada con TypeScript estricto.
- [ ] Timeline visual renderiza el historial completo de eventos de una transacción.
- [ ] Sandbox interactivo permite disparar pagos de prueba con las tarjetas deterministas.
- [ ] `Dockerfile` multi-stage optimizado para producción.

---

## 6. Convención de Git Commit
`feat(web-dashboard): implement Next.js dashboard, event timeline, and payment sandbox`
