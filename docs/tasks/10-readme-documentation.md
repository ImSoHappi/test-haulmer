# Task 10: Pitch-Ready Master Documentation (README.md)

- **Servicio:** `docs`
- **Dependencias:** `01-database-migrations` a `09-docker-e2e-orchestration`
- **Estado:** PENDING
- **Trazabilidad:** PRD §Visión | Technical Design §1, §3, §16

---

## 1. Contexto y Objetivo
Redactar el `README.md` maestro en la raíz del repositorio. Este documento no es un simple manual de instalación: es la carta de presentación técnica y el guión de defensa ante el comité evaluador de Haulmer. Debe explicar con claridad meridiana el problema resuelto, los diagramas de arquitectura, las 5 decisiones técnicas diferenciadoras, cómo probar los 6 escenarios deterministas y el plan de escalabilidad futuro.

---

## 2. Archivos Afectados
- `modificar`: `README.md` (raíz del repositorio)

---

## 3. Especificación Técnica Detallada

### 3.1 Secciones Obligatorias del `README.md`
1. **Encabezado y Visión:**
   - Título: *Payment Processing Service — Haulmer Technical Challenge*.
   - Filosofía de diseño: Checkout web interactivo enfocado en inmediatez, seguridad PCI y consistencia matemática contra dobles cobros.
2. **Cómo Ejecutar con 1 Comando:**
   - `docker compose up --build`
   - Enlaces a los servicios en localhost:
     - Web Dashboard: `http://localhost:3000`
     - API REST de Pagos: `http://localhost:8080`
     - Acquirer Mock: `http://localhost:8081`
     - Métricas de salud: `http://localhost:8080/healthz`
3. **Diagramas de Arquitectura:**
   - Diagrama de componentes del Monorepo (`payments-api`, `acquirer-mock`, `web-dashboard`, `postgres`).
   - Diagrama de secuencia del flujo síncrono.
   - Modelo entidad-relación de 4 tablas.
4. **Las 5 Decisiones Técnicas Clave (El Pitch de los Jueces):**
   - *Por qué síncrono y sin colas/workers:* Inmediatez en pantalla y cumplimiento PCI (la tarjeta solo vive en memoria volátil de la petición).
   - *Recuperación On-Demand:* Sin demonios en segundo plano mutando transacciones a espaldas del usuario.
   - *Idempotencia y Concurrencia en PostgreSQL:* Restricciones únicas parciales (`uq_tx_idempotency`, `uq_tx_one_approved`, `uq_tx_one_active`).
   - *Modelo en 3 niveles:* Intención comercial (`payment_intents`), Intento técnico (`transactions`) y Auditoría inmutable (`transaction_events`).
   - *Protección PCI en Go:* Tipos de dominio implementando `slog.LogValuer` y `fmt.Stringer`.
5. **Tabla de Tarjetas de Prueba y Escenarios del Mock:**
   - Los 6 PANs de prueba, comportamiento esperado y comando `curl` listo para copiar y pegar.
6. **Ejecución de Pruebas:**
   - Pruebas unitarias: `make test`
   - Pruebas E2E automatizadas: `make test-e2e`
7. **Visión de Futuro y Escalabilidad:**
   - API 100% Stateless lista para réplicas horizontales.
   - Réplicas de lectura en PostgreSQL.
   - Multi-adquirente y Circuit Breaker.

---

## 4. Protocolo de Verificación (Superpowers)
1. **Verificación de Enlaces:** Comprobar que todos los enlaces relativos a `docs/prd.md` y `docs/technical-design.md` funcionen.
2. **Verificación de Comandos:** Probar en una terminal limpia que los comandos `docker compose up --build` y `curl` documentados funcionen sin requerir pasos previos.

---

## 5. Criterios de Aceptación (Definition of Done)
- [ ] `README.md` redactado en español con prosa técnica impecable, clara y convincente.
- [ ] Incluye instrucciones de inicio rápido (*Quickstart*) con un solo comando.
- [ ] Documenta las respuestas a las preguntas críticas de arquitectura evaluadas en el Grill-Me.

---

## 6. Convención de Git Commit
`docs: craft comprehensive pitch-ready README with architecture diagrams and quickstart`
