package metrics

import (
	"sync"
)

type Counter interface {
	Inc()
	Add(float64)
}

type deferredCounter struct {
	sync.Mutex
	name, desc string
	tags       []string
	counter    Counter
}

func newDeferredCounter(name, desc string, tags []string) *deferredCounter {
	return &deferredCounter{name: name, desc: desc, tags: tags}
}

func (c *deferredCounter) Realize(m *Metrics) {
	c.Lock()
	defer c.Unlock()
	c.counter = m.RegisterCounter(c.name, c.desc, c.tags...)
}

func (c *deferredCounter) Inc() {
	c.Lock()
	defer c.Unlock()
	if c.counter != nil {
		c.counter.Inc()
	}
}

func (c *deferredCounter) Add(v float64) {
	c.Lock()
	defer c.Unlock()
	if c.counter != nil {
		c.counter.Add(v)
	}
}
