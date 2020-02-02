package vulnerabilities

import (
	"time"

	"github.com/AlexAkulov/hungryfox"
	"github.com/facebookgo/muster"
	"github.com/rs/zerolog"
	"gopkg.in/tomb.v2"
)

type VulnerabilitiesWorker struct {
	Searcher    hungryfox.IVulnerabilitySearcher
	DepsChannel <-chan *hungryfox.Dependency
	Log         zerolog.Logger
	Dying       <-chan struct{}
}

type batch struct {
	dependencies []hungryfox.Dependency
	searcher     hungryfox.IVulnerabilitySearcher
	log          zerolog.Logger
}

func (b *batch) Add(item interface{}) {
	b.dependencies = append(b.dependencies, *item.(*hungryfox.Dependency))
}

func (b *batch) Fire(notifier muster.Notifier) {
	defer notifier.Done()
	b.searcher.Search(b.dependencies) //TODO: process error
}

func (w *VulnerabilitiesWorker) MakeBatch() muster.Batch {
	return &batch{
		searcher: w.Searcher,
		log:      w.Log,
	}
}

func (w *VulnerabilitiesWorker) Run() error {
	batchClient := muster.Client{
		MaxBatchSize: 100,
		BatchTimeout: 2 * time.Second,
		BatchMaker:   w.MakeBatch,
	}
	batchClient.Start()
	for {
		select {
		case <-w.Dying:
			return tomb.ErrDying
		case dep := <-w.DepsChannel:
			batchClient.Work <- dep
		}
	}
}
