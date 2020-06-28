package runtime

import (
	"sync"

	"bitbucket.org/mosteknoloji/robomotion-go-lib/message"
)

var (
	handlers = make(map[string]NodeHandler)
	hMux     sync.Mutex
)

type NodeHandler struct {
	Node    Node
	Handler MessageHandler
}

type MessageHandler interface {
	OnCreate() error
	OnMessage(ctx message.Context) error
	OnClose() error
}

func AddMessageHandler(node Node, handler MessageHandler) {
	hMux.Lock()
	defer hMux.Unlock()
	handlers[node.GUID] = NodeHandler{
		Node:    node,
		Handler: handler,
	}
}

func GetMessageHandler(guid string) MessageHandler {
	hMux.Lock()
	defer hMux.Unlock()
	h, _ := handlers[guid]
	return h.Handler
}

func RegisterNodeHandlers(handlers ...NodeHandler) {
}
