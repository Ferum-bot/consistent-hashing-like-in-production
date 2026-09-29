package proxy

import "time"

type UpstreamObserver interface {
	ObserveUpstream(node string, code int, duration time.Duration)
}
