# ADR-0005: Message bus is NATS JetStream

- **Status**: Accepted
- **Date**: 2026-02-02

## Context
Нужны асинхронные процессы:
- ingestion/обновление каталога
- обновление индекса поиска
- пересчёт агрегатов рейтинга
- retries, DLQ

Приоритеты: простота эксплуатации (small team), низкая latency, возможность роста.

## Decision
- Используем **NATS JetStream** как message bus и task queue.
- События публикуются через **Outbox pattern** (будет реализовано позже) для гарантий доставки.
- Используем DLQ (отдельный stream/subject) для сообщений, которые не удалось обработать после N попыток.

## Consequences
### Positive
- простой старт и эксплуатация по сравнению с Kafka
- достаточная функциональность для event-driven архитектуры на начальных и средних нагрузках

### Negative
- экосистема вокруг Kafka богаче (streams processing, tooling)
- при экстремальном росте может понадобиться миграция (но контракты событий останутся)

## Alternatives considered
- Kafka: сильнее ecosystem, но дороже и сложнее для 1 человека.
- RabbitMQ: хорошая очередь, но для стриминга/retention и pub/sub сценариев JetStream удобнее.
