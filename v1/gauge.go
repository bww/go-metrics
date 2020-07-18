package metrics

import (
	"sync"
)

type Gauge interface {
	Inc()
	Dec()
	Set(float64)
	Add(float64)
	Sub(float64)
}

type deferredGauge struct {
	sync.Mutex
	name, desc string
	tags       []string
	gauge      Gauge
}

func newDeferredGauge(name, desc string, tags []string) *deferredGauge {
	return &deferredGauge{name: name, desc: desc, tags: tags}
}

func (c *deferredGauge) Realize(m *Metrics) {
	c.Lock()
	defer c.Unlock()
	c.gauge = m.RegisterGauge(c.name, c.desc, c.tags...)
}

func (c *deferredGauge) Inc() {
	c.Lock()
	defer c.Unlock()
	if c.gauge != nil {
		c.gauge.Inc()
	}
}

func (c *deferredGauge) Dec() {
	c.Lock()
	defer c.Unlock()
	if c.gauge != nil {
		c.gauge.Dec()
	}
}

func (c *deferredGauge) Set(v float64) {
	c.Lock()
	defer c.Unlock()
	if c.gauge != nil {
		c.gauge.Set(v)
	}
}

func (c *deferredGauge) Sub(v float64) {
	c.Lock()
	defer c.Unlock()
	if c.gauge != nil {
		c.gauge.Sub(v)
	}
}

func (c *deferredGauge) Add(v float64) {
	c.Lock()
	defer c.Unlock()
	if c.gauge != nil {
		c.gauge.Add(v)
	}
}
