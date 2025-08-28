package notifications

import "go-cf-zone-switch/pkg/config"

type TelegramNotifier struct {
	Notifier
}

func NewTelegramNotifier(config *config.Config) *TelegramNotifier {
	_ = config
	return &TelegramNotifier{}
}

func (t *TelegramNotifier) Notify(message string) error {
	return nil
}
