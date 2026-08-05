# HNL Bank — Sistema de Banca en Línea

Prueba técnica: sistema de banca en línea con backend en **Go**, frontend en **Vue 3 + Vite**,
ledger financiero en **TigerBeetle**, datos de usuarios/autenticación en **PostgreSQL**, todo
orquestado con **Docker Compose**, y un **chat con IA vía MCP** (Anthropic API) capaz de ejecutar
operaciones bancarias en lenguaje natural con confirmación obligatoria antes de acciones críticas.

## Estado del proyecto

Verificado end-to-end (build limpio desde cero con `docker compose down -v && docker compose up --build`, sin editar nada a mano):

- ✅ Los 4 contenedores levantan y quedan sanos con un solo `docker compose up`.
- ✅ Seed automático: 980 usuarios, cuentas y 6429 transacciones históricas cargadas correctamente (ver nota sobre emails duplicados más abajo).
- ✅ Login, registro, logout (con revocación real de JWT), dashboard, depósito, retiro (incluye rechazo por fondos insuficientes), transferencia e historial paginado — probados vía API y vía UI en navegador.
- ✅ El endpoint `/api/chat` y el servidor MCP (`cmd/mcpserver`) están completos y correctamente conectados: la petición llega a la API de Anthropic con el modelo, las tools y la autenticación correctos.
- ⚠️ **Para que el chat responda de verdad hace falta que la cuenta de Anthropic asociada a `ANTHROPIC_API_KEY` tenga saldo/crédito disponible.** Con la key configurada actualmente, Anthropic devuelve `"Your credit balance is too low to access the Anthropic API"` — es un tema de facturación de la cuenta, no un bug del proyecto (la petición llega bien formada hasta la API). Para probarlo: agrega crédito en [console.anthropic.com → Plans & Billing](https://console.anthropic.com) y reinicia el backend (`docker compose up -d backend`) para que tome la key actualizada si cambiaste el `.env`.

## Stack técnico

| Capa | Tecnología |
|---|---|
| Backend | Go 1.25 (chi, pgx/v5, tigerbeetle-go, golang-jwt, bcrypt, go-playground/validator) |
| Frontend | Vue 3 + Vite + Pinia + vue-router + Tailwind CSS |
| Ledger financiero | TigerBeetle (single-node, cluster de desarrollo) |
| Usuarios / Auth | PostgreSQL 16 |
| IA / Chat | Anthropic Messages API + servidor MCP propio (`modelcontextprotocol/go-sdk`) |
| Infraestructura | Docker Compose (4 servicios) |

## Cómo levantar el proyecto

Requisitos: Docker y Docker Compose.

```bash
cp .env.example .env
# Edita .env y agrega tu ANTHROPIC_API_KEY si quieres probar el chat con IA
docker compose up --build
```

- Frontend: http://localhost:5173
- API backend: http://localhost:8080 (health check en `/health`)
- TigerBeetle: puerto 3000 (interno)
- PostgreSQL: puerto 5432 (interno, expuesto también en el host para inspección)

Al primer arranque, el backend aplica las migraciones de PostgreSQL y **siembra automáticamente**
el dataset de prueba (`backend/seed-data/datos-prueba-HNL.json`: 1000 usuarios, 1605 cuentas,
6429 transacciones históricas). Este proceso es idempotente: en arranques posteriores, si ya hay
usuarios en la base, se omite.

### Credenciales de prueba

Cualquier usuario del dataset sirve para iniciar sesión. Por ejemplo:

- **Email:** `ihernandez@email.com` — **Password:** `Isabel2024!`
- **Email:** `mjimenez@example.com` — **Password:** `Miguel2024!`

(Las contraseñas siguen el patrón `Nombre2024!` para cada usuario del JSON de prueba; puedes
revisar `backend/seed-data/datos-prueba-HNL.json` para ver el resto de emails.)

También puedes registrar una cuenta nueva desde `/register`.

## Arquitectura

```
┌─────────────┐     HTTP/JSON      ┌──────────────┐
│   Frontend   │ ─────────────────▶ │   Backend    │
│  Vue 3 + Vite │ ◀───────────────── │   Go (API)   │
└─────────────┘                    └──────┬───────┘
                                           │
                     ┌─────────────────────┼─────────────────────┐
                     ▼                     ▼                     ▼
             ┌───────────────┐    ┌────────────────┐   ┌──────────────────┐
             │  PostgreSQL   │    │   TigerBeetle   │   │  Anthropic API    │
             │ usuarios/auth │    │ balances/ledger │   │  (chat con IA)    │
             │ historial/aud.│    │ partida doble   │   └─────────┬────────┘
             └───────────────┘    └────────────────┘             │
                                                                   │ tool-use
                                           ┌───────────────────────┘
                                           ▼
                                 ┌─────────────────────┐
                                 │  cmd/mcpserver (Go)  │
                                 │  servidor MCP stdio  │
                                 │  tools bancarias     │
                                 └─────────────────────┘
```

### Por qué dos bases de datos

- **TigerBeetle** es la única fuente de verdad para dinero: balances y transferencias con partida
  doble real. Las cuentas de cliente se crean con el flag `DebitsMustNotExceedCredits`, así que el
  propio ledger rechaza sobregiros (además de la validación de la aplicación) — no es posible que
  un retiro o transferencia deje un balance negativo, ni con un bug en el backend.
- **PostgreSQL** guarda usuarios, credenciales, la relación cuenta↔usuario y una copia de auditoría
  de cada transacción (para listados rápidos, paginación y filtros sin ir a TigerBeetle en cada
  consulta).
- Existe una cuenta de control especial `EXTERNAL` en TigerBeetle que representa fondos que entran
  o salen del banco: los depósitos son transferencias `EXTERNAL → cuenta` y los retiros
  `cuenta → EXTERNAL`.

### Chat con IA vía MCP

El chat no llama directamente a la lógica de negocio: el backend levanta un **subproceso MCP real**
(`cmd/mcpserver`, protocolo stdio/JSON-RPC) por cada mensaje, anclado al usuario autenticado. Ese
servidor expone las tools `list_accounts`, `get_balance`, `get_transaction_history`, `deposit`,
`withdraw` y `transfer`. El backend actúa como host/cliente MCP: lista las tools, se las ofrece al
modelo de Anthropic vía tool-use, y ejecuta el loop de llamadas hasta obtener una respuesta final.

Las operaciones críticas (`deposit`, `withdraw`, `transfer`) exigen un parámetro `confirmed=true`.
Si el modelo intenta ejecutarlas sin confirmación previa, la tool responde `needs_confirmation` sin
tocar ningún dato — el modelo debe primero describir la operación y preguntar al usuario, y solo
reintentar con `confirmed=true` después de que el usuario confirme explícitamente en el chat
siguiente. Esta regla está reforzada **en el servidor**, no solo en el prompt.

## Decisiones de diseño ante ambigüedades del enunciado

1. **`initial_balance` del JSON = saldo actual de cada cuenta.** Se carga en TigerBeetle mediante
   una única transferencia de apertura (`EXTERNAL → cuenta`). Las 6429 transacciones históricas del
   JSON se insertan en PostgreSQL como registro de auditoría/historial para la UI (con sus
   timestamps originales), pero **no se re-ejecutan contra TigerBeetle**: el dataset sintético no
   garantiza que `initial_balance` sea el resultado contable exacto de reproducir ese historial, y
   forzarlo produciría rechazos de sobregiro arbitrarios contra un ledger que sí exige partida doble
   real. Toda transacción **nueva** creada por un usuario (vía API o chat) sí pasa íntegramente por
   TigerBeetle.
2. **Montos como enteros (centavos) en TigerBeetle**, decimales exactos (`NUMERIC(18,2)`) en
   PostgreSQL y strings decimales (`"100.50"`) en la API JSON, para evitar errores de redondeo de
   punto flotante en dinero.
3. **Logout real:** JWT stateless + tabla `revoked_tokens` (blacklist por `jti`), no solo borrado en
   el cliente.
4. **Historial de chat en memoria** por `conversation_id`, no persistido tras reiniciar el backend.
   No es un requisito funcional persistirlo entre despliegues.
5. **Sesión MCP por mensaje:** cada turno del chat levanta y cierra un subproceso `mcpserver`. Es la
   forma más simple y robusta de evitar procesos huérfanos, al costo de un pequeño overhead de
   arranque por turno — aceptable para el alcance de esta prueba.
6. **Número de cuenta:** formato `4001-XXXX-XXXX-XXXX`, igual al del dataset de prueba, generado
   aleatoriamente con verificación de unicidad al registrar un usuario nuevo.
7. **Emails duplicados en el dataset de prueba:** el JSON provisto tiene 20 emails que se repiten
   entre usuarios distintos (con `id` diferente). Como el email debe ser único para el login, el
   seed conserva la primera aparición de cada email y descarta las siguientes (junto con sus
   cuentas asociadas): de 1000 usuarios y 1605 cuentas del JSON, se cargan 980 usuarios y 1575
   cuentas. Las 6429 transacciones históricas se cargan íntegras de todas formas (son solo registro
   de auditoría, sin restricción de integridad referencial). Queda registrado en los logs del
   backend al arrancar (`seed: skipped N user(s) with duplicate email`).

## Endpoints de la API

Todos los endpoints (salvo auth) requieren `Authorization: Bearer <token>`.

| Método | Ruta | Descripción |
|---|---|---|
| POST | `/api/auth/register` | Registra un usuario y crea su cuenta corriente |
| POST | `/api/auth/login` | Inicia sesión, devuelve un JWT |
| POST | `/api/auth/logout` | Revoca el JWT actual |
| GET | `/api/accounts` | Lista las cuentas del usuario con saldo actual |
| GET | `/api/accounts/{number}` | Detalle de una cuenta propia |
| GET | `/api/accounts/{number}/balance-history` | Serie histórica de saldo (para la gráfica) |
| POST | `/api/transactions/deposit` | `{account_number, amount, description}` |
| POST | `/api/transactions/withdraw` | `{account_number, amount, description}` |
| POST | `/api/transactions/transfer` | `{from_account_number, to_account_number, amount, description}` |
| GET | `/api/transactions?account=&page=&page_size=` | Historial paginado |
| GET | `/api/transactions/export?account=` | Historial completo como CSV descargable |
| GET | `/api/dashboard` | Resumen: cuentas, saldo total, transacciones recientes |
| POST | `/api/chat` | `{conversation_id, message}` → chat con IA |
| GET | `/api/ws?token=` | WebSocket de notificaciones en tiempo real |
| GET | `/health` | Health check |

## Seguridad

- Contraseñas con **bcrypt** (costo 12).
- **JWT** (HS256, 24h) + blacklist de revocación en PostgreSQL para logout real.
- Validación de entrada en todos los DTOs (`go-playground/validator`) y de formato de montos; ningún
  mensaje de error expone detalles internos (stack traces, SQL, rutas de archivo) al cliente.
- **SQL 100% parametrizado** (pgx, placeholders `$1..$n`) en toda la aplicación — cero concatenación
  de strings en queries, cero superficie de SQL injection. Auditado explícitamente (`grep` de
  patrones de concatenación en queries: sin resultados).
- Verificación de propiedad de cuenta en cada operación (no se puede operar sobre cuentas ajenas);
  todas las rutas de negocio están detrás del middleware de autenticación — solo `/api/auth/*`,
  `/health` y el handshake de `/api/ws` (que valida el JWT dentro del propio handler) quedan fuera.
- **Rate limiting** (token bucket por IP): `/api/auth/*` (registro/login) y también en
  depósito/retiro/transferencia, para frenar abuso automatizado de operaciones financieras.
- Límite de tamaño de body (1 MiB) en todas las peticiones, para evitar payloads gigantes.
- CORS restringido al origin del frontend; cabeceras de seguridad en backend
  (`X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `Cache-Control: no-store`) y en el
  nginx del frontend (mismas cabeceras + `Content-Security-Policy` restrictiva).
- **Superficie de red mínima:** PostgreSQL y TigerBeetle **no exponen puertos al host** — solo son
  alcanzables dentro de la red interna de docker-compose. Únicamente el frontend (5173) y la API
  (8080) quedan accesibles desde fuera.
- **Contenedores sin privilegios:** el backend corre como usuario sin privilegios (no root) dentro
  de su contenedor.
- Defensa en profundidad a nivel de ledger: el flag `DebitsMustNotExceedCredits` de TigerBeetle
  rechaza sobregiros incluso si hubiera un bug en la validación de la aplicación.
- Confirmación server-side obligatoria para operaciones críticas iniciadas por el chat con IA (no se
  confía únicamente en el prompt del modelo).

## Variables de entorno

Ver [.env.example](.env.example). Las más relevantes:

- `DATABASE_URL`, `TB_ADDRESS` — conexión a Postgres y TigerBeetle.
- `JWT_SECRET`, `JWT_EXPIRY_HOURS` — firma y expiración de tokens.
- `ANTHROPIC_API_KEY`, `ANTHROPIC_MODEL` — credenciales para el chat con IA. Si `ANTHROPIC_API_KEY`
  está vacía, el resto del sistema funciona con normalidad y `/api/chat` responde `503` indicando
  que el chat no está configurado.
- `CORS_ORIGIN` — origin permitido para llamadas cross-origin al backend.

## Estructura del repositorio

```
backend/
  cmd/api/            servidor HTTP principal (REST + arranque/seed/migraciones)
  cmd/mcpserver/       servidor MCP (stdio) con las tools bancarias
  internal/config/     carga de variables de entorno
  internal/db/         pool de Postgres + migraciones embebidas
  internal/ledger/      wrapper sobre TigerBeetle (cuentas, transferencias, balances)
  internal/banking/     lógica de dominio (única capa de negocio, usada por REST y MCP)
  internal/auth/        bcrypt, JWT, middleware, blacklist de revocación
  internal/httpapi/     router y handlers REST
  internal/mcptools/    definición de las tools MCP
  internal/chat/        cliente Anthropic + cliente MCP + orquestador del chat
  internal/seed/        carga del dataset de prueba
  seed-data/            datos-prueba-HNL.json (dataset provisto)
frontend/
  src/api/              cliente axios
  src/stores/            Pinia (auth, accounts, transactions, chat)
  src/views/             Login, Register, Dashboard, Transactions, History
  src/components/        NavBar, AccountCard, TransactionForm, TransactionList, ChatWidget, Toast
tigerbeetle/             Dockerfile + entrypoint.sh (format-si-hace-falta + start)
docker-compose.yml
```

## Desarrollo local sin Docker (opcional)

Backend (requiere Go 1.25, PostgreSQL y TigerBeetle corriendo aparte):

```bash
cd backend
go run ./cmd/api
```

Frontend:

```bash
cd frontend
npm install
npm run dev
```

## Pruebas manuales sugeridas

1. Inicia sesión con un usuario sembrado y verifica que el saldo mostrado sea el esperado.
2. Registra un usuario nuevo, deposita fondos, retira (prueba también un retiro mayor al saldo —
   debe rechazarse con un mensaje claro) y transfiere a otra cuenta.
3. Revisa `/history`: pagina, filtra por cuenta.
4. En el chat del dashboard, pregunta "¿cuánto dinero tengo?" y luego pide "transfiere $10 a la
   cuenta &lt;otra cuenta&gt;" — el asistente debe pedir confirmación antes de ejecutar, y el saldo
   solo debe cambiar después de confirmar.

## Bonus implementados

| Bonus | Estado | Detalle |
|---|---|---|
| **Paginación** | ✅ | `GET /api/transactions?page=&page_size=`, con controles de página en `/history`. |
| **Exportar (CSV)** | ✅ | `GET /api/transactions/export?account=` descarga el historial completo como CSV; botón "Exportar CSV" en `/history`. |
| **Gráficas** | ✅ | Gráfica de evolución de saldo en el dashboard, usando el historial de balances point-in-time de TigerBeetle (flag `History` en las cuentas de cliente). SVG propio, sin dependencias externas. |
| **WebSockets** | ✅ | `GET /api/ws?token=` — hub por usuario en el backend; notifica en tiempo real (toast + refresco automático del dashboard) cuando se completa un depósito, retiro o transferencia vía REST. *Alcance: las operaciones ejecutadas desde el chat con IA no disparan esta notificación (el propio chat ya informa el resultado en la misma respuesta); ampliar esto es una extensión menor si se necesita.* |
| **Deploy** | ⚙️ Preparado | [`render.yaml`](render.yaml) (Render Blueprint) con los 4 servicios, discos persistentes para TigerBeetle/Postgres y variables de entorno. El Dockerfile del frontend soporta `VITE_API_URL` como build arg para plataformas donde los servicios no comparten red por nombre como en docker-compose. **No se ejecutó el deploy real** porque requiere una cuenta en la plataforma elegida — avísame si quieres que lo haga y con qué proveedor. |
| Rate limiting | ✅ | Ver sección Seguridad. |
| Logs estructurados | Parcial | Logs con contexto (`chi/middleware.Logger` + logs propios de arranque/seed/errores); no son JSON estructurado. |
| Testing, CI/CD, 2FA, App móvil | ❌ | Fuera del alcance de esta entrega. |

## Limitaciones conocidas

- El historial de conversaciones del chat vive en memoria del proceso backend (se pierde al
  reiniciar).
- Las 6429 transacciones históricas del dataset son solo registro de auditoría (ver decisión #1);
  no representan movimientos reales replicados en TigerBeetle.
- Las notificaciones WebSocket solo se disparan para operaciones hechas vía REST/UI, no desde el chat.
- No se implementó 2FA (fuera del alcance obligatorio de la prueba).
