<div align="center">
  <img height="300" src="https://github.com/kokaq/.github/blob/main/kokaq-server.png" alt="cute quokka as kokaq logo"/>
</div>

`server` is the production server component of `kokaq`. It hosts APIs, integrates with backend storage, exposes metrics, and runs the scheduling/dispatch logic using `core`.

[![Go Reference](https://pkg.go.dev/badge/github.com/kokaq/server.svg)](https://pkg.go.dev/github.com/kokaq/server)
[![Tests](https://github.com/kokaq/server/actions/workflows/go.yml/badge.svg)](https://github.com/kokaq/server/actions/workflows/go.yml)

## Responsibilities
- REST/AMQP API gateway
- Connection to backend storage
- SLA-aware priority dispatch loop
- Metrics and tracing integration (OpenTelemetry)
- Auth/AuthZ hooks for secure multi-tenant usage
