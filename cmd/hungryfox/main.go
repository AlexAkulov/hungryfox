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

	"github.com/AlexAkulov/hungryfox/metrics"

	"github.com/AlexAkulov/hungryfox"
	"github.com/AlexAkulov/hungryfox/config"
	"github.com/AlexAkulov/hungryfox/helpers"
	"github.com/AlexAkulov/hungryfox/router"
	"github.com/AlexAkulov/hungryfox/scanmanager"
	"github.com/AlexAkulov/hungryfox/searcher"
	"github.com/AlexAkulov/hungryfox/state/filestate"

	"github.com/natefinch/lumberjack"
	"github.com/rs/zerolog"
)

var (
	version         = "unknown"
	skipScan        = flag.Bool("skip-scan", false, "Update state for all repo")
	configFlag      = flag.String("config", "config.yml", "config file location")
	pprofFlag       = flag.Bool("pprof", false, "Enable listen pprof on :6060")
	printConfigFlag = flag.Bool("default-config", false, "Print default config to stdout and exit")
)

func main() {
	flag.Parse()

	if *printConfigFlag {
		config.PrintDefaultConfig()
		os.Exit(0)
	}

	conf, err := config.LoadConfig(*configFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open config %s: %v\n", "config.yml", err)
		os.Exit(1)
	}

	logger := createLogger(conf.Logging)
	metricsRepo := metrics.StartMetricsRepo(conf.Metrics, logger)

	diffChannel := make(chan *hungryfox.Diff, 100)
	leakChannel := make(chan *hungryfox.Leak, 1)
	vulnsChannel := make(chan *hungryfox.VulnerableDependency, 1)

	if *skipScan {
		stateManager := &filestate.StateManager{
			Location: conf.Common.StateFile,
		}
		if err := stateManager.Start(); err != nil {
			logger.Error().Str("service", "state manager").Str("error", err.Error()).Msg("fail")
			os.Exit(1)
		}
		logger.Debug().Str("service", "state manager").Msg("started")

		logger.Debug().Str("service", "scan manager").Msg("start")
		scanManager := &scanmanager.ScanManager{
			DiffChannel:  diffChannel,
			Log:          logger,
			StateManager: stateManager,
		}
		scanManager.SetConfig(conf)
		scanManager.DryRun()
		stateManager.Stop()
		os.Exit(0)
	}

	logger.Debug().Str("service", "leaks router").Msg("start")
	leakRouter := &router.LeaksRouter{
		LeakChannel:  leakChannel,
		VulnsChannel: vulnsChannel,
		Config:       conf,
		Log:          logger,
	}
	if err := leakRouter.Start(); err != nil {
		logger.Error().Str("service", "leaks router").Str("error", err.Error()).Msg("fail")
		os.Exit(1)
	}
	logger.Debug().Str("service", "leaks router").Msg("started")

	logger.Debug().Str("service", "searcher dispatcher").Msg("start")
	numCPUs := runtime.NumCPU() - 1
	if numCPUs < 1 {
		numCPUs = 1
	}
	if conf.Common.Workers > 0 {
		numCPUs = conf.Common.Workers
	}
	searcherDispatcher := &searcher.SearcherDispatcher{
		Workers:                numCPUs,
		DiffChannel:            diffChannel,
		LeakChannel:            leakChannel,
		VulnerabilitiesChannel: vulnsChannel,
		Log:                    logger,
		Metrics: searcher.Metrics{
			Leaks:           metricsRepo.CreateCounter("leaks.found"),
			Vulnerabilities: metricsRepo.CreateCounter("vulnerabilities.found"),
		},
	}
	if err := searcherDispatcher.Start(conf); err != nil {
		logger.Error().Str("service", "searcher dispatcher").Str("error", err.Error()).Msg("fail")
		os.Exit(1)
	}
	logger.Debug().Str("service", "searcher dispatcher").Int("workers", numCPUs).Msg("started")

	logger.Debug().Str("service", "state manager").Msg("start")
	stateManager := &filestate.StateManager{
		Location: conf.Common.StateFile,
	}
	if err := stateManager.Start(); err != nil {
		logger.Error().Str("service", "state manager").Str("error", err.Error()).Msg("fail")
		os.Exit(1)
	}
	logger.Debug().Str("service", "state manager").Msg("started")

	logger.Debug().Str("service", "scan manager").Msg("start")
	scanManager := &scanmanager.ScanManager{
		DiffChannel:  diffChannel,
		Log:          logger,
		StateManager: stateManager,
	}
	if err := scanManager.Start(conf); err != nil {
		logger.Error().Str("service", "scan manager").Str("error", err.Error()).Msg("fail")
		os.Exit(1)
	}
	logger.Debug().Str("service", "scan manager").Msg("started")

	statusTicker := time.NewTicker(time.Second * 10)
	defer statusTicker.Stop()
	go func() {
		for range statusTicker.C {
			r := scanManager.Status()
			if r != nil {
				l := searcherDispatcher.Status(r.Location.URL)
				logger.Info().Int("leaks", l.LeaksFound).Int("leaks_filtred", l.LeaksFiltered).Str("duration", helpers.PrettyDuration(time.Since(r.Scan.StartTime))).Str("repo", r.Location.URL).Msg("scan")
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
		searcherDispatcher.Update(newConf)
		scanManager.SetConfig(newConf)
		logger.Info().Msg("settings reloaded")
	}

	if err := scanManager.Stop(); err != nil {
		logger.Error().Str("error", err.Error()).Str("service", "scan manager").Msg("can't stop")
	}
	logger.Debug().Str("service", "scan manager").Msg("stopped")

	if err := searcherDispatcher.Stop(); err != nil {
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

	logger.Debug().Str("service", "metrics repository").Msg("stop")
	if err := metricsRepo.Stop(); err != nil {
		logger.Error().Str("error", err.Error()).Str("service", "metrics repository").Msg("can't stop")
	}

	logger.Info().Str("version", version).Msg("stopped")
}

func createLogger(conf *config.Logging) zerolog.Logger {
	var lvl zerolog.Level
	switch conf.Level {
	case "debug":
		lvl = zerolog.DebugLevel
	case "info":
		lvl = zerolog.InfoLevel
	case "warn":
		lvl = zerolog.WarnLevel
	case "error":
		lvl = zerolog.ErrorLevel
	default:
		fmt.Fprintf(os.Stderr, "Unknown logging level '%s'", conf.Level)
		os.Exit(1)
	}
	if conf.File != "" {
		writer := &lumberjack.Logger{
			Filename: conf.File,
			MaxSize:  100, //MB
			MaxAge:   1,   //d
			Compress: true,
		}
		return zerolog.New(writer).Level(lvl).With().Timestamp().Logger()
	} else {
		return zerolog.New(os.Stdout).Level(lvl).With().Timestamp().Logger().Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}
}
