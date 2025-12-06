package bot

import (
	"errors"
	"fmt"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"secret-santa/internal/game"
	"secret-santa/internal/models"
)

// Bot wires Telegram updates to game logic.
type Bot struct {
	api           *tgbotapi.BotAPI
	game          *game.Game
	adminUsername string
}

// New constructs a Bot instance.
func New(api *tgbotapi.BotAPI, g *game.Game, adminUsername string) *Bot {
	return &Bot{
		api:           api,
		game:          g,
		adminUsername: strings.TrimPrefix(adminUsername, "@"),
	}
}

// Run starts receiving updates and handling commands.
func (b *Bot) Run() error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		if update.Message.IsCommand() {
			b.handleCommand(update.Message)
		}
	}

	return nil
}

func (b *Bot) handleCommand(msg *tgbotapi.Message) {
	user := msg.From
	if user == nil {
		return
	}

	command := msg.Command()
	args := strings.TrimSpace(msg.CommandArguments())

	switch command {
	case "join":
		b.handleJoin(msg)
	case "leave":
		b.handleLeave(msg)
	case "wishlist":
		b.handleWishlist(msg, args)
	case "editwishlist":
		b.handleWishlist(msg, args)
	case "mywishlist":
		b.handleMyWishlist(msg)
	case "mygiftee":
		b.handleMyGiftee(msg)
	case "thankyou":
		b.handleThankYou(msg, args)
	case "startgame":
		b.handleStartGame(msg)
	case "help", "start":
		b.handleHelp(msg)
	default:
		b.reply(msg.Chat.ID, msg.MessageID, "Не знаю такую команду, но я за любую движуху! Попробуй /help 😜")
	}
}

func (b *Bot) handleHelp(msg *tgbotapi.Message) {
	text := "Привет, Synapse-команда! Я ваш Тайный Санта-бот, заряженный весельем. Вот что умею:\n" +
		"/join — влетаешь в игру\n" +
		"/leave — тихо уходишь (пока не начали)\n" +
		"/wishlist текст — делишься желаниями\n" +
		"/editwishlist текст — обновляешь хотелки\n" +
		"/mywishlist — показываю твой список\n" +
		"/mygiftee — расскажу, кому даришь (после старта)\n" +
		"/thankyou текст — отправлю спасибо твоему Санте\n" +
		fmt.Sprintf("/startgame — запускает только @%s\n", b.adminUsername) +
		"Ждём тебя в нашей синэпсовой новогодней игре! 🎄"
	b.reply(msg.Chat.ID, msg.MessageID, text)
}

func (b *Bot) handleJoin(msg *tgbotapi.Message) {
	p := models.Participant{
		UserID:    msg.From.ID,
		Username:  msg.From.UserName,
		FirstName: msg.From.FirstName,
	}

	err := b.game.Join(p)
	switch err {
	case nil:
		b.reply(msg.Chat.ID, msg.MessageID, "Ура! Ты в игре 🎄✨ Добро пожаловать в синэпсовую тусовку тайных волшебников!")
	case game.ErrAlreadyJoined:
		b.reply(msg.Chat.ID, msg.MessageID, "Ты и так уже участвуешь — расслабься, Synapse тебя помнит 😎")
	case game.ErrGameStarted:
		b.reply(msg.Chat.ID, msg.MessageID, "Опоздал чутка: игра уже стартанула, набор закрыт 😜")
	default:
		b.reply(msg.Chat.ID, msg.MessageID, "Ой, что-то пошло не так. Позови хэндмейда 🤖")
	}
}

func (b *Bot) handleLeave(msg *tgbotapi.Message) {
	err := b.game.Leave(msg.From.ID)
	switch err {
	case nil:
		b.reply(msg.Chat.ID, msg.MessageID, "Окей, удаляю… но мы будем скучать всей Synapse-бандой 🥺")
	case game.ErrNotParticipant:
		b.reply(msg.Chat.ID, msg.MessageID, "Тебя и так нет в списке, присоединяйся через /join, если передумаешь 😉")
	case game.ErrGameStarted:
		b.reply(msg.Chat.ID, msg.MessageID, "Уже поздно сбежать — мешок с подарками запечатан 🎁🔒")
	default:
		b.reply(msg.Chat.ID, msg.MessageID, "Не могу удалить, магия барахлит 🤔")
	}
}

func (b *Bot) handleWishlist(msg *tgbotapi.Message, text string) {
	if strings.TrimSpace(text) == "" {
		b.reply(msg.Chat.ID, msg.MessageID, "Напиши свои хотелки после команды, например: /wishlist носки с оленями 🧦")
		return
	}

	if !b.game.IsParticipant(msg.From.ID) {
		b.reply(msg.Chat.ID, msg.MessageID, "Сначала залетай в игру через /join, а потом делись мечтами, синэпсовый волшебник 🎁")
		return
	}

	santaID, err := b.game.UpdateWishlist(msg.From.ID, text)
	if err != nil {
		b.reply(msg.Chat.ID, msg.MessageID, "Не смог сохранить вишлист, что-то сломалось 😢")
		return
	}

	if b.game.IsStarted() {
		b.reply(msg.Chat.ID, msg.MessageID, "Твой вишлист обновлён! Твой Санта теперь в курсе и уже потирает ручки 🎁🧑‍🎄")
		if santaID != 0 {
			b.notifySantaAboutWishlist(santaID, msg.From, text)
		}
		return
	}

	b.reply(msg.Chat.ID, msg.MessageID, "Записал твой вишлист! Пусть синэпсовые мечты сбудутся ✨")
}

func (b *Bot) handleMyWishlist(msg *tgbotapi.Message) {
	text, ok := b.game.GetWishlist(msg.From.ID)
	if !ok || strings.TrimSpace(text) == "" {
		b.reply(msg.Chat.ID, msg.MessageID, "У тебя пока пусто. Добавь желания через /wishlist 🎁")
		return
	}
	b.reply(msg.Chat.ID, msg.MessageID, fmt.Sprintf("Твой вишлист:\n%s", text))
}

func (b *Bot) handleMyGiftee(msg *tgbotapi.Message) {
	giftee, err := b.game.GetGiftee(msg.From.ID)
	if err != nil {
		if errors.Is(err, game.ErrGameNotStarted) {
			b.reply(msg.Chat.ID, msg.MessageID, "Игра ещё не началась, Synapse ждёт магического старта 🧙‍♂️")
			return
		}
		b.reply(msg.Chat.ID, msg.MessageID, "Не могу найти твою цель, похоже, гремлины шалят 🤷‍♂️")
		return
	}

	name := displayName(giftee)
	wishlist, ok := b.game.GetWishlist(giftee.UserID)
	if !ok || strings.TrimSpace(wishlist) == "" {
		b.reply(msg.Chat.ID, msg.MessageID, fmt.Sprintf("Ты даришь подарки для %s. Пока без вишлиста, пора подключить фантазию 🎁", name))
		return
	}

	b.reply(msg.Chat.ID, msg.MessageID, fmt.Sprintf("Ты даришь подарки для %s. Его вишлист:\n%s", name, wishlist))
}

func (b *Bot) handleThankYou(msg *tgbotapi.Message, text string) {
	if !b.game.IsStarted() {
		b.reply(msg.Chat.ID, msg.MessageID, "Сначала сыграем, потом благодарим, синэпсовый волшебник 😉")
		return
	}

	if strings.TrimSpace(text) == "" {
		b.reply(msg.Chat.ID, msg.MessageID, "Напиши, за что хочешь поблагодарить, в формате '/thankyou спасибо за свитер с оленями, он мне очень идет!' и я передам дальше 🙏")
		return
	}

	santaID, err := b.game.GetSanta(msg.From.ID)
	if err != nil || santaID == 0 {
		b.reply(msg.Chat.ID, msg.MessageID, "Не могу найти твоего Санту, что-то пошло не так 🤔")
		return
	}

	from := displayName(models.Participant{
		Username:  msg.From.UserName,
		FirstName: msg.From.FirstName,
	})
	payload := fmt.Sprintf("Тебе привет от твоего подопечного %s:\n\"%s\"", from, text)

	if _, err := b.api.Send(tgbotapi.NewMessage(santaID, payload)); err != nil {
		log.Printf("cannot forward thankyou: %v", err)
		b.reply(msg.Chat.ID, msg.MessageID, "Не смог доставить благодарность, попробуй позже 🙈")
		return
	}
	b.reply(msg.Chat.ID, msg.MessageID, "Передал! Твой Санта теперь улыбается до ушей 😁")
}

func (b *Bot) handleStartGame(msg *tgbotapi.Message) {
	if strings.TrimPrefix(msg.From.UserName, "@") != b.adminUsername {
		b.reply(msg.Chat.ID, msg.MessageID, fmt.Sprintf("Только @%s может жать красную кнопку 🚦", b.adminUsername))
		return
	}

	assignments, err := b.game.Start()
	switch err {
	case nil:
		b.reply(msg.Chat.ID, msg.MessageID, "Поехали, Synapse! Рассылаю, кто кому дарит 🎄")
	case game.ErrTooFewPlayers:
		b.reply(msg.Chat.ID, msg.MessageID, "Нужно хотя бы двое игроков, чтобы обменяться магией 🎅🤶")
		return
	case game.ErrAlreadyStarted:
		b.reply(msg.Chat.ID, msg.MessageID, "Уже запустились, назад пути нет ✨")
		return
	default:
		b.reply(msg.Chat.ID, msg.MessageID, "Не получилось запустить игру, магические шестерёнки застряли 😢")
		return
	}

	for santaID, gifteeID := range assignments {
		giftee := b.game.Participants()[gifteeID]
		name := displayName(giftee)
		wishlist, ok := b.game.GetWishlist(gifteeID)
		text := fmt.Sprintf("Ты тайный Санта для %s!\n", name)
		if ok && strings.TrimSpace(wishlist) != "" {
			text += fmt.Sprintf("Вот его вишлист:\n%s\n", wishlist)
		} else {
			text += "Вишлиста нет, пора проявить фантазию! 🧠🎁\n"
		}
		text += "Тсс, это секрет! 🤫"

		if _, err := b.api.Send(tgbotapi.NewMessage(santaID, text)); err != nil {
			log.Printf("cannot notify santa %d: %v", santaID, err)
		}
	}
}

func (b *Bot) notifySantaAboutWishlist(santaID int64, from *tgbotapi.User, wishlist string) {
	name := from.UserName
	if name != "" {
		name = "@" + name
	} else {
		name = from.FirstName
	}

	text := fmt.Sprintf("Эй! Твой подопечный %s обновил вишлист:\n%s", name, wishlist)
	if _, err := b.api.Send(tgbotapi.NewMessage(santaID, text)); err != nil {
		log.Printf("cannot notify santa: %v", err)
	}
}

func (b *Bot) reply(chatID int64, replyTo int, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyToMessageID = replyTo
	b.api.Send(msg) //nolint:errcheck
}

func displayName(p models.Participant) string {
	if p.Username != "" {
		return "@" + p.Username
	}
	if p.FirstName != "" {
		return p.FirstName
	}
	return "таинственный герой"
}
