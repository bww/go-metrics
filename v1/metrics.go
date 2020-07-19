package metrics

import (
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var errReinitialized = errors.New("Initialized more than once")

type Tags map[string]string

type Config struct {
	Addr      string
	Namespace string
	System    string
}

type Deferred interface {
	Realize(m *Metrics)
}

type Metrics struct {
	server    *http.Server
	namespace string
	system    string
}

func New(conf Config) (*Metrics, error) {
	s := &http.Server{
		Addr:           conf.Addr,
		Handler:        promhttp.Handler(),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	return &Metrics{
		s,
		conf.Namespace,
		conf.System,
	}, nil
}

func (m *Metrics) Run() {
	go func() {
		log.Fatal(m.server.ListenAndServe())
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
	prometheus.MustRegister(v)
	return v
}

func (m *Metrics) RegisterCounterVec(name, desc string, opts []string) CounterVec {
	v := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: m.namespace,
		Subsystem: m.system,
		Name:      name,
		Help:      desc,
	}, opts)
	prometheus.MustRegister(v)
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
	prometheus.MustRegister(v)
	return v
}

func (m *Metrics) RegisterGaugeVec(name, desc string, opts []string) GaugeVec {
	v := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: m.namespace,
		Subsystem: m.system,
		Name:      name,
		Help:      desc,
	}, opts)
	prometheus.MustRegister(v)
	p := prometheusGaugeVec(*v)
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
