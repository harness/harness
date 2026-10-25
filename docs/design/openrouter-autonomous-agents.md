# Plan: errores y vulnerabilidades con agentes autónomos vía OpenRouter

Estado: propuesta · Autor: Claude Code a pedido de @juhasztobias · Fecha: 2026-10-25

## 1. Objetivo

Agregar a Harness dos productos que comparten una misma base de IA:

- **Insights**: error tracking estilo Sentry, integrado con el código, los pipelines y los releases de Harness.
- **Shield**: análisis de vulnerabilidades estilo Snyk (dependencias, código, secretos, imágenes del registry).

En los dos casos, cuando aparece un problema que supera un umbral configurable, un **agente** lo valida, reproduce y corrige, y abre un PR. Los modelos se consumen vía **OpenRouter**, con routing por tarea, fallbacks, límites de gasto y políticas de datos.

## 2. Lo que ya existe y se reutiliza

| Pieza existente | Dónde | Uso en este plan |
|---|---|---|
| `AITask`: tareas de IA que corren Claude Code dentro de un Gitspace | `types/ai_task.go`, `app/services/aitaskevent`, `app/gitspace/orchestrator` | Runtime de los agentes que arreglan código. Ya registra costo, tokens y modelos (`AIUsageMetric`). |
| Auth de agentes (`AIAgentAuth`, hoy solo `anthropic-api-key`) | `types/enum/ai_agent_auth.go`, `orchestrator/utils/ai_agent.go` | Se agrega `openrouter-api-key`. |
| Script de configuración de Claude Code | `orchestrator/utils/script_templates/configure_claude_code.sh` | Se apunta Claude Code a OpenRouter (`ANTHROPIC_BASE_URL`). |
| Sistema de eventos (streams, readers, reporters) | `events/`, `app/events/*` | Nuevas categorías `insights` y `shield`, y consumo de `pipeline.executed` y eventos de git. |
| Scheduler de jobs (recurrentes y one-shot) | `job/` | Scans periódicos, sync de bases de vulnerabilidades, agrupación de errores. |
| Principals de servicio | `app/bootstrap` (`systemServicePrincipal`, etc.) | Nuevo principal `harness-agent` que firma commits y PRs de los agentes. |
| Protection rules con status checks requeridos | `app/services/protection` | Gate de merge: el check `shield/security` bloquea PRs que superan el umbral. |
| Automerge | `app/services/automerge` | Nivel máximo de autonomía (solo si se habilita explícitamente). |
| Secret scanning (`SecretScanningEnabled`) | `app/services/settings/security.go` | Shield lo absorbe como un scanner más. |
| Settings de repo y space | `app/services/settings` | Umbrales, nivel de autonomía, modelos y presupuesto por scope. |
| Gestor de secretos | `app/services/secret`, `platformSecret` | Guarda las API keys de OpenRouter (BYOK por space). |
| Artifact registry | `registry/` | Guarda source maps y símbolos (Insights) y escanea imágenes y paquetes (Shield). |
| Language analyzer | `app/services/languageanalyzer` | Detecta ecosistemas para elegir scanners y manifiestos. |
| Audit | `audit/` | Registro de toda acción autónoma. |

## 3. Arquitectura

```
                 ┌───────────────── Harness ─────────────────┐
 SDKs Sentry ───▶│ Insights ingest ─▶ grouping ─▶ issues ─┐  │
                 │                                        │  │
 push / cron ───▶│ Shield scanners ─▶ findings ─▶ score ──┤  │
                 │                                        ▼  │
                 │                    Policy engine (umbral, │
                 │                    autonomía, presupuesto)│
                 │                                        │  │
                 │                Triage agent (LLM, sin   │  │
                 │                ejecutar código) ◀──────┘  │
                 │                         │ decide "fix"     │
                 │                         ▼                  │
                 │   AITask ─▶ Gitspace + Claude Code ─▶ PR   │
                 │                         │                  │
                 │   Pipeline + check shield ─▶ review/merge  │
                 └──────────────┬─────────────────────────────┘
                                ▼
                     AI Gateway ─▶ OpenRouter ─▶ N modelos
```

Hay dos tipos de agente:

1. **Triage agent** (en el server, sin ejecución de código). Hace llamadas LLM cortas para clasificar, deduplicar, explicar, estimar severidad, decidir si una vulnerabilidad es alcanzable y elegir commits sospechosos. Es barato y rápido.
2. **Fix agent** (en un Gitspace aislado). Es un `AITask` con Claude Code, u otro agente en el futuro, que clona el repo, reproduce el problema, escribe un test, aplica el fix, corre los tests y abre un PR.

## 4. Base compartida

### 4.1 AI Gateway (`app/services/aigateway`)

Es un paquete Go que encapsula OpenRouter. Toda feature de IA pasa por acá.

- **Cliente**: API compatible con OpenAI (`/api/v1/chat/completions`), streaming, tool calling y structured outputs (JSON schema) para respuestas parseables.
- **Perfiles por tarea**: en lugar de nombres de modelo dispersos en el código, cada llamada declara un perfil.

  | Perfil | Uso | Ejemplo de routing |
  |---|---|---|
  | `triage-fast` | clasificación y dedupe | modelo chico y barato, `sort: price` |
  | `reasoning` | causa raíz, alcanzabilidad | modelo grande con fallback |
  | `embed` | similitud de errores y de findings | modelo de embeddings |
  | `agent` | fix agent (Claude Code) | endpoint compatible con Anthropic |

  Los perfiles se configuran a nivel sistema y se pueden sobrescribir por space o repo.
- **Fallbacks**: se usa el array `models` de OpenRouter más reintentos con backoff propios.
- **Políticas de datos**: `provider.data_collection: "deny"`, filtro de proveedores ZDR y allowlist o denylist de proveedores por space. Esto es crítico porque los eventos de error pueden contener PII.
- **Redacción** antes de enviar: se eliminan secretos (reutilizando el detector de secret scanning), emails, tokens y headers de auth del payload.
- **Presupuestos**: límite mensual en USD por space, repo y feature. Se lleva contabilidad con el `usage` que devuelve OpenRouter. Al superar el límite, la feature se degrada a modo sin IA y no falla.
- **Métricas**: costo, tokens, latencia, modelo y tasa de fallback, expuestos en Prometheus (`app/services/metric`) y en una tabla `ai_usage`.
- **Claves**: BYOK por space mediante el gestor de secretos, con fallback a una clave de sistema en la config.

### 4.2 Fix agents vía OpenRouter

- Nuevo `enum.AIAgentAuth`: `openrouter-api-key`.
- `configure_claude_code.sh` acepta un `BaseURL` y un modelo opcionales: `ANTHROPIC_BASE_URL=https://openrouter.ai/api`, `ANTHROPIC_AUTH_TOKEN=<key>`, `ANTHROPIC_MODEL=<perfil agent>`. Hay que validar la compatibilidad exacta con la versión de Claude Code que se instale.
- `decryptAIAuth` deja de asumir Anthropic y lee el tipo de auth guardado.
- El `AITask` gana dos campos: `Origin` (manual, `insights` o `shield`) y `OriginRef`, el ID del issue o finding.
- Se agrega un *prompt template* versionado por tipo de tarea: fix de error, bump de dependencia o fix de SAST.
- Los permisos de Claude Code se restringen por tipo de tarea, partiendo de la allowlist actual del script, y se suman los comandos de test que detecte el agente o que declare el repo.

### 4.3 Policy engine y niveles de autonomía

Se configura por space con herencia a repo, siguiendo el patrón actual de `settings`:

| Nivel | Nombre | Qué hace el agente |
|---|---|---|
| 0 | Observe | Solo registra y clasifica. |
| 1 | Suggest | Comenta en el issue o PR con el análisis y un parche propuesto, sin tocar el repo. |
| 2 | Draft PR | Abre un PR draft desde una rama `agent/...`. Requiere review humano. **Default recomendado.** |
| 3 | Auto-merge | Igual que el nivel 2, pero activa automerge si el pipeline y los checks pasan y el cambio es de bajo riesgo (por ejemplo, un bump patch con tests verdes). |

Además se configuran los umbrales que disparan cada nivel (sección 5.4 y 6.4), los límites de concurrencia (máximo de agentes simultáneos por repo y por space), el presupuesto, ventanas horarias y un **kill switch** global y por repo.

### 4.4 Guardrails

- El agente actúa como el principal de servicio `harness-agent`. Nunca pushea a ramas protegidas: las protection rules existentes se aplican igual.
- **Prevención de loops**: si un PR del agente genera errores nuevos o regresiones, se marca, se suspende la autonomía para ese issue y se avisa.
- **Deduplicación**: un issue o finding tiene a lo sumo un `AITask` activo, y un PR abierto se actualiza en vez de crear otro.
- **Auditoría**: cada decisión del triage agent, con su input resumido, modelo, salida y costo, y cada acción del fix agent quedan en `audit`.
- **Evaluación offline**: un set de casos dorados (errores y CVEs con fix conocido) para medir los prompts y modelos antes de cambiarlos.

## 5. Insights: error tracking

### 5.1 Ingesta compatible con Sentry

La decisión clave es implementar el **protocolo de envelopes de Sentry** (`/api/{project}/envelope/` y `/store/`) con autenticación por DSN. Así cualquier SDK oficial de Sentry funciona solo con cambiar el DSN, y no hace falta escribir ni mantener SDKs propios.

- Un **proyecto Insights** pertenece a un repo de Harness. Así cada error está ligado al código.
- Hay rate limiting y sampling por proyecto, reutilizando `app/services/usage` y `limiter`.
- Se ingiere a un stream de eventos y se procesa de forma asíncrona.

### 5.2 Procesamiento

- **Fingerprinting determinístico**: tipo de excepción más frames "in-app" normalizados. Los embeddings (`embed`) son una segunda pasada para unir issues casi idénticos y no son el mecanismo principal.
- **Simbolicación y source maps**: los artefactos se suben al artifact registry desde el pipeline y se resuelven por `release`.
- **Releases**: `release` se asocia al commit SHA. Los pipelines de Harness lo inyectan automáticamente, de modo que cada issue sabe en qué commit apareció.
- **Estados del issue**: nuevo, en curso, resuelto, regresión e ignorado. La regresión se detecta cuando un issue resuelto reaparece en un release posterior al fix.

### 5.3 Triage agent

Se dispara cuando hay un issue nuevo, una regresión o un pico de frecuencia (por ejemplo, x10 en 15 minutos). Hace lo siguiente:

1. Resume el error en lenguaje natural.
2. Hace **suspect commits**: cruza los frames del stack con `git blame` y con los commits entre el último release sano y el actual. Esto es determinístico, y el LLM solo los rankea y explica.
3. Clasifica la severidad según usuarios afectados, frecuencia, si es una ruta crítica y si es una regresión.
4. Decide la acción: ignorar, notificar, sugerir o arreglar, con un `confidence` en 0..1 en salida estructurada.

### 5.4 Disparo del fix agent

El fix agent se dispara si se cumplen las tres condiciones:

- `severity >= umbral_severidad`, con default `high`.
- `confidence >= umbral_confianza`, con default 0.7.
- El nivel de autonomía del repo es 2 o más, y hay presupuesto y cupo de concurrencia.

El prompt del agente incluye el stack trace simbolizado, breadcrumbs y contexto redactados, los suspect commits y la instrucción de **escribir primero un test que reproduzca el error**. El PR resultante enlaza al issue, y el issue se cierra automáticamente cuando el release con el fix deja de reportarlo.

### 5.5 UI

Muestra la lista de issues por repo con gráfico de frecuencia, el detalle del issue (stack, contexto, suspect commits, análisis del agente) y el estado del `AITask` y el PR asociado. En la vista del PR aparece un banner que dice: "Este PR toca código involucrado en N errores activos".

## 6. Shield: vulnerabilidades

### 6.1 Scanners

Se usan herramientas open source maduras. El valor diferencial está en la orquestación y el agente, no en reinventar los scanners.

| Tipo | Herramienta propuesta | Disparo |
|---|---|---|
| SCA (dependencias) | OSV-Scanner y base OSV (con mirror local para entornos offline) | push a la rama default, PR y cron diario |
| SAST | Opengrep o Semgrep con reglas de la comunidad | PR y push |
| Secretos | el secret scanning existente | push (ya existe) |
| Imágenes y paquetes | Trivy o Grype sobre el artifact registry | push de imagen o paquete al registry |
| IaC | Trivy config | PR |

Los scanners corren como jobs (`job/`) en un runner aislado, o como steps de pipeline para repos con CI. Las bases de vulnerabilidades se sincronizan con un job recurrente.

### 6.2 Modelo de findings

Cada finding se deduplica por `(repo, tipo, regla o CVE, paquete y versión, o archivo y línea normalizada)` y tiene un ciclo de vida: abierto, validado, falso positivo, aceptado (con vencimiento), en fix y resuelto.

### 6.3 Score de riesgo

```
riesgo = CVSS base
       × factor EPSS (probabilidad de explotación)
       × 1.5 si está en CISA KEV
       × factor de alcanzabilidad (0.2 no alcanzable · 1.0 desconocido · 1.3 alcanzable)
       × factor de exposición (servicio público vs. tooling interno, configurable por repo)
```

Normalizado a 0–100.

### 6.4 Validación por el agente

Es la diferencia principal frente a un scanner tradicional:

1. **Alcanzabilidad**: el triage agent recibe el CVE (descripción y funciones afectadas según OSV o el advisory) y los usos del paquete en el repo, obtenidos con búsqueda de código (`keywordsearch`) y el grafo de imports. Decide si es alcanzable, no alcanzable o desconocido y justifica la decisión.
2. **Falsos positivos de SAST y secretos**: decide si el hallazgo es real según el contexto (código de test, valores de ejemplo, input ya sanitizado).
3. La decisión queda registrada y es **revisable por humanos**. Un humano puede marcar falso positivo y eso se usa como señal para futuras evaluaciones.

### 6.5 Parcheo

Si `riesgo >= umbral_parche` (default 70) y la autonomía es 2 o más:

- **Bump de dependencia**: primero se intenta de forma determinística, eligiendo la versión mínima que corrige el CVE y actualizando el lockfile con la herramienta del ecosistema. Si los tests pasan, se abre el PR directamente, sin LLM. Si fallan por breaking changes, se lanza el fix agent con el changelog de la dependencia y los errores de compilación o tests.
- **Fix de SAST**: se lanza el fix agent con la regla, el fragmento y la guía de remediación.
- **Secreto expuesto**: no se parchea automáticamente. Se notifica con urgencia, se sugiere rotar el secreto y se abre un PR que lo saca del código hacia el gestor de secretos.

Los PRs se agrupan por repo y ecosistema (por ejemplo, "Shield: 3 dependencias npm") para no inundar a los reviewers.

### 6.6 Gate de merge

- Hay un status check `shield/security` en cada PR que falla si el PR **introduce** findings por encima del umbral de bloqueo.
- Se integra con las protection rules existentes: se puede marcar como check requerido.
- Se agregan comentarios inline en el PR con la explicación generada por el triage agent.

### 6.7 UI

Muestra un dashboard por space (riesgo agregado, tendencias, MTTR), la lista de findings por repo con filtros por tipo, riesgo y estado, el detalle con la explicación del agente y la evidencia de alcanzabilidad, y la bandeja de PRs abiertos por Shield.

## 7. Modelo de datos (nuevas tablas)

| Tabla | Campos clave |
|---|---|
| `insights_projects` | id, repo_id, dsn_public_key, rate_limit, created |
| `insights_issues` | id, project_id, fingerprint, title, level, status, first_seen, last_seen, event_count, user_count, first_release, resolved_in_release, ai_summary, ai_severity, ai_confidence |
| `insights_events` | id, issue_id, received, release, environment, payload (redactado y comprimido) con retención configurable; a gran escala se movería a blob storage |
| `shield_scans` | id, repo_id, commit_sha, scanner, status, started, finished |
| `shield_findings` | id, repo_id, type, rule_or_cve, package, version, path, line, cvss, epss, kev, reachability, risk_score, status, ai_verdict, ai_reasoning, accepted_until |
| `agent_actions` | id, origin (`insights` o `shield`), origin_id, ai_task_id, pr_id, level, decision, cost_usd, created |
| `ai_usage` | id, space_id, repo_id, feature, profile, model, tokens_in, tokens_out, cost_usd, created |

También se agregan campos nuevos en `ai_tasks`: `origin` y `origin_ref`. Las migraciones van en `app/store/database/migrate/{postgres,sqlite}` a partir de la `0192`.

## 8. Eventos nuevos

- `insights`: `issue_created`, `issue_regressed`, `issue_spiked`, `issue_resolved`
- `shield`: `scan_completed`, `finding_opened`, `finding_validated`, `finding_resolved`
- Consumo de eventos existentes: `pipeline.executed` (asocia releases y verifica fixes), eventos de `pullreq` (merge del PR del agente) y eventos de push de `git`.

Estos eventos también se exponen como **webhooks** (`app/services/webhook`) para integrar con Slack o PagerDuty.

## 9. API (resumen)

```
POST   /api/{project}/envelope/                         # ingesta compatible con Sentry
GET    /api/v1/repos/{repo}/insights/issues
GET    /api/v1/repos/{repo}/insights/issues/{id}
PATCH  /api/v1/repos/{repo}/insights/issues/{id}        # estado, asignación
POST   /api/v1/repos/{repo}/insights/issues/{id}/fix    # lanzar el agente manualmente
GET    /api/v1/repos/{repo}/shield/findings
PATCH  /api/v1/repos/{repo}/shield/findings/{id}        # falso positivo, aceptar riesgo
POST   /api/v1/repos/{repo}/shield/scans                # scan on-demand
POST   /api/v1/repos/{repo}/shield/findings/{id}/fix
GET    /api/v1/spaces/{space}/ai/usage
PUT    /api/v1/spaces/{space}/settings  (ai.*, insights.*, shield.*)
```

## 10. Fases

| Fase | Contenido | Resultado |
|---|---|---|
| **0. Base IA** (2–3 semanas) | AI Gateway, perfiles, presupuestos, redacción, `ai_usage`, auth `openrouter-api-key` para `AITask`, principal `harness-agent`, settings de autonomía y kill switch | Claude Code en Gitspaces funcionando vía OpenRouter, con costos visibles |
| **1. Shield MVP** (3–4 semanas) | SCA con OSV, findings, score sin alcanzabilidad, check `shield/security`, PRs de bump determinísticos (nivel 2) | Detección más parches automáticos de dependencias |
| **2. Insights MVP** (4–5 semanas) | ingesta compatible con Sentry, fingerprinting, issues, releases ligados a commits, UI básica | Error tracking utilizable sin IA |
| **3. Triage agents** (3 semanas) | resúmenes, severidad, suspect commits (Insights), alcanzabilidad y falsos positivos (Shield), set de evaluación offline | Menos ruido y priorización real |
| **4. Fix agents** (3–4 semanas) | `AITask` disparado por políticas, prompts por tipo de tarea, test-first, prevención de loops, auditoría | Errores y vulnerabilidades corregidos de forma autónoma con PR draft |
| **5. Expansión** | SAST, imágenes del registry, IaC, source maps, nivel 3 (automerge), alertas y webhooks, embeddings para dedupe | Paridad funcional con Sentry y Snyk en lo esencial |

Shield va antes que Insights porque es más acotado, reutiliza más piezas existentes (eventos de push, checks, protection rules) y da un caso de uso autónomo completo antes, gracias al bump determinístico.

## 11. Métricas de éxito

- **Shield**: el MTTR de findings críticos y la tasa de falsos positivos descartados por el agente (validada por humanos).
- **Insights**: el tiempo desde la primera ocurrencia hasta un PR con fix y el porcentaje de regresiones detectadas.
- **Agentes**: el porcentaje de PRs del agente que se mergean sin cambios, que se mergean con cambios y que se cierran; el costo por PR mergeado; los incidentes causados por agentes, cuyo objetivo es 0.
- **Gateway**: la tasa de fallback, la latencia p95 por perfil y la desviación respecto al presupuesto.

## 12. Riesgos y mitigaciones

| Riesgo | Mitigación |
|---|---|
| Un agente introduce bugs | Nivel 2 por defecto, test-first, pipeline obligatorio, prevención de loops, kill switch |
| Fuga de PII o secretos a proveedores LLM | Redacción previa, proveedores ZDR, allowlist de proveedores, opción de desactivar la IA por repo |
| Costos descontrolados | Presupuestos con degradación, perfiles baratos para triage, bump sin LLM cuando sea posible |
| Volumen de ingesta de errores | Sampling, rate limits, retención corta y payloads en blob storage |
| Ruido de PRs | Agrupación, límites de concurrencia, un PR por issue o finding |
| Prompt injection desde el contenido del error o del repo | Tratar todo input como datos, salidas estructuradas, permisos mínimos del agente en el Gitspace y sin acceso a secretos del repo |
| Licencias de los scanners y las reglas | Revisar las licencias de Semgrep y Opengrep y de las reglas antes de empaquetarlos |

## 13. Preguntas abiertas

1. ¿Las features se distribuyen en el binario de Harness Open Source o como un módulo opcional?
2. ¿La clave de OpenRouter es BYOK por space solamente, o también hay una clave global del administrador?
3. ¿Se necesita soporte para entornos air-gapped? Eso implica un mirror de OSV y modelos locales vía un endpoint compatible con OpenAI configurable en el gateway.
4. ¿El fix agent será solo Claude Code o se suman otros agentes (Codex, Aider, OpenHands) usando el mapa `installationMap` existente?
5. ¿Hay algún volumen de ingesta objetivo que obligue a usar ClickHouse o similar desde la fase 2?
