# ADR-0004: Data ownership and source of truth

- **Status**: Accepted
- **Date**: 2026-02-02

## Context
Метаданные по аниме будут поступать из внешних источников (Jikan/MAL), а ссылки на просмотр — из другого провайдера/парсера.
Нельзя зависеть от внешних API на пути пользовательского запроса (latency, rate limit, availability).

## Decision
- **Catalog service** хранит локальную нормализованную копию метаданных (наша read model).
- Внешние API (Jikan и т.п.) используются только ingestion/worker слоем.
- Вводим внутренний `anime_id` как primary key и отдельные поля/таблицы для `mal_id` и других внешних идентификаторов.
- Streaming Resolver хранит свои сущности (sources/playback) отдельно от каталога.

## Consequences
### Positive
- стабильная скорость и доступность пользовательских запросов
- контролируемая схема данных и индексация
- можно менять поставщика данных без полной миграции

### Negative
- требуется ingestion pipeline, задачи обновления и мониторинг свежести данных

## Alternatives considered
- Дёргать Jikan на каждый запрос: не масштабируется, ломается при rate limit.
