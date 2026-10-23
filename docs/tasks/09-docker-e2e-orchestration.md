# Task 09: Docker Compose Orchestration, CI Pipeline & E2E Test Suite

- **Servicio:** `infra`
- **Dependencias:** `01-database-migrations` a `08-web-dashboard`
- **Estado:** PENDING
- **Trazabilidad:** PRD §R20, Métricas de Éxito | Technical Design §6, §13.3

---

## 1. Contexto y Objetivo
Orquestar todos los componentes del sistema (`postgres`, `acquirer-mock`, `payments-api`, `web-dashboard`) para que el stack completo se inicialice con **un solo comando** (`docker compose up --build`). Automatizar los flujos operativos mediante un `Makefile`, implementar un pipeline de Integración Continua (CI) en GitHub Actions (`.github/workflows/ci.yml`) que valide el código en cada commit, y construir una suite de pruebas End-to-End (E2E) que valide automáticamente los 6 escenarios de prueba del adquirente y las garantías de concurrencia.

---

## 2. Archivos Afectados
- `crear`: `docker-compose.yml`
- `crear`: `.env.example`
- `crear`: `Makefile`
- `crear`: `.github/workflows/ci.yml` (Pipeline de CI con GitHub Actions)
- `crear`: `scripts/test-e2e.sh`

---

## 3. Especificación Técnica Detallada

### 3.1 Servicios en `docker-compose.yml`
1. **`postgres`**:
   - Imagen: `postgres:16-alpine`.
   - Healthcheck: `pg_isready -U postgres -d payments_db`.
   - Montaje de migraciones automáticas en `/docker-entrypoint-initdb.d/`.
2. **`acquirer-mock`**:
   - Build: `./acquirer-mock`.
   - Puerto: `8081:8081`.
3. **`payments-api`**:
   - Build: `./payments-api`.
   - Puerto: `8080:8080`.
   - Dependencias: `postgres` (`condition: service_healthy`) y `acquirer-mock` (`condition: service_started`).
   - Variables: `DATABASE_URL`, `ACQUIRER_BASE_URL=http://acquirer-mock:8081`, `HTTP_ADDR=:8080`.
4. **`web-dashboard`**:
   - Build: `./web-dashboard`.
   - Puerto: `3000:3000`.
   - Variables: `API_INTERNAL_URL=http://payments-api:8080`.

### 3.2 Comandos del `Makefile`
- `make up`: `docker compose up --build -d`
- `make down`: `docker compose down -v`
- `make test`: Ejecuta pruebas unitarias de `payments-api` y `acquirer-mock`.
- `make test-e2e`: Ejecuta `scripts/test-e2e.sh` contra el stack en ejecución.
- `make lint`: Ejecuta análisis estático (`golangci-lint` y `npm run lint`).

### 3.3 Pipeline de Integración Continua (CI en `.github/workflows/ci.yml`)
- **Triggers:** `push` y `pull_request` a ramas principales (`main`, `master`).
- **Job 1: `backend-ci` (Go):**
  - Setup de Go (1.22+).
  - Ejecución de `golangci-lint` para `payments-api` y `acquirer-mock`.
  - Ejecución de pruebas unitarias y de concurrencia: `go test -v -race ./...`.
- **Job 2: `frontend-ci` (Node/Next.js):**
  - Setup de Node.js (20.x).
  - Instalación limpia: `npm ci`.
  - Typecheck y Linter: `npm run lint`.
  - Verificación de Build: `npm run build`.
- **Job 3: `e2e-smoke`:**
  - Inicialización con `docker compose up -d`.
  - Espera a healthchecks y ejecución de `scripts/test-e2e.sh`.

### 3.4 Suite de Pruebas E2E (`scripts/test-e2e.sh`)
Verifica automáticamente:
1. **Caso 1 (Aprobación):** PAN `4242424242424242` $\rightarrow$ `201 Created`, `status = APPROVED`.
2. **Caso 2 (Saldo Insuficiente):** PAN `4000000000009995` $\rightarrow$ `201 Created`, `status = DECLINED`, código `51`.
3. **Caso 3 (Reintentos 503):** PAN `4000000000000341` $\rightarrow$ `201 Created`, `status = APPROVED` (tras 2 reintentos).
4. **Caso 4 (Idempotencia Exacta):** Reenvío con la misma clave $\rightarrow$ `200 OK`, `Idempotent-Replayed: true`.
5. **Caso 5 (Payload Alterado):** Misma clave con monto diferente $\rightarrow$ `409 Conflict` / `422`.
6. **Caso 6 (Integridad de la Orden):** Segundo intento aprobado sobre orden ya pagada $\rightarrow$ `409 Conflict` (`order_already_paid`).

---

## 4. Protocolo TDD (Superpowers)
1. **Fase Roja:** Crear `scripts/test-e2e.sh` con aserciones `curl` y validaciones JSON que fallarán antes de levantar el stack.
2. **Fase Verde:** Configurar `docker-compose.yml`, Dockerfiles multi-stage, `.github/workflows/ci.yml` y resolver dependencias de red.
3. **Fase Verificación:** Ejecutar `make up && make test-e2e` verificando salida en verde para todos los casos.

---

## 5. Criterios de Aceptación (Definition of Done)
- [ ] Todo el ecosistema levanta con `docker compose up --build` sin intervención manual.
- [ ] El script `test-e2e.sh` valida los 6 casos con códigos de salida `0`.
- [ ] Los healthchecks garantizan que la API no arranque antes de que la BD esté lista.
- [ ] El workflow de CI `.github/workflows/ci.yml` está configurado para validar backend, frontend y tests automáticamente.

---

## 6. Convención de Git Commit
`feat(infra): configure Docker Compose, CI pipeline, Makefile, and automated E2E test suite`
