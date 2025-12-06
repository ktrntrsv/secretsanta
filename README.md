# Secret Santa Telegram Bot

Весёлый бот для одной команды Тайного Санты. Все сообщения — на русском, тёплые и озорные.

## Как запустить локально

1. Создай в переменной окружения токен бота:
   ```bash
   export TELEGRAM_BOT_TOKEN=твой_токен
   ```
2. Запусти:
   ```bash
   go run ./cmd/bot
   ```

Данные хранятся в `data/` (участники, вишлисты, назначения). Бот создаёт папку сам.

## Docker

Собрать и запустить:
```bash
docker build -t secret-santa-bot .
docker run -e TELEGRAM_BOT_TOKEN=$TELEGRAM_BOT_TOKEN -v $(pwd)/data:/app/data secret-santa-bot
```

## Docker Compose

```bash
TELEGRAM_BOT_TOKEN=твой_токен docker-compose up --build
```

## Команды бота

- `/join` — вступить в игру
- `/leave` — выйти до старта
- `/wishlist <текст>` — создать вишлист
- `/editwishlist <текст>` — обновить вишлист
- `/mywishlist` — показать свой вишлист
- `/mygiftee` — узнать, кому даришь (после старта)
- `/thankyou <текст>` — отправить спасибо своему Санте
- `/startgame` — запускает только админ `@ktrntrsv`

## Структура проекта

- `cmd/bot/main.go` — входная точка
- `internal/bot` — обработчики Telegram-команд и ответы
- `internal/game` — логика игры, рандомные пары, проверки
- `internal/storage` — JSON-персистенс, потокобезопасная запись
- `internal/models` — структуры данных
- `data/` — сохранённые файлы (JSON)

## Памятка

- Токен берётся из переменной `TELEGRAM_BOT_TOKEN`.
- Админ единственный — `@ktrntrsv`.
- При старте игры список участников блокируется, всем рассылаются их подопечные и вишлисты.
