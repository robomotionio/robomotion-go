package runtime

type INode interface {
	Init(RuntimeHelper) error
}

type Node struct {
	Scope           string
	GUID            string
	Name            string
	DelayBefore     float32
	DelayAfter      float32
	ContinueOnError bool
}

func (n *Node) Init(e RuntimeHelper) error {
	client = e
	return nil
}
