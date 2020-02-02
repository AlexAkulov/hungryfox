package searcher

import (
	"github.com/AlexAkulov/hungryfox"
	"github.com/rs/zerolog"
	"gopkg.in/tomb.v2"
)

type IDiffProcessor interface {
	Process(*hungryfox.Diff)
}

type Worker struct {
	Searcher    IDiffProcessor
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
			w.Searcher.Process(diff)
		}
	}
}
