# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-10-23

### Added
- **Product Requirements Document (PRD):** Formalization of functional requirements (R1–R20), business rules (BR-01–BR-11), state machine, and Gherkin acceptance criteria in `docs/prd.md`.
- **Technical Design:** Hexagonal architecture specification, 4-table PostgreSQL schema with physical concurrency constraints (`uq_tx_idempotency`, `uq_tx_one_approved`, `uq_tx_one_active`), and end-to-end correlation tracing in `docs/technical-design.md`.
- **Architecture Decisions:** Formal justification of synchronous interactive checkout, on-demand recovery without background daemons, SHA-256 idempotency hash, and PCI masking via Go `slog.LogValuer` / `fmt.Stringer`.
- **Task Execution Framework:** Granular task specifications (Tasks 01–10) in `docs/tasks/` following OpenSpec and Superpowers TDD methodology.
