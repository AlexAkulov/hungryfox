package metrics

import (
	m "github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/discard"
)

type noopRepo struct {
}

func (r *noopRepo) CreateCounter(name string) m.Counter {
	return discard.NewCounter()
}

func (r *noopRepo) CreateGauge(name string) m.Gauge {
	return discard.NewGauge()
}

func (r *noopRepo) CreateHistogram(name string) m.Histogram {
	return discard.NewHistogram()
}

func (r *noopRepo) Stop() error {
	return nil
}
