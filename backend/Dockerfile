# TigerBeetle-go usa cgo (bindings nativos), por lo que evitamos Alpine/musl y
# usamos Debian (glibc) tanto para compilar como para ejecutar, eliminando el
# riesgo de incompatibilidad de libc entre build y runtime.
FROM golang:1.25-bookworm AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1
RUN go build -o /out/api ./cmd/api
RUN go build -o /out/mcpserver ./cmd/mcpserver

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*

# Usuario sin privilegios: el proceso api/mcpserver no necesita root para nada
# (solo abre sockets salientes y un puerto no privilegiado).
RUN groupadd --system hnl && useradd --system --gid hnl --no-create-home hnl

WORKDIR /app
COPY --from=build /out/api ./api
COPY --from=build /out/mcpserver ./mcpserver
COPY seed-data ./seed-data
RUN chown -R hnl:hnl /app

USER hnl

ENV MCP_SERVER_BIN=/app/mcpserver
EXPOSE 8080

CMD ["./api"]
