package chat

import (
	"fmt"
	"time"
)

type Filter struct {
	UserName string
}

type Message struct {
	Content string
	Author  *User
	Time    time.Time

	filters []Filter
}

func NewMessage(content string, author *User, filters ...Filter) Message {
	return Message{content, author, time.Now(), filters}
}

func (m *Message) String() string {
	return fmt.Sprintf("%s - %s: %s", m.Time.Format(time.Stamp), m.Author.GetName(), m.Content)
}

// Checks whether the message should be sent to the given user by check all
// message and user filters.
func (m *Message) CanSend(u *User) bool {
	if m.Author.GetName() == u.GetName() {
		return false
	}

	for _, filter := range u.filters {
		if m.Author.GetName() == filter.UserName {
			return false
		}
	}

	return m.checkMessageFilters(u)
}

func (m *Message) checkMessageFilters(u *User) bool {
	for _, filter := range m.filters {
		if u.GetName() == filter.UserName {
			return false
		}
	}

	return true
}
