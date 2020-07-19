package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type Counter interface {
	Inc()
	Add(float64)
}

type noopCounter struct{}

func (c noopCounter) Inc()          {}
func (c noopCounter) Add(_ float64) {}

type deferredCounter struct {
	sync.Mutex
	name, desc string
	tags       Tags
	counter    Counter
}

func newDeferredCounter(name, desc string, tags Tags) *deferredCounter {
	return &deferredCounter{name: name, desc: desc, tags: tags}
}

func (c *deferredCounter) Realize(m *Metrics) {
	c.Lock()
	defer c.Unlock()
	c.counter = m.RegisterCounter(c.name, c.desc, c.tags)
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

type CounterVec interface {
	With(Tags) Counter
}

type prometheusCounterVec prometheus.CounterVec

func (g *prometheusCounterVec) With(t Tags) Counter {
	return (*prometheus.CounterVec)(g).With(prometheus.Labels(t))
}

type deferredCounterVec struct {
	sync.Mutex
	name, desc string
	opts       []string
	counter    CounterVec
}

func newDeferredCounterVec(name, desc string, opts []string) *deferredCounterVec {
	return &deferredCounterVec{name: name, desc: desc, opts: opts}
}

func (c *deferredCounterVec) Realize(m *Metrics) {
	c.Lock()
	defer c.Unlock()
	c.counter = m.RegisterCounterVec(c.name, c.desc, c.opts)
}

func (c *deferredCounterVec) With(t Tags) Counter {
	c.Lock()
	defer c.Unlock()
	if c.counter != nil {
		return c.counter.With(t)
	} else {
		return noopCounter{}
	}
}
