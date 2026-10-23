# Registro de Cambios (Changelog)

Todos los cambios notables en este proyecto se documentarán en este archivo.

El formato se basa en [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/)
y este proyecto se adhiere a [Semantic Versioning](https://semver.org/lang/es/).

## [0.1.0] - 2026-10-23

### Agregado
- **Documento de Requisitos de Producto (PRD):** Formalización de requerimientos funcionales (R1–R20), reglas de negocio (BR-01–BR-11), máquina de estados y criterios de aceptación en formato Gherkin en `docs/prd.md`.
- **Diseño Técnico:** Especificación de arquitectura hexagonal, esquema PostgreSQL de 4 tablas con restricciones físicas de concurrencia (`uq_tx_idempotency`, `uq_tx_one_approved`, `uq_tx_one_active`) y protocolo de trazabilidad de correlación de punto a punto en `docs/technical-design.md`.
- **Decisiones de Arquitectura:** Justificación técnica del procesamiento síncrono para checkout interactivo, recuperación on-demand sin procesos en segundo plano, hash de idempotencia SHA-256 y protección de datos PCI mediante tipos seguros en Go (`slog.LogValuer` / `fmt.Stringer`).
- **Framework de Ejecución de Tareas:** Especificación granular de tareas (Tareas 01 a 10) en `docs/tasks/` siguiendo las metodologías OpenSpec y Superpowers (TDD).
