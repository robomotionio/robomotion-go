package runtime

import (
	"sync"

	"bitbucket.org/mosteknoloji/robomotion-go-lib/message"
)

var (
	handlers = make(map[string]MessageHandler)
	hMux     sync.Mutex
)

type MessageHandler interface {
	OnCreate() error
	OnMessage(ctx message.Context) error
	OnClose() error
}

func AddMessageHandler(guid string, handler MessageHandler) {
	hMux.Lock()
	defer hMux.Unlock()
	handlers[guid] = handler
}

func GetMessageHandler(guid string) MessageHandler {
	hMux.Lock()
	defer hMux.Unlock()
	h, _ := handlers[guid]
	return h
}

func RegisterMessageHandlers(nodes ...MessageHandler) {
}
