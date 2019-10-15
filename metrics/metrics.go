package metrics

import (
	"github.com/AlexAkulov/hungryfox/config"
	m "github.com/go-kit/kit/metrics"
	"github.com/rs/zerolog"
)

type IMetricsRepo interface {
	CreateCounter(string) m.Counter
	CreateGauge(string) m.Gauge
	CreateHistogram(string) m.Histogram
	Stop() error
}

func StartMetricsRepo(config *config.Metrics, log zerolog.Logger) IMetricsRepo {
	if config.GraphiteAddress != "" && config.Prefix != "" {
		return startGraphiteRepo(config.GraphiteAddress, config.Prefix, config.SendInterval, &log)
	}
	return &noopRepo{}
}
