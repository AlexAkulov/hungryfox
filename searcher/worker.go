package searcher

import (
	"github.com/AlexAkulov/hungryfox"
	"github.com/rs/zerolog"
)

type Worker struct {
	Analyzer    hungryfox.IDiffAnalyzer
	DiffChannel <-chan *hungryfox.Diff
	Log         zerolog.Logger
	Done        <-chan struct{}
}

func (w *Worker) Run() error {
	for {
		select {
		case <-w.Done:
			return nil
		case diff := <-w.DiffChannel:
			w.Analyzer.Analyze(diff)
		}
	}
}
