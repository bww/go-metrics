package metrics

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var errReinitialized = errors.New("Initialized more than once")

type Tags map[string]string

var defaultQuantiles = map[float64]float64{
	0.5:  0.05,
	0.9:  0.01,
	0.99: 0.001,
}

type Config struct {
	Addr       string
	Namespace  string
	System     string
	Registerer prometheus.Registerer
	Gatherer   prometheus.Gatherer
}

func (c Config) WithRegistry(reg *prometheus.Registry) Config {
	c.Registerer = reg
	c.Gatherer = reg
	return c
}

type Deferred interface {
	Realize(m *Metrics)
}

type Metrics struct {
	server     *http.Server
	registerer prometheus.Registerer
	gatherer   prometheus.Gatherer
	namespace  string
	system     string
}

func New(conf Config) (*Metrics, error) {
	r := conf.Registerer
	if r == nil {
		r = prometheus.DefaultRegisterer
	}
	g := conf.Gatherer
	if g == nil {
		g = prometheus.DefaultGatherer
	}
	s := &http.Server{
		Addr:           conf.Addr,
		Handler:        promhttp.InstrumentMetricHandler(r, promhttp.HandlerFor(g, promhttp.HandlerOpts{})),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	return &Metrics{
		server:     s,
		registerer: r,
		gatherer:   g,
		namespace:  conf.Namespace,
		system:     conf.System,
	}, nil
}

func (m *Metrics) Run() {
	go func() {
		panic(m.server.ListenAndServe())
	}()
}

func (m *Metrics) RegisterCounter(name, desc string, tags Tags) Counter {
	v := prometheus.NewCounter(prometheus.CounterOpts{
		Namespace:   m.namespace,
		Subsystem:   m.system,
		Name:        name,
		Help:        desc,
		ConstLabels: prometheus.Labels(tags),
	})
	err := m.registerer.Register(v)
	if err != nil {
		panic(fmt.Errorf("Could not register metric: %q: %w", name, err))
	}
	return v
}

func (m *Metrics) RegisterCounterVec(name, desc string, opts []string) CounterVec {
	v := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: m.namespace,
		Subsystem: m.system,
		Name:      name,
		Help:      desc,
	}, opts)
	err := m.registerer.Register(v)
	if err != nil {
		panic(fmt.Errorf("Could not register metric: %q: %w", name, err))
	}
	p := prometheusCounterVec(*v)
	return &p
}

func (m *Metrics) RegisterGauge(name, desc string, tags Tags) Gauge {
	v := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   m.namespace,
		Subsystem:   m.system,
		Name:        name,
		Help:        desc,
		ConstLabels: prometheus.Labels(tags),
	})
	err := m.registerer.Register(v)
	if err != nil {
		panic(fmt.Errorf("Could not register metric: %q: %w", name, err))
	}
	return v
}

func (m *Metrics) RegisterGaugeVec(name, desc string, opts []string) GaugeVec {
	v := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: m.namespace,
		Subsystem: m.system,
		Name:      name,
		Help:      desc,
	}, opts)
	err := m.registerer.Register(v)
	if err != nil {
		panic(fmt.Errorf("Could not register metric: %q: %w", name, err))
	}
	p := prometheusGaugeVec(*v)
	return &p
}

func (m *Metrics) RegisterSampler(name, desc string, tags Tags) Sampler {
	v := prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace:   m.namespace,
		Subsystem:   m.system,
		Name:        name,
		Help:        desc,
		ConstLabels: prometheus.Labels(tags),
		Objectives:  defaultQuantiles,
	})
	err := m.registerer.Register(v)
	if err != nil {
		panic(fmt.Errorf("Could not register metric: %q: %w", name, err))
	}
	return v
}

func (m *Metrics) RegisterSamplerVec(name, desc string, opts []string) SamplerVec {
	v := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace:  m.namespace,
		Subsystem:  m.system,
		Name:       name,
		Help:       desc,
		Objectives: defaultQuantiles,
	}, opts)
	err := m.registerer.Register(v)
	if err != nil {
		panic(fmt.Errorf("Could not register metric: %q: %w", name, err))
	}
	p := prometheusSamplerVec(*v)
	return &p
}

var (
	lock    sync.Mutex
	pending []Deferred
	shared  *Metrics
)

func Init(conf Config) (*Metrics, error) {
	lock.Lock()
	defer lock.Unlock()

	var err error
	if shared != nil {
		return shared, nil
	}

	shared, err = New(conf)
	if err != nil {
		return nil, err
	}
	for _, e := range pending {
		e.Realize(shared)
	}

	shared.Run()
	return shared, nil
}

func RegisterCounter(name, desc string, tags Tags) Counter {
	lock.Lock()
	defer lock.Unlock()
	if shared != nil {
		return shared.RegisterCounter(name, desc, tags)
	} else {
		d := newDeferredCounter(name, desc, tags)
		pending = append(pending, d)
		return d
	}
}

func RegisterCounterVec(name, desc string, opts []string) CounterVec {
	lock.Lock()
	defer lock.Unlock()
	if shared != nil {
		return shared.RegisterCounterVec(name, desc, opts)
	} else {
		d := newDeferredCounterVec(name, desc, opts)
		pending = append(pending, d)
		return d
	}
}

func RegisterGauge(name, desc string, tags Tags) Gauge {
	lock.Lock()
	defer lock.Unlock()
	if shared != nil {
		return shared.RegisterGauge(name, desc, tags)
	} else {
		d := newDeferredGauge(name, desc, tags)
		pending = append(pending, d)
		return d
	}
}

func RegisterGaugeVec(name, desc string, opts []string) GaugeVec {
	lock.Lock()
	defer lock.Unlock()
	if shared != nil {
		return shared.RegisterGaugeVec(name, desc, opts)
	} else {
		d := newDeferredGaugeVec(name, desc, opts)
		pending = append(pending, d)
		return d
	}
}

func RegisterSampler(name, desc string, tags Tags) Sampler {
	lock.Lock()
	defer lock.Unlock()
	if shared != nil {
		return shared.RegisterSampler(name, desc, tags)
	} else {
		d := newDeferredSampler(name, desc, tags)
		pending = append(pending, d)
		return d
	}
}

func RegisterSamplerVec(name, desc string, opts []string) SamplerVec {
	lock.Lock()
	defer lock.Unlock()
	if shared != nil {
		return shared.RegisterSamplerVec(name, desc, opts)
	} else {
		d := newDeferredSamplerVec(name, desc, opts)
		pending = append(pending, d)
		return d
	}
}
