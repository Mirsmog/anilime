# ADR-0008: Billing uses Stripe webhooks as source of truth

- **Status**: Accepted
- **Date**: 2026-02-02

## Context
Подписки/оплаты должны быть корректными и проверяемыми. Клиент не может быть источником истины о факте оплаты.
Stripe предоставляет webhooks с событием и уникальным `event.id`.

## Decision
- **Billing service** — владелец данных о подписке.
- Источник истины: **Stripe webhooks** (подпись проверяется `STRIPE_WEBHOOK_SECRET`).
- Обработка webhook событий **идемпотентна** по `event.id`.
- Entitlements (доступ к premium-фичам) выводятся из статуса подписки.

## Consequences
### Positive
- корректная модель оплаты
- возможность расследования через аудит событий

### Negative
- требуется публичный endpoint webhook и инфраструктура для приёма событий
- нужно учитывать задержки доставки webhooks

## Alternatives considered
- доверять client-side успеху checkout: небезопасно и приводит к фроду.
