# Directrices para Agentes de IA (AGENTS.md)

Este documento define el contexto del proyecto, estándares de ingeniería, arquitectura y el protocolo estricto de desarrollo para cualquier agente de IA o desarrollador que colabore en este repositorio.

---

## 1. Contexto del Proyecto

Sistema de Procesamiento de Pagos con Tarjeta de Crédito/Débito para comercios chilenos en pesos (CLP).

### Componentes Principales
- **`payments-api/`**: Servicio backend central en Go (`1.23`), PostgreSQL (`16`) y Docker.
- **`acquirer-mock/`**: Simulador bancario independiente en Go (`1.23`) con comportamiento determinista.
- **`web-dashboard/`**: Panel de control interactivo para comercios en Next.js (`15`), React (`19`) y TypeScript.
- **`docker-compose.yml`**: Orquestación unificada para levantar el ecosistema completo con un comando.

---

## 2. Estándares de Arquitectura y Reglas de Negocio

### 2.1 Arquitectura Hexagonal
- `internal/domain/`: Núcleo puro sin dependencias externas ni SQL (entidades, value objects, máquina de estados).
- `internal/app/`: Casos de uso colocalizados:
  - `processpayment/` (usecase, dto, ports, tests)
  - `getpayment/` (usecase, dto, ports, tests)
- Adaptadores directos en `internal/`:
  - `internal/http/`: Adaptador primario (handlers HTTP, router, middleware).
  - `internal/postgres/`: Adaptador secundario (conexión, repositorio, migraciones).
  - `internal/acquirer/`: Adaptador secundario (cliente HTTP hacia el adquirente).

### 2.2 Reglas Críticas del Dominio Bancario
1. **Garantía Física de No Doble Cobro:**
   - La base de datos es la última línea de defensa mediante índices parciales únicos:
     - `uq_tx_idempotency`: `(merchant_id, idempotency_key)` evita cargos duplicados por clave.
     - `uq_tx_one_approved`: a lo sumo 1 transacción `APPROVED` por intención de pago.
     - `uq_tx_one_active`: a lo sumo 1 transacción en curso (`PENDING`, `PROCESSING`) simultánea.
2. **Regla de Oro de Persistencia (Anti-Carding / Anti-Spam):**
   - Rechazos tempranos de validación (Luhn fallido, tarjeta vencida, comercio inexistente/inactivo, montos fuera de rango) se manejan **en memoria en el umbral HTTP (400/422)** con logs estructurados. La base de datos almacena únicamente intentos legítimos enviados al adquirente.
3. **PCI-DSS Estricto (Cero Filtraciones):**
   - **NUNCA** registrar en logs ni almacenar en base de datos el PAN completo ni el CVV.
   - En BD solo se persisten `card_bin` (6 dígitos) y `card_last4` (4 dígitos).
   - Los tipos de tarjeta implementan `slog.LogValuer` y `fmt.Stringer` para auto-enmascararse (`424242******4242`).
4. **Recuperación On-Demand (Sin Cronjobs ni Daemons):**
   - En checkout interactivo, la API resuelve de forma síncrona. Si ocurre un timeout de red con el adquirente, la API reconcilia on-demand mediante `GET /authorizations/{reference_id}` antes de fallar.

---

## 3. Protocolo de Desarrollo y Flujo de Trabajo (MANDATORIO)

Cada tarea del proyecto sigue rigurosamente el siguiente flujo:

### Paso 1: Plan de Acción y Alineación
- Presentar o actualizar el artefacto `task_XX_action_plan.md` detallando arquitectura, contratos y pruebas.
- Recibir retroalimentación y aprobación del usuario antes de tocar código.

### Paso 2: Creación de Rama
- Toda tarea se trabaja en una rama dedicada:
  ```bash
  git checkout -b task/XX-<nombre-tarea>
  ```

### Paso 3: Protocolo TDD (Superpowers)
- **Fase Roja:** Escribir tests unitarios/integración que fallen primero.
- **Fase Verde:** Implementar el código mínimo necesario para pasar los tests.
- **Fase Refactor:** Optimizar código y verificar ejecución con `-race`:
  ```bash
  go test -v -race ./...
  ```

### Paso 4: CERO COMMITS DIRECTOS (Regla de Oro)
- **ESTÁ ESTRICTAMENTE PROHIBIDO** hacer `git commit` de forma automática durante la fase de desarrollo.
- Los cambios deben permanecer en el árbol de trabajo (*working tree*) para revisión del usuario.
- Presentar al usuario los resultados de los tests y el `git status`. Esperar su visto bueno explícito.

### Paso 5: Registro en Changelog
- Cada incremento funcional debe registrarse en `CHANGELOG.md` siguiendo [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/) y [Semantic Versioning](https://semver.org/lang/es/):
  - `[0.1.0]` Especificaciones y PRD.
  - `[0.2.0]` Tarea 01: Migraciones iniciales y layout.
  - `[0.3.0]` Tarea 02: Acquirer Mock independiente.
  - `[0.X.0]` Siguientes tareas.

### Paso 6: Commit, Pull Request y Merge
- Crear el commit con Conventional Commits en inglés:
  ```bash
  git add .
  git commit -m "feat(scope): descriptive message"
  git push -u origin task/XX-<nombre-tarea>
  ```
- Crear el Pull Request mediante la CLI de GitHub:
  ```bash
  gh pr create --title "feat(scope): ..." --body "..."
  ```
- Mergear a `master` y eliminar la rama remota y local:
  ```bash
  gh pr merge <PR-NUMBER> --merge --delete-branch
  git checkout master
  git pull origin master
  ```

---

## 4. Convenciones de Idioma

- **Código fuente, variables, nombres de funciones, comentarios de código, commits y descripciones de PR:** estrictamente en **Inglés**.
- **Documentación formal (`docs/`, `README.md`, `CHANGELOG.md`) y comunicación con el usuario:** en **Español**.
