package runtime

var (
	factories = make(map[string]INodeFactory)
	nodes     = make(map[string]Node)
	Plugin    Node
)

type SNode struct {
	GUID            string
	Name            string
	DelayBefore     float32
	DelayAfter      float32
	ContinueOnError bool
	Scope           string
}

func (n *SNode) Init(e RuntimeHelper) error {
	runtimeHelper = e
	return nil
}

func (n *SNode) OnCreate(string, []byte) error {
	return nil
}

func CreateNode(name string, factory INodeFactory) {
	factories[name] = factory
}

func Factories() map[string]INodeFactory {
	return factories
}

func AddNode(guid string, node Node) {
	nodes[guid] = node
}

func Nodes() map[string]Node {
	return nodes
}

func InitNodes(nodes ...Node) {
}
