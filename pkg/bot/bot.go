package bot

import (
	"log"
)

// Telegram bot integration scaffold.
// HARD CONSTRAINT / PRIVACY NOTICE:
// Disabled by default. If enabled, images transit Telegram's cloud servers,
// weakening Munim's 100% local privacy guarantee.
type Bot struct {
	token   string
	enabled bool
}

func NewBot(token string, enabled bool) *Bot {
	return &Bot{
		token:   token,
		enabled: enabled,
	}
}

func (b *Bot) Start() {
	if !b.enabled {
		log.Println("[Telegram Bot] Disabled by default to protect local bill privacy. Enable via -telegram-token if desired.")
		return
	}
	log.Println("[WARNING] Telegram bot active: Note that bill photos will transit Telegram servers.")
}
