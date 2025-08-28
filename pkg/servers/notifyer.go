package servers


type Notifier interface {
	Notify() error
}
