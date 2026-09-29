package middlewares

type TopologyFence interface {
	Observe(version int64) (int64, error)
	Current() int64
	Strict() bool
}
