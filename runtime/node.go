package runtime

type Node struct {
	Scope           string
	GUID            string
	Name            string
	DelayBefore     float32
	DelayAfter      float32
	ContinueOnError bool
}
