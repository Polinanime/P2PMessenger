package models

import (
	"sync"
	"time"
)

type ChatMessage struct {
	From      string
	To        string
	Content   string
	Timestamp time.Time
}

type MessageHistory struct {
	messages []ChatMessage
	mutex    sync.RWMutex
}

func NewMessageHistory() *MessageHistory {
	return &MessageHistory{
		messages: make([]ChatMessage, 0),
	}
}

func (mh *MessageHistory) AddMessage(msg ChatMessage) {
	mh.mutex.Lock()
	defer mh.mutex.Unlock()

	mh.messages = append(mh.messages, msg)
}

func (mh *MessageHistory) GetMessages() []ChatMessage {
	mh.mutex.RLock()
	defer mh.mutex.RUnlock()

	return mh.messages
}
