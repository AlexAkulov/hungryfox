package searcher

import (
	"github.com/AlexAkulov/hungryfox"
	"github.com/rs/zerolog"
	"gopkg.in/tomb.v2"
)

type Worker struct {
	Analyzer    hungryfox.IDiffAnalyzer
	DiffChannel <-chan *hungryfox.Diff
	Log         zerolog.Logger
	Dying       <-chan struct{}
}

func (w *Worker) Run() error {
	for {
		select {
		case <-w.Dying:
			return tomb.ErrDying
		case diff := <-w.DiffChannel:
			w.Analyzer.Analyze(diff)
		}
	}
}
