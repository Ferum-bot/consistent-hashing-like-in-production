# libs

Общий код сервисов wallet, router и loadgen. Отдельный Go-модуль, сервисы подключают его через `replace` в своём `go.mod`.

| Пакет | Что внутри |
| --- | --- |
| `sharding` | стратегии шардирования: `Mod` (`hash % N`) и `Ring` (кольцо с `vnodes × weight` точками на ноду), хеш xxHash64 |
| `service` | запуск сервиса: флаг `-config`, остановка по сигналу, метрики процесса, `/healthz`, `/metrics` и Swagger |
| `config` | чтение `config.yaml`; поле с тегом `env:"NAME"` перекрывается переменной окружения, неизвестное поле в файле — ошибка |
| `snapshot` | построчное чтение снапшота пользователей и `user_id` из строки |
| `wait` | пауза, которая прерывается при остановке сервиса |
| `metrics` | общие бакеты гистограмм задержек |
| `http/request` | разбор запроса: числа из пути и заголовков, JSON-тело без лишних полей |
| `http/response` | ответы JSON и ошибки в виде `{"error": "..."}` |
| `http/ndjson` | запись и чтение NDJSON |
| `http/headers` | общие заголовки: `X-Topology-Version`, `X-Routed-To`, `Retry-After` |
| `http/recorder` | обёртка ответа, которая запоминает код |
| `http/middlewares` | метрики HTTP-запросов по маршрутам и кодам |
| `http/server` | HTTP-сервер с корректной остановкой |
| `http/swagger` | Swagger UI и отдача `openapi.yaml` |

```bash
task libs:test
task libs:bench
```
