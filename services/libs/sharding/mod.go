package sharding

type Mod struct {
	nodes []Node
}

func NewMod(nodes []Node) *Mod {
	return &Mod{nodes: nodes}
}

func (m *Mod) Locate(key string) Node {
	return m.nodes[Hash(key)%uint64(len(m.nodes))]
}
