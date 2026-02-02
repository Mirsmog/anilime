# ADR-0001: Monorepo and shared platform packages

- **Status**: Accepted
- **Date**: 2026-02-02

## Context
Проект состоит из множества сервисов (BFF, auth, catalog, ingestion, streaming, activity, social, billing). Для разработки одним человеком важны:
- единая сборка и запуск локально
- минимизация дублирования инфраструктурного кода (логирование, конфиг, HTTP сервер, shutdown)
- возможность безопасного рефакторинга контрактов и общих утилит

## Decision
- Используем **monorepo**: все сервисы в одном репозитории.
- Общий инфраструктурный код размещаем в `internal/platform/*`.
- Запрещаем шарить бизнес-логику между доменами через общие пакеты; шарим только platform/infrastructure.

## Consequences
### Positive
- меньше дублирования кода
- проще CI/CD и локальная разработка
- проще массовые изменения (например внедрение tracing)

### Negative
- сервисы сильнее связаны на уровне репозитория
- требуется дисциплина: не превращать `internal/platform` в "свалку"

## Alternatives considered
1. **Multi-repo** + versioned shared libs — больше операционной сложности и затрат времени.
2. **Полная изоляция без shared кода** — приведёт к копипасте (особенно middlewares/observability).
