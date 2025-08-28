package notifications

import (
	"errors"
	"log"
)

type Notifier interface {
	Notify(message string) error
}

type StackNotifier struct {
	notifiers []Notifier
	Notifier
}

func NewStackNotifier() *StackNotifier {
	return &StackNotifier{
		notifiers: []Notifier{},
	}
}

func (s *StackNotifier) AddNotifier(n Notifier) {
	s.notifiers = append(s.notifiers, n)
}

func (s *StackNotifier) Notify(message string) error {
	log.Printf("notifier: received message \"%s\"", message)
	errSlice := []error{}
	for _, n := range s.notifiers {
		err := n.Notify(message)
		if err != nil {
			errSlice = append(errSlice, err)
		}
	}

	if len(errSlice) == 0 {
		return nil
	}
	return errors.Join(errSlice...)
}