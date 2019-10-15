package metrics

import (
	"context"
	"strings"
	"time"

	"github.com/AlexAkulov/hungryfox/helpers"
	"github.com/go-kit/kit/log"
	m "github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/graphite"
	"github.com/rs/zerolog"
)

type graphiteRepo struct {
	sendTicker  *time.Ticker
	graphite    *graphite.Graphite
	endSendLoop func()
}

func startGraphiteRepo(address, prefix string, sendInterval time.Duration, logger *zerolog.Logger) IMetricsRepo {
	graph := graphite.New(preparePrefix(prefix), makeLog(logger))
	sendTicker := time.NewTicker(sendInterval)
	ctx, endSend := context.WithCancel(context.Background())
	go graph.SendLoop(ctx, sendTicker.C, "tcp", address)

	return &graphiteRepo{
		sendTicker:  sendTicker,
		graphite:    graph,
		endSendLoop: endSend,
	}
}

func makeLog(logger *zerolog.Logger) log.Logger {
	if logger == nil {
		return log.NewNopLogger()
	}
	return helpers.WrapDebug(*logger)
}

func preparePrefix(prefix string) string {
	prefix = strings.TrimSuffix(prefix, ".")
	return prefix + "."
}

func (r *graphiteRepo) CreateCounter(name string) m.Counter {
	return r.graphite.NewCounter(name)
}

func (r *graphiteRepo) CreateGauge(name string) m.Gauge {
	return r.graphite.NewGauge(name)
}

func (r *graphiteRepo) CreateHistogram(name string) m.Histogram {
	return r.graphite.NewHistogram(name, 50)
}

func (r *graphiteRepo) Stop() (err error) {
	defer helpers.RecoverTo(&err)
	r.endSendLoop()
	r.graphite = nil
	return nil
}
