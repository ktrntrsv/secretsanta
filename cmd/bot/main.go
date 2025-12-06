package main

import (
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"secret-santa/internal/bot"
	"secret-santa/internal/game"
	"secret-santa/internal/storage"
)

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN не задан")
	}

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("не удалось создать бота: %v", err)
	}

	dataDir := "data"
	store, err := storage.NewStore(dataDir)
	if err != nil {
		log.Fatalf("не удалось создать хранилище: %v", err)
	}

	gameState, err := game.NewGame(store)
	if err != nil {
		log.Fatalf("не удалось загрузить состояние: %v", err)
	}

	b := bot.New(api, gameState, "ktrntrsv")

	_, err = api.Request(tgbotapi.NewSetMyCommands(
		tgbotapi.BotCommand{Command: "join", Description: "Вступить в игру"},
		tgbotapi.BotCommand{Command: "leave", Description: "Выйти (пока не начали)"},
		tgbotapi.BotCommand{Command: "wishlist", Description: "Добавить вишлист"},
		tgbotapi.BotCommand{Command: "editwishlist", Description: "Обновить вишлист"},
		tgbotapi.BotCommand{Command: "mywishlist", Description: "Посмотреть свой вишлист"},
		tgbotapi.BotCommand{Command: "mygiftee", Description: "Узнать, кому даришь"},
		tgbotapi.BotCommand{Command: "thankyou", Description: "Сказать спасибо своему Санте"},
		tgbotapi.BotCommand{Command: "startgame", Description: "Запустить игру (только админ)"},
	))
	if err != nil {
		log.Printf("не удалось установить команды: %v", err)
	}

	log.Printf("Бот запущен как %s", api.Self.UserName)
	if err := b.Run(); err != nil {
		log.Fatalf("бот остановился: %v", err)
	}
}
