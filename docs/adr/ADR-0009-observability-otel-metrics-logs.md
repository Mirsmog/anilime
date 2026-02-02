# ADR-0009: Observability baseline (OTel + metrics + logs)

- **Status**: Accepted
- **Date**: 2026-02-02

## Context
Система состоит из нескольких сервисов и внешних зависимостей (Jikan, streaming providers, Stripe). Для эксплуатации нужны:
- трассировка запросов end-to-end
- метрики latency/error rate/queue depth
- структурированные логи с корреляцией

## Decision
- Внедряем **OpenTelemetry** (traces) во все сервисы.
- Экспорт метрик в **Prometheus** (через OTel или напрямую).
- Логи — structured JSON (zap), обязателен `request_id`/correlation id.

## Consequences
### Positive
- быстрый поиск проблем, понимание p95/p99
- проще соблюдение SLO и диагностика внешних зависимостей

### Negative
- добавляет инфраструктуру (otel-collector, prom, grafana)
- требует дисциплины в метриках и именовании

## Alternatives considered
- Только логи: плохо для распределённых трасс.
- Только метрики: не дают конкретного пути запроса.
