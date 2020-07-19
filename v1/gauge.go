package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type Gauge interface {
	Inc()
	Dec()
	Set(float64)
	Add(float64)
	Sub(float64)
}

type noopGauge struct{}

func (g noopGauge) Inc()          {}
func (g noopGauge) Dec()          {}
func (g noopGauge) Set(_ float64) {}
func (g noopGauge) Add(_ float64) {}
func (g noopGauge) Sub(_ float64) {}

type deferredGauge struct {
	sync.Mutex
	name, desc string
	tags       Tags
	gauge      Gauge
}

func newDeferredGauge(name, desc string, tags Tags) *deferredGauge {
	return &deferredGauge{name: name, desc: desc, tags: tags}
}

func (c *deferredGauge) Realize(m *Metrics) {
	c.Lock()
	defer c.Unlock()
	c.gauge = m.RegisterGauge(c.name, c.desc, c.tags)
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

type GaugeVec interface {
	With(Tags) Gauge
}

type prometheusGaugeVec prometheus.GaugeVec

func (g *prometheusGaugeVec) With(t Tags) Gauge {
	return (*prometheus.GaugeVec)(g).With(prometheus.Labels(t))
}

type deferredGaugeVec struct {
	sync.Mutex
	name, desc string
	opts       []string
	gauge      GaugeVec
}

func newDeferredGaugeVec(name, desc string, opts []string) *deferredGaugeVec {
	return &deferredGaugeVec{name: name, desc: desc, opts: opts}
}

func (c *deferredGaugeVec) Realize(m *Metrics) {
	c.Lock()
	defer c.Unlock()
	c.gauge = m.RegisterGaugeVec(c.name, c.desc, c.opts...)
}

func (c *deferredGaugeVec) With(t Tags) Gauge {
	c.Lock()
	defer c.Unlock()
	if c.gauge != nil {
		return c.gauge.With(t)
	} else {
		return noopGauge{}
	}
}
