package at

type Notifier interface {
	Notify(message string) error
}
