package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type Sampler interface {
	Observe(float64)
}

type noopSampler struct{}

func (c noopSampler) Observe(_ float64) {}

type deferredSampler struct {
	sync.Mutex
	name, desc string
	tags       Tags
	sampler    Sampler
}

func newDeferredSampler(name, desc string, tags Tags) *deferredSampler {
	return &deferredSampler{name: name, desc: desc, tags: tags}
}

func (c *deferredSampler) Realize(m *Metrics) {
	c.Lock()
	defer c.Unlock()
	c.sampler = m.RegisterSampler(c.name, c.desc, c.tags)
}

func (c *deferredSampler) Observe(v float64) {
	c.Lock()
	defer c.Unlock()
	if c.sampler != nil {
		c.sampler.Observe(v)
	}
}

type SamplerVec interface {
	With(Tags) Sampler
}

type prometheusSamplerVec prometheus.SummaryVec

func (g *prometheusSamplerVec) With(t Tags) Sampler {
	return (*prometheus.SummaryVec)(g).With(prometheus.Labels(t))
}

type deferredSamplerVec struct {
	sync.Mutex
	name, desc string
	opts       []string
	sampler    SamplerVec
}

func newDeferredSamplerVec(name, desc string, opts []string) *deferredSamplerVec {
	return &deferredSamplerVec{name: name, desc: desc, opts: opts}
}

func (c *deferredSamplerVec) Realize(m *Metrics) {
	c.Lock()
	defer c.Unlock()
	c.sampler = m.RegisterSamplerVec(c.name, c.desc, c.opts)
}

func (c *deferredSamplerVec) With(t Tags) Sampler {
	c.Lock()
	defer c.Unlock()
	if c.sampler != nil {
		return c.sampler.With(t)
	} else {
		return noopSampler{}
	}
}
