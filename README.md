# kokaq server

This is the production server component of kokaq. It hosts APIs, integrates with backend storage, exposes metrics, and runs the scheduling/dispatch logic using `kokaq-core`.

---

## Responsibilities
- REST/AMQP API gateway
- Connection to backend storage
- SLA-aware priority dispatch loop
- Metrics and tracing integration (OpenTelemetry)
- Auth/AuthZ hooks for secure multi-tenant usage
