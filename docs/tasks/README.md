# Task Execution Framework (OpenSpec + Superpowers)

Este directorio contiene las especificaciones granulares y atómicas para construir el sistema de procesamiento de pagos paso a paso, bajo los principios de **OpenSpec** (especificación rigurosa previa) y **Superpowers** (desarrollo guiado por pruebas TDD, verificación sistemática y commits atómicos).

---

## 1. Estándar de una Tarea (Task Template)

Cada archivo `docs/tasks/XX-<nombre>.md` sigue una estructura estandarizada para que cualquier agente o desarrollador pueda ejecutarla de manera autónoma y sin ambigüedad:

```markdown
# Task XX: [Nombre de la Tarea]

- **Servicio:** [payments-api | acquirer-mock | web-dashboard | infra]
- **Dependencias:** [Tareas previas requeridas]
- **Estado:** [PENDING | IN_PROGRESS | COMPLETED]
- **Trazabilidad:** [Referencias a secciones de prd.md y technical-design.md]

## 1. Contexto y Objetivo
Descripción concisa del problema que resuelve y el valor técnico que aporta.

## 2. Archivos Afectados
- `crear`: ruta/al/archivo.go
- `modificar`: ruta/al/archivo_existente.go

## 3. Especificación Técnica Detallada
Tipos de datos, interfaces, algoritmos y reglas de negocio aplicables.

## 4. Protocolo TDD (Superpowers)
1. **Fase Roja (Test First):** Archivos de prueba y aserciones esperadas.
2. **Fase Verde (Implementación Mínima):** Código mínimo para pasar la prueba.
3. **Fase Refactor / Verificación:** Comandos de validación (`go test`, `linter`, `build`).

## 5. Criterios de Aceptación (Definition of Done)
- [ ] Criterio 1
- [ ] Criterio 2

## 6. Convención de Git Commit
`tipo(scope): descripción concisa`
```

---

## 2. Definición de Hecho (Definition of Done - DoD)

Una tarea solo se considera **COMPLETED** si cumple con:
1. **Tests en Verde:** Todas las pruebas unitarias o de integración asociadas pasan limpiamente.
2. **Cero Violaciones PCI:** Ningún log ni archivo almacena PAN completo ni CVV.
3. **Tipado Estricto y Linter:** Sin advertencias de compilación ni errores de análisis estático (`go vet` / `golangci-lint` / `tsc`).
4. **Commit Atómico:** Los cambios se confirman en Git siguiendo Conventional Commits (`feat`, `test`, `refactor`, `fix`).

---

## 3. Mapa de Tareas (Roadmap Granular)

| # | Archivo | Servicio | Alcance Principal |
| :---: | :--- | :--- | :--- |
| **01** | [01-database-migrations.md](file:///home/samuel/Projects/haulmer/docs/tasks/01-database-migrations.md) | `payments-api` | Esquema DDL PostgreSQL (4 tablas, restricciones únicas parciales, seeds) |
| **02** | [02-acquirer-mock.md](file:///home/samuel/Projects/haulmer/docs/tasks/02-acquirer-mock.md) | `acquirer-mock` | Simulador HTTP independiente con 6 PANs deterministas y latencias |
| **03** | [03-domain-core.md](file:///home/samuel/Projects/haulmer/docs/tasks/03-domain-core.md) | `payments-api` | Entidades, Value Objects (Luhn, Mask, `slog.LogValuer`), máquina de estados |
| **04** | [04-persistence-adapters.md](file:///home/samuel/Projects/haulmer/docs/tasks/04-persistence-adapters.md) | `payments-api` | Repositorios SQL (`pgx/v5`), transacciones ACID, mapeo de carreras a errores |
| **05** | [05-acquirer-client-resilience.md](file:///home/samuel/Projects/haulmer/docs/tasks/05-acquirer-client-resilience.md) | `payments-api` | Cliente HTTP hacia el adquirente con reintentos 503, backoff y semáforo in-flight |
| **06** | [06-application-usecases.md](file:///home/samuel/Projects/haulmer/docs/tasks/06-application-usecases.md) | `payments-api` | Casos de uso (`ProcessPayment`, `GetPayment`, `ListPayments`), hash SHA-256, TTL 5m |
| **07** | [07-http-api-adapters.md](file:///home/samuel/Projects/haulmer/docs/tasks/07-http-api-adapters.md) | `payments-api` | Servidor HTTP `net/http`, middlewares (idempotencia, correlación), RFC 9457 |
| **08** | [08-web-dashboard.md](file:///home/samuel/Projects/haulmer/docs/tasks/08-web-dashboard.md) | `web-dashboard` | Panel Next.js 15: Server Components, tabla con filtros, timeline de eventos, sandbox |
| **09** | [09-docker-e2e-orchestration.md](file:///home/samuel/Projects/haulmer/docs/tasks/09-docker-e2e-orchestration.md) | `infra` | Dockerfiles, `docker-compose.yml`, CI con GitHub Actions, Makefile y suite E2E |
| **10** | [10-readme-documentation.md](file:///home/samuel/Projects/haulmer/docs/tasks/10-readme-documentation.md) | `docs` | README maestro para los evaluadores: pitch, diagramas, arquitectura y ejecución |
