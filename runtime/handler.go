package runtime

import (
	"sync"

	"github.com/robomotionio/robomotion-go/message"
)

var (
	handlerList []MessageHandler
	handlers    = make(map[string]*NodeHandler)
	hMux        sync.Mutex
)

type NodeHandler struct {
	Node
	Handler MessageHandler
}

type MessageHandler interface {
	OnCreate() error
	OnMessage(ctx message.Context) error
	OnClose() error
}

func AddNodeHandler(node Node, handler MessageHandler) {
	hMux.Lock()
	defer hMux.Unlock()
	
	// Automatically wrap handler with tool interceptor if needed
	wrappedHandler := NewToolInterceptor(handler)
	
	handlers[node.GUID] = &NodeHandler{
		Handler: wrappedHandler,
		Node:    node,
	}
}

func GetNodeHandler(guid string) *NodeHandler {
	hMux.Lock()
	defer hMux.Unlock()
	h, _ := handlers[guid]
	return h
}

func RegisterNodes(handlers ...MessageHandler) {
	handlerList = handlers
}

func RegisteredNodes() []MessageHandler {
	return handlerList
}
