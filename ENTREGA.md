# HNL Bank — Resumen de Entrega

**Repositorio:** https://github.com/AyathAriel/hnl-bank
**Cómo correrlo:** clonar, `cp .env.example .env` (agregar `ANTHROPIC_API_KEY` para el chat), `docker compose up --build`. Frontend en `http://localhost:5173`.

## Stack

Go 1.25 · Vue 3 + Vite · TigerBeetle · PostgreSQL · Docker Compose · Anthropic API + MCP (`modelcontextprotocol/go-sdk`)

## Qué incluye

- **Autenticación completa**: registro, login, JWT con revocación real en logout, y **2FA (TOTP)** opcional por usuario — probado con app autenticadora real.
- **Cuentas y transacciones**: depósito, retiro (con validación de fondos), transferencia, historial paginado — todo respaldado por TigerBeetle con partida doble real.
- **Chat con IA vía MCP** (obligatorio): consulta de saldo/historial y operaciones bancarias en lenguaje natural, con confirmación obligatoria antes de ejecutar cualquier depósito/retiro/transferencia — reforzada en el servidor, no solo en el prompt.
- **Seguridad**: bcrypt, SQL 100% parametrizado, rate limiting, superficie de red mínima (solo frontend y API expuestos), contenedor backend sin privilegios.
- **Bonus**: paginación, exportar historial a CSV, gráfica de evolución de saldo, notificaciones en tiempo real por WebSocket, 18 tests unitarios, diseño responsive verificado en móvil/tablet/desktop, y config de deploy lista para Render (no ejecutada por decisión: TigerBeetle requiere disco persistente, que implica costo en cualquier proveedor).

## Credenciales de prueba

Cualquier usuario del dataset sirve para iniciar sesión (contraseñas con el patrón `Nombre2024!`).
Dataset completo de 980 usuarios de prueba en `backend/seed-data/`, se carga automático al primer
arranque — ver ahí los emails disponibles.

## Documentación completa

Arquitectura, decisiones de diseño ante ambigüedades del enunciado, todos los endpoints, variables de entorno y guía de pruebas: ver [README.md](README.md) en el repositorio.
