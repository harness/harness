# Plan: errores y vulnerabilidades con agentes autónomos vía OpenRouter (solo modelos gratuitos)

Estado: propuesta v2 · Autor: Claude Code a pedido de @juhasztobias · Fecha: 2026-10-25

Cambios respecto de v1:
- Se usan **solo modelos gratuitos** de OpenRouter, siempre y sin excepciones.
- **Mammoth 1** (`baseten/mammoth-1:free`) es el "Sistema 1": toma las decisiones y elige qué modelo hace cada tarea.
- Se agrega un **módulo de Tareas** para registrar y seguir cada issue.
- El diseño ahora contempla las cuotas del plan gratuito.

## Estado de implementación

Todo lo listado abajo está implementado, probado con fakes y sqlite, y deshabilitado por defecto.

| Área | Qué hay | Paquetes |
|---|---|---|
| IA gratuita (AI Gateway) | Guardia solo-gratuito; catálogo persistido en `ai_models` con cuarentenas; cuota en base compartida entre réplicas con reservas por clase; reintentos con backoff; redacción con gitleaks; Mammoth → Elephant; cache; escalamiento por confianza con segunda opinión; elección de modelo `model_for_task`; latencia p95; atribución de uso; eventos `ai.*`. Migración 0192. | `app/services/aigateway` |
| Contrato de decisiones y preguntas | Contrato de la Decisions API; preguntas tipadas (Shield, Insights y genéricas) | `aigateway/decisions`, `aigateway/questions` |
| Capa común de decisiones | Registra decisiones en `ai_decisions` y en el historial; respaldo por reglas; modos de IA | `app/services/aidecide` |
| Modos de IA por repo | Setting `ai_mode` | `aigateway/mode.go` |
| API de IA | `/api/v1/system/ai/models` y `/quota` | `/api/v1/system/ai/*` |
| Tareas | Estados, historial, SLA y reapertura; API REST por repo y por space; eventos y webhooks `task_created`, `task_updated`, `task_resolved`; labels; enlace con PRs (`Fixes TASK-n`); sugerencia de asignación por CODEOWNERS. Migraciones 0193 y 0197 (índice único de deduplicación). | `app/services/task`, `app/events/task`, `app/services/tasklink`, `app/services/taskassign`, `/api/v1/repos/{repo}/tasks`, `/api/v1/spaces/{space}/tasks` |
| Shield | Escaneo de dependencias con OSV (Go, npm, PyPI); riesgo como CVSS × EPSS × KEV × alcanzabilidad × exposición, normalizado contra CVSS 10; SAST con SARIF de Opengrep/Semgrep; Trivy para imágenes e IaC; sync de datos; disparo en push a la rama por defecto; creación de tareas; API `/shield/scans` y `/shield/findings`. Migración 0194. | `app/services/shield`, `shield/osv`, `shield/risk`, `shield/sast`, `shield/trivy` |
| Insights | Ingesta de envelopes Sentry, DSN y fingerprinting; endpoints `POST /api/{project}/envelope/` y `/store/`; API de administración de proyectos, issues y eventos; suspect commits; source maps; triage (Mammoth → tareas, duplicados, prioridad); verificación y reapertura automática. Migración 0195. | `insights/ingest`, `insights/suspect`, `insights/sourcemap`, `insights/triage`, `insights/verify` |
| Agentes | Principal `harness-agent`; kill switch, autonomía, loops y audit; lanzamiento de fix agent por política (`lanzar_fix_agent`); presupuesto de requests por AITask (migración 0196); Claude Code con modelo gratuito vía OpenRouter. | `app/services/agentguard`, `app/services/agentpolicy`, `app/services/aitaskbudget`, `orchestrator/ai_auth.go` |

**Pendiente**

- En curso:
  - Notificaciones de Tareas.
  - Bumps de dependencias sin LLM.
  - UI de Tareas.
  - Check de Shield/security en PRs.
  - Triage de Shield con Mammoth.
  - Lanzamiento de AITask por política.
  - Reanudación de `esperando_cuota`.
- Bloqueado por red o key:
  - Prueba real de la Decisions API.
  - Spike de Claude Code.
  - Calibración con set de evaluación.
  - Agente real abriendo PRs.
  - Auto-merge.
  - Embeddings.
- Las migraciones de postgres no están probadas en este entorno (solo sqlite).

## Orden de ejecución

Primero va todo lo que no tiene bloqueos externos, sin importar la fase, respetando las dependencias internas. Se asume D5 = dentro del binario, apagado por defecto. Una tarea bloqueada pasa al frente apenas se libera su bloqueo.

**Bloqueos externos:**
- **R1**: `openrouter.ai` está denegado en el entorno de desarrollo y no hay key.
- **R2**: los feeds de EPSS (`api.first.org`) y KEV (`cisa.gov`) están denegados, igual que las descargas desde `github.com`.
- **D1 a D5**: las preguntas abiertas de la sección 16.

### A. Sin bloqueos

| # | Tarea | Fase | Depende de |
|---|---|---|---|
| 1 | Conectar `aigateway.WireSet` y regenerar `wire_gen.go` | 0 | — |
| 2 | Migraciones 0192+ y stores de `ai_models`, `ai_decisions`, `ai_usage` y `ai_quota` | 0 | — |
| 3 | Cuota persistida y compartida entre réplicas | 0 | 2 |
| 4 | Reintentos con backoff ante un 429, contando cada intento | 0 | — |
| 5 | Eventos `ai.quota_exhausted` y `ai.paid_model_detected`, más audit | 0 | 1 |
| 6 | Modos de IA por repo (default `solo-decisiones` en privados; D2 puede cambiarlo después) | 0 | 1 |
| 7 | API `/system/ai/models` y `/system/ai/quota` | 0 | 1, 3 |
| 8 | Claude Code recibe un modelo gratuito del catálogo en `decryptAIAuth` | 0 | 1 |
| 9 | Escalamiento por confianza (umbrales alto y bajo, Sistema 2) | 3 | 1 |
| 10 | `model_for_task`: filtro, pregunta a Mammoth y desempate por tasa histórica | 3 | 2, 9 |
| 11 | Guardrails: principal `harness-agent`, kill switch, prevención de loops, audit | 5 | — |
| 12 | Presupuesto de requests por `AITask` y campos nuevos en `ai_tasks` | 5 | 2, 3 |
| 13 | Migraciones y stores de `tasks`, `task_activities`, `task_labels` y `task_links` | 1 | — |
| 14 | Máquina de estados y numeración `TASK-n` | 1 | 13 |
| 15 | API REST de Tareas | 1 | 14 |
| 16 | Historial de la tarea | 1 | 13 |
| 17 | Labels, eventos y webhooks de Tareas | 1 | 15 |
| 18 | Notificaciones | 1 | 15 |
| 19 | `Fixes TASK-n`: vincular el PR y cerrar la tarea al mergear | 1 | 14 |
| 20 | Responsable sugerido por codeowners | 1 | 15 |
| 21 | Decisiones de Mammoth registradas en el historial | 3 | 2, 16 |
| 22 | Tablas de Shield (scans y findings) con `task_id` | 2 | 13 |
| 23 | Escaneo de dependencias con OSV (librería Go y base de OSV, ambas accesibles) | 2 | 22 |
| 24 | Bumps de versión sin LLM y apertura del PR | 2 | 23, 19 |
| 25 | Ingesta compatible con Sentry y autenticación por DSN | 4 | 13 |
| 26 | Fingerprinting, releases y regresiones | 4 | 25 |
| 27 | Suspect commits con git blame | 4 | 26 |
| 28 | Source maps en el artifact registry | 6 | 26 |

### B. Se programan ya, pero la validación final está bloqueada o depende de una decisión

| # | Tarea | Fase | Depende de | Bloqueo |
|---|---|---|---|---|
| 29 | UI de Tareas: lista, kanban y detalle | 1 | 15, 16 | D4 define el alcance |
| 30 | Riesgo con CVSS, EPSS y KEV, probado con datos de prueba | 2 | 23 | R2 para los feeds reales |
| 31 | Check `shield/security` en las protection rules | 2 | 30 | R2 |
| 32 | Preguntas de Shield (`reachable`, `false_positive`, `fix_strategy`) | 3 | 10, 22 | R1 para probar contra Mammoth real |
| 33 | Preguntas de Insights y creación de tareas | 4 | 10, 15, 27 | R1 |
| 34 | Verificación automática de tareas | 5 | 24, 26 | — |
| 35 | Policy engine (`lanzar_fix_agent`) | 5 | 6, 11, 12, 32, 33 | R1 |
| 36 | Panel del agente | 6 | 7, 29 | D4 |
| 37 | SAST, imágenes del registry e IaC, vía módulos Go o PyPI | 6 | 22 | R2 para los binarios de github.com |

### C. Bloqueadas

| # | Tarea | Fase | Bloqueo |
|---|---|---|---|
| 38 | Probar la Decisions API real contra el contrato | 0 | R1 |
| 39 | Validar el límite de crédito en la key (capa 5) | 0 | Acción del dueño de la cuenta de OpenRouter |
| 40 | Prueba de Claude Code con modelos gratuitos | 0 | R1 y un entorno de Gitspace |
| 41 | Set de evaluación y calibración de umbrales | 3 | Casos etiquetados por personas y R1 |
| 42 | Agente real abriendo PRs | 5 | R1, 40 y D1 |
| 43 | Auto-merge (nivel 3) | 6 | 42 |
| 44 | Embeddings gratuitos | 6 | R1 y que exista un modelo gratuito de embeddings |

## 1. Objetivo

Agregar a Harness tres productos que comparten una misma base de IA:

- **Insights**: error tracking estilo Sentry, integrado con el código, los pipelines y los releases.
- **Shield**: análisis de vulnerabilidades estilo Snyk (dependencias, código, secretos, imágenes del registry).
- **Tareas**: un registro único de los problemas detectados (errores, vulnerabilidades y tareas manuales), con estado, responsable (humano o agente), prioridad e historial.

Cuando un problema supera un umbral, un **agente** lo valida, lo corrige y abre un PR. Todo queda registrado en su tarea. **Toda inferencia usa modelos gratuitos de OpenRouter.** Las decisiones las toma Mammoth 1, y la generación de texto y código la hacen modelos gratuitos elegidos por Mammoth para cada tarea.

## 2. Restricciones del plan gratuito y cómo las absorbe el diseño

| Restricción | Detalle | Respuesta del diseño |
|---|---|---|
| Cuota diaria | 50 requests por día sin créditos; **1000 por día** si la cuenta compró al menos 10 USD de créditos alguna vez. Los requests fallidos también cuentan. | Planificador de cuota global, con reservas por clase de trabajo y caminos determinísticos sin LLM siempre que se pueda (sección 4.4). Recomendación: cargar 10 USD una única vez para pasar a 1000 por día. Los créditos no se consumen porque solo se usan modelos gratuitos. |
| Ritmo | 20 requests por minuto | Token bucket local a 18 por minuto con margen; las colas se procesan por prioridad. |
| La cuota es por cuenta | Usar otro modelo `:free` o generar otra API key no aumenta el límite. | Una sola cuota compartida por toda la instalación. No se intenta esquivarla con varias cuentas, que además iría contra los términos de OpenRouter. |
| El catálogo cambia | Los modelos gratuitos aparecen, desaparecen o cambian de proveedor. | Un job sincroniza el catálogo y Mammoth elige entre los gratuitos disponibles en ese momento. No hay modelos fijos en el código (sección 4.2). |
| Privacidad | Algunos proveedores gratuitos exigen permitir logging o entrenamiento con los prompts. Esto **reemplaza** el requisito de v1 de usar solo proveedores que no retienen datos. | Redacción obligatoria antes de cada envío, IA desactivable por repo, y un modo "solo decisiones" para repos sensibles (sección 4.6). |
| Decisions API en alpha | El endpoint `/api/alpha/decisions` puede cambiar. | Adaptador aislado con tests de contrato y degradación a reglas determinísticas si falla (sección 5.5). |
| Contexto de Mammoth | 16K tokens | Se construye un `state` compacto: resúmenes, top-N frames y metadatos, no payloads crudos. |

## 3. Lo que ya existe y se reutiliza

| Pieza existente | Dónde | Uso en este plan |
|---|---|---|
| `AITask`: Claude Code dentro de un Gitspace | `types/ai_task.go`, `app/services/aitaskevent`, `app/gitspace/orchestrator` | Runtime de los fix agents. Ya registra tokens y modelos en `AIUsageMetric`. |
| `AIAgentAuth` (hoy solo `anthropic-api-key`) y `installationMap` | `types/enum/ai_agent_auth.go`, `orchestrator/utils/ai_agent.go` | Se agrega `openrouter-api-key` y la posibilidad de enchufar agentes alternativos. |
| `configure_claude_code.sh` | `orchestrator/utils/script_templates/` | Apunta Claude Code a OpenRouter con el modelo gratuito que elija Mammoth. |
| Sistema de eventos | `events/`, `app/events/*` | Nuevas categorías `insights`, `shield` y `task`. |
| Scheduler de jobs | `job/` | Scans, sync del catálogo de modelos, cola de trabajo con cuota. |
| Principals de servicio | `app/bootstrap` | Nuevo principal `harness-agent`. |
| Protection rules y status checks | `app/services/protection` | Gate de merge `shield/security`. |
| Automerge | `app/services/automerge` | Nivel de autonomía 3. |
| Secret scanning | `app/services/settings/security.go` | Es un scanner más de Shield y también se usa para redactar prompts. |
| Settings de repo y space | `app/services/settings` | Umbrales, autonomía, reservas de cuota y modo de privacidad. |
| Labels de space y repo | `types/label.go`, `app/services/label` (hoy ligadas a PRs) | Se extienden a tareas con `label_task`, siguiendo el patrón de `label_pullreq`. |
| Notificaciones | `app/services/notification` | Asignación, cambios de estado y vencimiento de SLA de tareas. |
| Gestor de secretos | `app/services/secret` | Guarda la API key de OpenRouter. |
| Artifact registry | `registry/` | Source maps (Insights) y escaneo de imágenes (Shield). |
| Audit | `audit/` | Registro de toda acción autónoma. |

Harness no tiene hoy un tracker de issues, así que el módulo de Tareas es nuevo, aunque se apoya en labels, notificaciones y eventos que ya existen.

## 4. Base compartida: AI Gateway solo gratuito

Es un paquete Go en `app/services/aigateway` por el que pasa toda llamada a un modelo. Tiene tres clientes:

- **Decisions**: `POST /api/alpha/decisions`, para Mammoth 1 y Elephant 1.
- **Chat**: `POST /api/v1/chat/completions`, compatible con OpenAI, para la generación de texto con modelos gratuitos.
- **Configuración de agentes**: base URL, token y modelo para Claude Code u otro agente dentro del Gitspace.

### 4.1 Garantía de "siempre gratuito"

La regla se aplica en varias capas, y cualquiera de ellas alcanza para bloquear un modelo pago:

1. **Allowlist del catálogo**: solo se aceptan modelos con `pricing.prompt == 0` y `pricing.completion == 0` en `/api/v1/models`, o los que tienen sufijo `:free`, más el router `openrouter/free`.
2. **Precio máximo en cada request**: se envía `provider.max_price` con prompt y completion en 0, para que OpenRouter nunca enrute a un endpoint pago.
3. **Validación en el gateway**: el gateway rechaza cualquier ID de modelo fuera de la allowlist, aunque venga de una configuración de usuario o de la salida de Mammoth.
4. **Verificación posterior**: si `usage.cost > 0` en una respuesta, se dispara una alerta, el modelo se pone en cuarentena y el evento queda en el audit log.
5. **Límite de crédito en la API key**: la key de OpenRouter se crea con un límite de crédito mínimo como red de seguridad. Hay que validar que un límite en 0 no bloquee los modelos gratuitos.

**No hay fallback a modelos pagos.** Si no queda cuota ni modelos gratuitos disponibles, el trabajo espera en la cola o se resuelve por el camino determinístico.

### 4.2 Catálogo de modelos gratuitos

El job `ai-catalog-sync` corre cada 6 horas y guarda en `ai_models` los modelos gratuitos disponibles. Para cada uno registra:

- **Capacidades**, tomadas del catálogo: contexto máximo, soporte de tool calling, structured outputs y modalidades.
- **Rendimiento observado**, calculado por Harness: tasa de éxito por tipo de tarea (por ejemplo, PR mergeado o JSON válido), latencia p50 y p95, y tasa de errores 429 y 5xx.
- **Estado**: activo, degradado, en cuarentena o retirado.

Mammoth recibe estos datos como descripción de cada opción. Así elige con información actual y no con un listado fijo.

### 4.3 Clases de modelo

| Clase | Qué hace | Modelos |
|---|---|---|
| **Sistema 1 (decisión)** | sí/no, elegir entre opciones, puntuar | Mammoth 1 (`baseten/mammoth-1:free`), con Elephant 1 (`baseten/elephant-1:free`, 9B) como respaldo o para decisiones triviales |
| **Sistema 2 (generación)** | resúmenes, explicaciones, análisis de causa raíz | modelos de chat gratuitos elegidos por Mammoth |
| **Agente (código)** | reproducir, testear y parchear dentro del Gitspace | modelo gratuito con tool calling elegido por Mammoth |
| **Embeddings** | dedupe semántico de errores y findings | solo si hay un modelo de embeddings gratuito; si no, dedupe solo con fingerprints determinísticos |
| **Último recurso** | cuando falla la elección de Mammoth o no hay candidatos | el router `openrouter/free`, solo para tareas sin tool calling |

### 4.4 Planificador de cuota

Con 1000 requests por día para toda la instalación, la cuota es el recurso más escaso, así que la gestiona un componente propio:

- **Contador global** sincronizado entre réplicas mediante `pubsub` o `lock` y persistido. Lee también los headers de rate limit de OpenRouter cuando están disponibles.
- **Reservas por clase**, configurables. Un reparto inicial razonable:

  | Clase | Reserva diaria |
  |---|---|
  | Decisiones (Mammoth y Elephant) | 25 % |
  | Generación del triage | 25 % |
  | Fix agents | 40 % |
  | Interacciones manuales (por ejemplo, "explicar esto" en la UI) | 10 % |

  Lo que una clase no usa se libera para las demás a partir del mediodía UTC.
- **Prioridad de la cola**, de mayor a menor: tarea crítica, luego regresión, luego tarea alta, luego el resto. Las tareas de un mismo issue se agrupan.
- **Presupuesto por fix agent**: cada `AITask` tiene un tope de requests (por ejemplo, 60). Si lo alcanza, se detiene, guarda su progreso en la tarea y pasa a revisión humana.
- **Ahorro antes que todo**:
  - Cache de decisiones por hash del `state`.
  - Bumps de dependencias sin LLM.
  - Dedupe por fingerprint antes de consultar a ningún modelo.
  - Varias preguntas agrupadas en una sola llamada a la Decisions API.
- Cuando la cuota se agota, el trabajo queda en estado `esperando_cuota` dentro de la tarea y se reanuda al reiniciarse el día UTC.

### 4.5 Fix agents con modelos gratuitos

- Nuevo `enum.AIAgentAuth`: `openrouter-api-key`.
- `configure_claude_code.sh` recibe `BaseURL`, `AuthToken` y `Model`: `ANTHROPIC_BASE_URL=https://openrouter.ai/api`, `ANTHROPIC_AUTH_TOKEN=<key>`, `ANTHROPIC_MODEL=<modelo gratuito elegido por Mammoth>`.
- **Spike en la fase 0**: hay que verificar que Claude Code funcione bien con modelos gratuitos vía el endpoint compatible con Anthropic de OpenRouter, sobre todo en tool calling y en contexto largo. Si no rinde, se agrega un agente alternativo compatible con OpenAI (por ejemplo, Aider u OpenHands) usando el `installationMap` existente. El runner es agnóstico al agente.
- `AITask` gana los campos `TaskID`, que lo enlaza al módulo de Tareas, y `ModelDecisionID`, que registra qué decisión eligió el modelo.
- Hay prompt templates versionados por tipo de tarea: fix de error, bump con breaking changes y fix de SAST. Todos siguen el enfoque test-first.

### 4.6 Privacidad con proveedores gratuitos

Como algunos proveedores gratuitos pueden registrar o entrenar con los datos, la privacidad se resuelve del lado de Harness:

- **Redacción obligatoria** antes de cada envío: secretos (con el detector de secret scanning), emails, IPs, tokens, cookies, headers de auth y valores de variables locales en los stack traces.
- **Modo de IA por repo**:

  | Modo | Qué se envía |
  |---|---|
  | `completo` | todo lo redactado, incluido código (fix agents) |
  | `solo-decisiones` | solo metadatos compactos a Mammoth; ni código ni payloads a modelos generativos; no hay fix agents |
  | `apagado` | nada |

  Los repos privados arrancan en `solo-decisiones` y un admin puede subirlos a `completo`.
- La UI avisa de forma clara que los modelos gratuitos pueden retener datos según el proveedor.

## 5. Sistema 1: Mammoth como cerebro de decisiones

### 5.1 Idea

Se usa la división Sistema 1 / Sistema 2:

- **Sistema 1 (Mammoth)** responde rápido, barato y con probabilidades calibradas: ¿es un duplicado?, ¿qué tan grave es?, ¿se arregla solo?, ¿qué modelo lo hace?
- **Sistema 2 (modelos generativos y agentes)** se usa solo cuando hace falta pensar o escribir: explicar, analizar la causa raíz o programar.

Como Mammoth devuelve probabilidades y no texto, cada decisión tiene una **confianza numérica** que se compara directamente con los umbrales del policy engine.

### 5.2 Decisiones que toma Mammoth

Cada fila es una pregunta de la Decisions API, y varias se agrupan en un solo request:

| Momento | Pregunta (`id`) | Tipo | Opciones |
|---|---|---|---|
| Error nuevo | `is_duplicate_of_candidate` | noul | el error se compara con hasta 3 issues candidatos por fingerprint cercano |
| Error nuevo | `severity` | score | bajo, medio, alto, crítico |
| Error nuevo | `action` | choice | `ignorar`, `notificar`, `sugerir`, `arreglar` |
| Error nuevo | `suspect_commit` | choice | top-5 commits sospechosos (git blame) más `ninguno` |
| Finding nuevo | `reachable` | choice | `alcanzable`, `no_alcanzable`, `desconocido` |
| Finding nuevo | `false_positive` | noul | el hallazgo es real o un falso positivo |
| Finding nuevo | `fix_strategy` | choice | `bump_deterministico`, `agente`, `manual`, `aceptar_riesgo` |
| Cualquier trabajo generativo | `model_for_task` | choice | los modelos gratuitos candidatos (sección 5.3) |
| Antes de lanzar un fix agent | `worth_agent_budget` | noul | si vale la pena gastar la cuota del agente hoy, considerando cuota restante, prioridad y probabilidad de éxito |
| Tras un intento fallido | `next_step` | choice | `reintentar_otro_modelo`, `escalar_humano`, `abandonar` |
| Tarea nueva | `assignee_kind` | choice | `agente`, `equipo_X`, `code_owner` (con `codeowners`) |

### 5.3 Elección de modelo

1. **Filtro determinístico**: se descartan los modelos que no cumplen los requisitos duros (contexto mínimo, tool calling si es un agente, estado activo, sin cuarentena).
2. **Pregunta `model_for_task` a Mammoth**:
   - El `state` describe la tarea: tipo, lenguaje, tamaño del diff o contexto, dificultad estimada e intentos previos con sus modelos.
   - Los `criteria` describen cada candidato: capacidades más la tasa de éxito observada para ese tipo de tarea.
3. **Uso de la confianza**:
   - Si `confidence >= 0.6`, se usa el modelo elegido.
   - Si es menor, se usa el modelo con mejor tasa de éxito histórica para ese tipo de tarea, como desempate determinístico.
4. **Registro**: la decisión se guarda en `ai_decisions` y su resultado (éxito o fallo) vuelve al catálogo como rendimiento observado. Así, con el tiempo, Mammoth recibe descripciones cada vez más precisas.

### 5.4 Escalamiento por confianza

Cada decisión tiene dos umbrales configurables, `alto` y `bajo`:

- **Confianza mayor o igual a `alto`**: se actúa según el nivel de autonomía.
- **Confianza entre `bajo` y `alto`**: se consulta al Sistema 2 (un modelo generativo gratuito) para obtener un segundo análisis con justificación, y se vuelve a preguntar a Mammoth con ese análisis incluido en el `state`.
- **Confianza menor que `bajo`**: la tarea se asigna a un humano con el análisis disponible.

### 5.5 Robustez

- **Adaptador aislado** (`aigateway/decisions`) con el contrato de request y respuesta de la Decisions API, verificado con tests de contrato. El parseo es estricto: siempre se verifica `type`, y una clave faltante cuenta como error.
- **Respaldo**: si Mammoth falla, se usa Elephant 1. Si fallan los dos, se aplican reglas determinísticas (CVSS más KEV para findings, frecuencia más regresión para errores) y la decisión queda marcada como `fallback`.
- **Set de evaluación**: un conjunto de casos etiquetados por humanos (errores y CVEs conocidos) sirve para calibrar umbrales y detectar regresiones cuando cambia la versión de Mammoth. El campo `model` de la respuesta trae el build exacto.

## 6. Módulo de Tareas

### 6.1 Propósito

Es la fuente única de verdad del seguimiento de trabajo. Cada problema relevante de Insights o Shield genera o actualiza una tarea, y también se pueden crear tareas a mano. Toda acción, humana o del agente, queda en el historial de la tarea.

### 6.2 Modelo

- **Alcance**: la tarea pertenece a un repo, con vista agregada por space.
- **Campos**: título, descripción, `origen` (`insights`, `shield` o `manual`), `origen_ref`, prioridad (P0 a P3), estado, responsable (un usuario, un grupo con `usergroup` o el agente), labels, fecha límite según el SLA de la prioridad, PRs vinculados y `AITask` vinculados.
- **Estados**:
  - `abierta` → `triage` → `en_curso` → `en_revision` → `resuelta` → `verificada`
  - Estados laterales: `esperando_cuota`, `bloqueada`, `descartada` (con motivo) y `reabierta`.
- **Transiciones automáticas**:
  - Al abrir un PR del agente o de una persona que la referencia, pasa a `en_revision`.
  - Al mergear el PR, pasa a `resuelta`.
  - Pasa a `verificada` cuando Insights deja de recibir el error en el release con el fix, o cuando Shield no detecta el finding en el siguiente scan.
  - Si el error reaparece, pasa a `reabierta` y su prioridad sube.
- **Deduplicación**: un issue de Insights o un finding de Shield tiene a lo sumo una tarea activa. Si Mammoth detecta un duplicado, la tarea se vincula a la existente.
- **Agrupación**: los findings del mismo paquete o los errores con la misma causa raíz pueden agruparse en una tarea padre con subtareas.

### 6.3 Historial

Es una línea de tiempo por tarea, con estas entradas:

- Creación con su origen y datos clave.
- Cada decisión de Mammoth con sus probabilidades. Por ejemplo: "severidad: alto (0.81) · acción: arreglar (0.74) · modelo: X (0.66)".
- Inicio y fin de cada `AITask`, con el modelo usado y los requests consumidos.
- PRs abiertos y mergeados, y resultados del pipeline.
- Cambios de estado, de responsable y de prioridad.
- Comentarios de personas.

Esto da trazabilidad completa de por qué el sistema hizo cada cosa.

### 6.4 Integraciones

- **Labels**: se reutiliza `app/services/label` con un nuevo `label_task`.
- **Notificaciones**: asignación, cambio de estado, SLA por vencer y "el agente necesita ayuda".
- **Webhooks**: `task_created`, `task_updated` y `task_resolved`, para Slack o integraciones externas.
- **PRs**: la sintaxis `Fixes TASK-123` en el título o la descripción vincula el PR y cierra la tarea al mergear, reutilizando el patrón de `autolink`.
- **Codeowners**: sugiere el responsable humano según los archivos del stack o del finding.

### 6.5 UI

- **Lista** de tareas por repo y space, con filtros por estado, prioridad, origen, responsable (incluido "agente") y label.
- **Tablero kanban** por estado.
- **Detalle**: descripción, origen (con link al issue o finding), historial, PRs y estado del agente en vivo.
- **Panel del agente**: tareas en curso, cuota usada y restante del día, cola `esperando_cuota` y tasa de éxito por modelo.

## 7. Insights: error tracking

Sigue lo definido en v1, con estos ajustes.

### 7.1 Ingesta compatible con Sentry
Se implementa el protocolo de envelopes de Sentry (`/api/{project}/envelope/`) con autenticación por DSN, así que funcionan los SDKs oficiales de Sentry. Cada proyecto pertenece a un repo, y hay rate limiting y sampling por proyecto.

### 7.2 Procesamiento
- Fingerprinting determinístico (tipo de excepción más frames "in-app" normalizados).
- Source maps en el artifact registry.
- Releases ligados al commit SHA, inyectado por los pipelines.
- Detección de regresiones.
- **Antes de usar IA**: los suspect commits se calculan de forma determinística (frames del stack cruzados con `git blame` y los commits entre releases), y Mammoth solo elige entre ellos.

### 7.3 Flujo con IA
1. Llega un issue nuevo, una regresión o un pico de frecuencia.
2. Se dedupe por fingerprint, sin gastar cuota.
3. Mammoth recibe un request agrupado con `is_duplicate_of_candidate`, `severity`, `action` y `suspect_commit`.
4. Se crea o actualiza la tarea con la prioridad derivada de `severity`.
5. Si `action = arreglar` y se cumplen confianza, autonomía, modo de IA `completo` y `worth_agent_budget`, Mammoth elige el modelo (`model_for_task`) y se lanza un `AITask` con test-first.
6. Se abre un PR que referencia la tarea. La verificación es automática por release.

## 8. Shield: vulnerabilidades

Sigue lo definido en v1, con estos ajustes.

### 8.1 Scanners
OSV-Scanner (SCA), Opengrep o Semgrep (SAST), el secret scanning existente, Trivy o Grype (imágenes del registry y IaC). Todos corren como jobs o como steps de pipeline. Las bases de vulnerabilidades se sincronizan con un job.

### 8.2 Score de riesgo
```
riesgo = CVSS × factor EPSS × (1.5 si está en KEV) × factor de alcanzabilidad × factor de exposición
```
Normalizado a 0–100. La alcanzabilidad sale de la pregunta `reachable` de Mammoth y se pondera por su confianza:
- `alcanzable`: factor 1.3
- `desconocido`: factor 1.0
- `no_alcanzable`: factor 0.2, aplicado solo si la confianza es alta; si no, se usa 1.0

### 8.3 Flujo
1. Se escanea y se dedupe el finding.
2. Se calcula el riesgo base de forma determinística.
3. Mammoth responde `reachable`, `false_positive` y `fix_strategy` en un único request.
4. Se crea o actualiza la tarea.
5. Si `riesgo >= umbral_parche` (default 70):
   - Con `bump_deterministico`, se elige la versión mínima que corrige el CVE, se actualiza el lockfile con la herramienta del ecosistema y se abre el PR **sin consumir cuota de LLM**.
   - Si ese bump rompe los tests, o si la estrategia es `agente`, se lanza el fix agent con el modelo elegido por Mammoth.
   - Con un secreto expuesto, no se parchea: se notifica con urgencia y se crea una tarea P0 para rotarlo.
6. El check `shield/security` bloquea los PRs que introducen findings por encima del umbral, en integración con las protection rules.

## 9. Policy engine y autonomía

Los cuatro niveles siguen igual que en v1: 0 = observar, 1 = sugerir, 2 = PR draft (es el default), 3 = auto-merge.

Las condiciones para que un agente actúe solo se juntan en un único punto de decisión:

```
lanzar_fix_agent =
     autonomía(repo) >= 2
  && modo_ia(repo) == completo
  && decision(action|fix_strategy).confidence >= umbral_alto
  && prioridad(tarea) >= umbral_prioridad
  && cuota.disponible(clase=agentes) >= presupuesto_agente
  && decision(worth_agent_budget) == true
  && agentes_activos(repo) < máximo
  && !kill_switch
```

Se mantienen los guardrails de v1:
- Principal `harness-agent`, sin push a ramas protegidas.
- Prevención de loops.
- Un PR por tarea.
- Audit log de cada acción.
- Kill switch global y por repo.

## 10. Modelo de datos (nuevas tablas)

| Tabla | Campos clave |
|---|---|
| `tasks` | id, repo_id, number (por repo, `TASK-123`), title, description, origin, origin_ref, priority, state, assignee_principal_id, assignee_group_id, assignee_is_agent, due, parent_id, created_by, created, updated, resolved, verified |
| `task_activities` | id, task_id, kind (state_change, decision, agent_run, pr_link, comment…), payload JSON, principal_id, created |
| `task_labels` | task_id, label_id, value_id (patrón de `pullreq_labels`) |
| `task_links` | task_id, kind (pullreq, ai_task, insights_issue, shield_finding), ref_id |
| `ai_models` | id (ID de OpenRouter), kind (decision, chat, embed), context_length, supports_tools, status, success_rate_by_task JSON, p95_latency_ms, last_seen, quarantined_reason |
| `ai_decisions` | id, task_id, question_id, model_build, state_hash, answer JSON (probabilidades y confianza), fallback bool, outcome, created |
| `ai_quota` | day, class, used, reserved |
| `ai_usage` | id, space_id, repo_id, feature, model, requests, tokens_in, tokens_out, cost_usd (debe ser 0), created |
| `insights_projects`, `insights_issues`, `insights_events` | igual que en v1, más `task_id` |
| `shield_scans`, `shield_findings` | igual que en v1, más `task_id` y `ai_decision_ids` |

Nuevos campos en `ai_tasks`: `task_id`, `model_decision_id` y `request_budget`. Las migraciones empiezan en la `0192`, en `app/store/database/migrate/{postgres,sqlite}`.

## 11. Eventos nuevos

- `task`: `created`, `state_changed`, `assigned`, `resolved`, `verified`, `reopened`
- `insights`: `issue_created`, `issue_regressed`, `issue_spiked`
- `shield`: `scan_completed`, `finding_opened`
- `ai`: `decision_made`, `quota_exhausted`, `paid_model_detected` (alerta; no debería ocurrir nunca)

Todos se exponen también como webhooks.

## 12. API (resumen)

```
# Tareas
GET    /api/v1/repos/{repo}/tasks
POST   /api/v1/repos/{repo}/tasks
GET    /api/v1/repos/{repo}/tasks/{number}
PATCH  /api/v1/repos/{repo}/tasks/{number}               # estado, responsable, prioridad
GET    /api/v1/repos/{repo}/tasks/{number}/activities
POST   /api/v1/repos/{repo}/tasks/{number}/comments
POST   /api/v1/repos/{repo}/tasks/{number}/agent         # lanzar el agente manualmente (respeta la cuota)
GET    /api/v1/spaces/{space}/tasks                      # vista agregada

# Insights y Shield
POST   /api/{project}/envelope/                          # ingesta compatible con Sentry
GET    /api/v1/repos/{repo}/insights/issues
GET    /api/v1/repos/{repo}/shield/findings
PATCH  /api/v1/repos/{repo}/shield/findings/{id}
POST   /api/v1/repos/{repo}/shield/scans

# IA
GET    /api/v1/system/ai/models                          # catálogo gratuito y rendimiento
GET    /api/v1/system/ai/quota                           # cuota del día por clase
GET    /api/v1/repos/{repo}/ai/decisions?task={number}
PUT    /api/v1/spaces/{space}/settings  (ai.*, tasks.*, insights.*, shield.*)
```

## 13. Fases

| Fase | Contenido | Resultado |
|---|---|---|
| **0. Base IA gratuita** (3 semanas) | AI Gateway (decisions y chat), allowlist gratuita con todas sus capas, sync del catálogo, planificador de cuota, redacción, modos de IA por repo, auth `openrouter-api-key`, **spike de Claude Code con modelos gratuitos**, adaptador de Mammoth y Elephant con tests de contrato | Infraestructura que garantiza solo modelos gratuitos y gestiona la cuota |
| **1. Tareas** (2–3 semanas) | tablas, API, estados, historial, labels, notificaciones, vínculo con PRs (`Fixes TASK-n`), UI de lista, kanban y detalle | Seguimiento de trabajo usable aunque no haya IA |
| **2. Shield MVP** (3–4 semanas) | OSV, findings, riesgo determinístico, check `shield/security`, bumps sin LLM, creación de tareas | Detección y parches de dependencias con costo de cuota cero |
| **3. Sistema 1** (2–3 semanas) | preguntas de Mammoth para Shield (`reachable`, `false_positive`, `fix_strategy`), `model_for_task`, escalamiento por confianza, registro de decisiones en el historial, set de evaluación | Decisiones calibradas y elección dinámica de modelos |
| **4. Insights MVP** (4–5 semanas) | ingesta compatible con Sentry, fingerprinting, releases, suspect commits, preguntas de Mammoth, creación de tareas | Error tracking con triage automático |
| **5. Fix agents** (3–4 semanas) | `AITask` disparado por políticas, presupuesto de requests por agente, test-first, prevención de loops, verificación automática de tareas | Ciclo autónomo completo: se detecta el problema, se crea la tarea, Mammoth decide, el agente abre el PR y la tarea se verifica |
| **6. Expansión** | SAST, imágenes del registry, IaC, source maps, nivel 3 (automerge), embeddings gratuitos si existen, panel del agente | Paridad funcional con Sentry y Snyk en lo esencial |

El módulo de Tareas va antes que Shield e Insights porque los dos crean tareas, y el historial es lo que hace auditable lo que decide Mammoth.

## 14. Métricas de éxito

- **Costo**: USD gastado igual a **0**, siempre. Cualquier valor distinto es un incidente.
- **Cuota**: el porcentaje de la cuota diaria usado de forma útil (requests que terminan en una decisión aplicada o en un PR) y los días con `quota_exhausted`.
- **Mammoth**: la precisión de sus decisiones contra el set de evaluación y contra las correcciones humanas, la calibración (si confianza 0.8 acierta cerca del 80 % de las veces) y la tasa de fallback.
- **Elección de modelo**: la tasa de éxito por modelo y por tipo de tarea y su evolución.
- **Tareas**: el MTTR por prioridad, el porcentaje de tareas resueltas por el agente y verificadas, y las reaperturas.
- **Agentes**: el porcentaje de PRs del agente que se mergean sin cambios, que se mergean con cambios y que se cierran, y los requests consumidos por PR mergeado.

## 15. Riesgos y mitigaciones

| Riesgo | Mitigación |
|---|---|
| 1000 requests por día no alcanzan para muchos fix agents | Caminos determinísticos primero, `worth_agent_budget`, presupuesto por agente, cola priorizada. Lo esperable es que solo se arreglen solas las tareas de mayor prioridad cada día. |
| Los modelos gratuitos rinden peor en código que los pagos | Test-first con pipeline obligatorio, autonomía 2 por defecto, rendimiento observado que alimenta a Mammoth y escalamiento a humanos |
| Claude Code puede no funcionar bien con modelos gratuitos | Spike en la fase 0 y runner agnóstico con un agente alternativo |
| Los proveedores gratuitos retienen datos | Redacción obligatoria, `solo-decisiones` por defecto en repos privados, aviso en la UI |
| Desaparece un modelo gratuito, incluido Mammoth | Sync del catálogo, Elephant 1 como respaldo y reglas determinísticas como último recurso |
| Cambia la Decisions API (está en alpha) | Adaptador aislado, tests de contrato y versión de modelo registrada en cada decisión |
| Se cuela un modelo pago | Las cinco capas de la sección 4.1 y una alerta si `cost > 0` |
| Prompt injection desde errores o código | Todo input se trata como datos; Mammoth solo puede elegir entre opciones cerradas, lo que reduce la superficie de ataque; permisos mínimos del agente |
| El rate limit de 20 por minuto genera ráfagas de 429 | Token bucket a 18 por minuto y reintentos con backoff, contando que los 429 consumen cuota |

## 16. Preguntas abiertas

1. ¿Se carga la compra única de 10 USD para pasar de 50 a 1000 requests por día? Sin eso, el sistema solo sirve para decisiones y no para fix agents.
2. ¿Los repos privados arrancan en `solo-decisiones`, o la organización acepta enviar código redactado a proveedores gratuitos?
3. ¿Es aceptable que, cuando no queda cuota, el trabajo espere al día siguiente, o se quiere una cola de prioridad manual para casos urgentes?
4. ¿El módulo de Tareas debe ser genérico para cualquier trabajo del equipo, o solo para lo que generan Insights y Shield?
5. ¿Las features se distribuyen dentro del binario de Harness Open Source o como un módulo opcional?

## 17. Referencias

- Mammoth 1 (free): https://openrouter.ai/baseten/mammoth-1:free
- Elephant 1 (free): https://openrouter.ai/baseten/elephant-1:free
- Router de modelos gratuitos: https://openrouter.ai/openrouter/free
- Decisions API: https://openrouter.ai/docs/api/api-reference/alphadecisions/submit-a-decisions-request
- Límites y precios: https://openrouter.ai/pricing
