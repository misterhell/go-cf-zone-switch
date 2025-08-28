package notifications

type TelegramNotifier struct {
	Notifier
}

func (t *TelegramNotifier) Notify(message string) error {
	return nil
}
