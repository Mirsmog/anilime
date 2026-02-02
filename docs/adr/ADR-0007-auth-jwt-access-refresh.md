# ADR-0007: Auth uses JWT access + refresh rotation

- **Status**: Accepted
- **Date**: 2026-02-02

## Context
Нужна авторизация для пользовательских действий (progress, коллекции, отзывы) и для контроля подписки.
Важно:
- низкая латентность проверки
- возможность отзывать/обновлять сессии
- защита от кражи refresh token

## Decision
- Используем **короткоживущий access token (JWT)**.
- Используем **refresh token** с ротацией (refresh rotation) и хранением состояния в БД.
- Ограничиваем brute force на login (rate limit).
- Пароли хешируем (bcrypt/argon2id — будет уточнено реализацией).

## Consequences
### Positive
- быстрое чтение access token без обращения к auth на каждый запрос
- refresh rotation уменьшает риск долгой компрометации

### Negative
- нужны механизмы отзывов/blacklist или короткий TTL access
- усложнение реализации (по сравнению с "просто JWT")

## Alternatives considered
- Только JWT без refresh state: сложно отзывать и управлять сессиями.
- Opaque tokens + introspection: более централизовано, но повышает нагрузку на auth.
