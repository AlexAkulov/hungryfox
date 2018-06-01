package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/AlexAkulov/hungryfox"
	"github.com/AlexAkulov/hungryfox/config"
	"github.com/AlexAkulov/hungryfox/helpers"
	"github.com/AlexAkulov/hungryfox/router"
	"github.com/AlexAkulov/hungryfox/scanmanager"
	"github.com/AlexAkulov/hungryfox/searcher"
	"github.com/AlexAkulov/hungryfox/state/filestate"

	"github.com/rs/zerolog"
)

var (
	version    = "unknown"
	skipScan   = flag.Bool("skip-scan", false, "Update state for all repo")
	configFlag = flag.String("config", "config.yml", "config file location")
	pprofFlag  = flag.Bool("pprof", false, "Enable listen pprof on :6060")
)

func main() {
	flag.Parse()

	conf, err := config.LoadConfig(*configFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open config %s: %v\n", "config.yml", err)
		os.Exit(1)
	}

	var lvl zerolog.Level
	switch conf.Common.LogLevel {
	case "debug":
		lvl = zerolog.DebugLevel
	case "info":
		lvl = zerolog.InfoLevel
	case "warn":
		lvl = zerolog.WarnLevel
	case "error":
		lvl = zerolog.ErrorLevel
	default:
		fmt.Fprintf(os.Stderr, "Unknown log_level '%s'", conf.Common.LogLevel)
		os.Exit(1)
	}
	// logger := zerolog.New(os.Stdout).Level(lvl).With().Timestamp().Logger()
	logger := zerolog.New(os.Stdout).Level(lvl).With().Timestamp().Logger().Output(zerolog.ConsoleWriter{Out: os.Stdout})

	diffChannel := make(chan *hungryfox.Diff, 100)
	leakChannel := make(chan *hungryfox.Leak, 1)

	if *skipScan {
		stateManager := filestate.StateManager{
			Location: conf.Common.StateFile,
		}
		if err := stateManager.Start(); err != nil {
			logger.Error().Str("service", "state manager").Str("error", err.Error()).Msg("fail")
			os.Exit(1)
		}
		logger.Debug().Str("service", "state manager").Msg("started")

		logger.Debug().Str("service", "scan manager").Msg("start")
		scanManager := &scanmanager.ScanManager{
			DiffChannel: diffChannel,
			Log:         logger,
			State:       stateManager,
		}
		scanManager.SetConfig(conf)
		scanManager.DryRun()
		stateManager.Stop()
		os.Exit(0)
	}

	logger.Debug().Str("service", "leaks router").Msg("start")
	leakRouter := &router.LeaksRouter{
		LeakChannel: leakChannel,
		Config:      conf,
		Log:         logger,
	}
	if err := leakRouter.Start(); err != nil {
		logger.Error().Str("service", "leaks router").Str("error", err.Error()).Msg("fail")
		os.Exit(1)
	}
	logger.Debug().Str("service", "leaks router").Msg("strated")

	logger.Debug().Str("service", "leaks searcher").Msg("start")

	numCPUs := runtime.NumCPU() - 1
	if numCPUs < 1 {
		numCPUs = 1
	}
	leakSearcher := &searcher.Searcher{
		Workers:     numCPUs,
		DiffChannel: diffChannel,
		LeakChannel: leakChannel,
	}
	leakSearcher.SetConfig(conf)
	if err := leakSearcher.Start(); err != nil {
		logger.Error().Str("service", "leaks searcher").Str("error", err.Error()).Msg("fail")
		os.Exit(1)
	}
	logger.Debug().Str("service", "leaks searcher").Msg("started")

	logger.Debug().Str("service", "state manager").Msg("start")
	stateManager := filestate.StateManager{
		Location: conf.Common.StateFile,
	}
	if err := stateManager.Start(); err != nil {
		logger.Error().Str("service", "state manager").Str("error", err.Error()).Msg("fail")
		os.Exit(1)
	}
	logger.Debug().Str("service", "state manager").Msg("started")

	logger.Debug().Str("service", "scan manager").Msg("start")
	scanManager := &scanmanager.ScanManager{
		DiffChannel: diffChannel,
		Log:         logger,
		State:       stateManager,
	}
	if err := scanManager.Start(conf); err != nil {
		logger.Error().Str("service", "scan manager").Str("error", err.Error()).Msg("fail")
		os.Exit(1)
	}
	logger.Debug().Str("service", "scan manager").Msg("started")

	statusTicker := time.NewTicker(time.Second * 60)
	defer statusTicker.Stop()
	go func() {
		for range statusTicker.C {
			r, s := scanManager.Status()
			if r != nil {
				l := leakSearcher.Status(r.RepoURL)
				logger.Info().Int("total", s.CommitsTotal).Int("scanned", s.CommitsScanned).Int("leaks", l.LeaksFound).Int("leaks_filtred", l.LeaksFiltred).Str("duration", helpers.PrettyDuration(time.Since(s.StartTime))).Str("repo", r.RepoURL).Msg("scan")
				continue
			}
		}
	}()
	if *pprofFlag {
		go func() {
			if err := http.ListenAndServe(":6060", nil); err != nil {
				logger.Error().Str("error", err.Error()).Msg("can't start pprof")
			}
		}()
	}

	logger.Info().Str("version", version).Msg("started")

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	for {
		s := <-signalChannel
		logger.Info().Str("signal", s.String()).Msg("received signal")
		if s != syscall.SIGHUP {
			break
		}

		newConf, err := config.LoadConfig("config.yml")
		if err != nil {
			logger.Error().Str("error", err.Error()).Msg("can't update config")
			continue
		}
		leakSearcher.SetConfig(newConf)
		scanManager.SetConfig(newConf)
		logger.Info().Msg("settings reloaded")
	}

	if err := scanManager.Stop(); err != nil {
		logger.Error().Str("error", err.Error()).Str("service", "scan manager").Msg("can't stop")
	}
	logger.Debug().Str("service", "scan manager").Msg("stopped")

	if err := leakSearcher.Stop(); err != nil {
		logger.Error().Str("error", err.Error()).Str("service", "leak searcher").Msg("can't stop")
	}
	logger.Debug().Str("service", "leak searcher").Msg("stopped")

	if err := leakRouter.Stop(); err != nil {
		logger.Error().Str("error", err.Error()).Str("service", "leaks router").Msg("can't stop")
	}
	logger.Debug().Str("service", "leaks router").Msg("stopped")

	logger.Debug().Str("service", "state manager").Msg("stop")
	if err := stateManager.Stop(); err != nil {
		logger.Error().Str("error", err.Error()).Str("service", "state manager").Msg("can't stop")
	}

	logger.Info().Str("version", version).Msg("stopped")
}
