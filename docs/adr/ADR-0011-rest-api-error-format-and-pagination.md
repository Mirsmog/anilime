# ADR-0011: REST API error format and pagination

- **Status**: Accepted
- **Date**: 2026-02-02

## Context
У проекта будет множество REST эндпоинтов (BFF + сервисы). Для клиентов и для поддержки важно:
- единый предсказуемый формат ошибок
- корректные HTTP статусы
- стабильная пагинация для больших списков

Без стандартов API быстро деградирует: разные сервисы возвращают разные поля, клиенты начинают хардкодить кейсы.

## Decision

### Error response format
Все ошибки API возвращаются JSON-объектом:

```json
{
  "error": {
    "code": "AUTH_INVALID_CREDENTIALS",
    "message": "Invalid email or password",
    "details": {
      "field": "password"
    },
    "request_id": "01H..."
  }
}
```

Правила:
- `code` — стабильный машинный код (UPPER_SNAKE_CASE), не меняется без major версии API.
- `message` — короткое человекочитаемое описание (может локализоваться на уровне BFF).
- `details` — опциональный объект для структурных деталей (валидация, лимиты и т.п.).
- `request_id` — корреляционный идентификатор для логов/трейсов.

Маппинг классов ошибок на HTTP:
- `400` — invalid argument/validation
- `401` — unauthenticated
- `403` — permission denied / subscription required
- `404` — not found
- `409` — conflict (например, уже существует)
- `429` — rate limited
- `5xx` — internal / dependency failure

### Pagination strategy
Для списков используем **cursor/keyset pagination**, чтобы избежать проблем `OFFSET/LIMIT` на больших объёмах.

Запрос:
- `GET /v1/anime?limit=50&cursor=...`

Ответ:

```json
{
  "items": [ ... ],
  "page": {
    "limit": 50,
    "next_cursor": "...",
    "prev_cursor": null
  }
}
```

Правила:
- `limit` ограничивается сервером (например 1..100).
- `cursor` — opaque строка, которую клиент не парсит.
- Сортировка должна быть детерминированной (например `(created_at, id)` или `(score, id)`), чтобы cursor работал.

Где cursor-pagination слишком сложна (например admin-listing), допускается `offset` пагинация, но это должно быть явно оговорено в документации эндпоинта.

## Consequences
### Positive
- единый developer experience для клиентов
- проще поддержка и логирование инцидентов
- масштабируемые листинги без деградации от `OFFSET`

### Negative
- cursor pagination сложнее в реализации и тестировании
- нужно аккуратно проектировать сортировки и индексы

## Alternatives considered
- Разрозненный формат ошибок: быстрее на старте, но ломает клиентов и поддержку.
- Только offset pagination: просто, но плохо масштабируется и даёт "дырки" при изменении данных между запросами.
