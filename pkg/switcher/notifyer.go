package switcher


type Notifier interface {
	Notify(message string) error
}
