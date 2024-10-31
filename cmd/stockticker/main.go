package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"

	"stockticker/internal/cache"
	"stockticker/internal/controller"
	"stockticker/internal/monitoring"
	"stockticker/internal/server"
	"stockticker/internal/stockclient"

	"github.com/spf13/pflag"
)

const (
	AppName                   = "stockticker"
	MaxRequestDurationSeconds = 30

	PprofServerHost = "127.0.0.1"
	PprofServerPort = 6060

	PromServerHost = "0.0.0.0"
	PromServerPort = 9102
)

type hostPortType struct {
	Host string
	Port int
}

type cmdArgsType struct {
	ListenAddr  hostPortType
	EnableCache bool
	RedisSrv    hostPortType
}

type envVars struct {
	apiKey  string
	symbol  string
	numDays int
}

func init() {
	logLevelStr, ok := os.LookupEnv("LOG_LEVEL")
	if !ok {
		logLevelStr = "info"
	}
	logLevel, err := log.ParseLevel(logLevelStr)
	if err != nil {
		logLevel = log.InfoLevel
	}
	log.SetLevel(logLevel)
}

func parseCmdArgs() (*cmdArgsType, error) {
	args := &cmdArgsType{}

	flags := pflag.NewFlagSet(AppName, pflag.ExitOnError)
	flags.SortFlags = false

	flags.StringVar(&args.ListenAddr.Host, "listen-ip", "0.0.0.0", "The IP address to listen on for HTTP requests")
	flags.IntVar(&args.ListenAddr.Port, "listen-port", 8080, "The port to listen on for HTTP requests")
	flags.BoolVar(&args.EnableCache, "enable-cache", false, "Enable/disable caching")
	flags.StringVar(&args.RedisSrv.Host, "redis-host", "127.0.0.1", "The Redis host address to connect to")
	flags.IntVar(&args.RedisSrv.Port, "redis-port", 6379, "The Redis port to connect to")
	err := flags.Parse(os.Args[1:])
	return args, err
}

func getEnvVars() (*envVars, error) {
	var valid bool
	var err error
	ev := &envVars{}

	ev.symbol, valid = os.LookupEnv("SYMBOL")
	if !valid {
		return nil, fmt.Errorf("SYMBOL not set")
	}

	numDaysStr, valid := os.LookupEnv("NDAYS")
	if !valid {
		return nil, fmt.Errorf("NDAYS not set")
	}
	ev.numDays, err = strconv.Atoi(numDaysStr)
	if err != nil {
		return nil, fmt.Errorf("NDAYS not a valid integer")
	}
	if ev.numDays <= 0 {
		return nil, fmt.Errorf("NDAYS must be greater than zero")
	}

	ev.apiKey, valid = os.LookupEnv("APIKEY")
	if !valid {
		return nil, fmt.Errorf("APIKEY not set")
	}

	return ev, nil
}

func main() {
	// Config
	cmdArgs, err := parseCmdArgs()
	if err != nil {
		log.Fatalf("Could not parse command line args: %s", err)
	}

	envVars, err := getEnvVars()
	if err != nil {
		log.Fatalf("Could not get required environment variables: %s", err)
	}

	// Pprof
	pprofServer, err := monitoring.NewPprofServer(PprofServerHost, PprofServerPort)
	if err != nil {
		log.Fatalf("Could not create pprof server: %v", err)
	}
	log.Printf("Pprof HTTP server listening on %s:%d", PprofServerHost, PprofServerPort)
	err = pprofServer.Listen()
	if err != nil {
		log.Fatalf("Could not start pprof server: %v", err)
	}

	// Prometheus
	promServer, err := monitoring.NewPrometheusServer(PromServerHost, PromServerPort)
	if err != nil {
		log.Fatalf("Could not create Prometheus server: %v", err)
	}
	log.Infof("Prometheus HTTP server listening on %s:%d", PromServerHost, PromServerPort)
	err = promServer.Listen()
	if err != nil {
		log.Fatalf("Could not start Prometheus server: %v", err)
	}

	// Cache
	var cacheClient cache.Client
	if cmdArgs.EnableCache {
		cacheClient, err = cache.NewRedisClient(cmdArgs.RedisSrv.Host, cmdArgs.RedisSrv.Port)
		if err != nil {
			log.Fatalf("Could not create Redis client: %v", err)
		}
	} else {
		cacheClient, _ = cache.NewNullClient("", 0)
	}
	defer cacheClient.Close()

	// Stock
	av_client, err := stockclient.NewAlphaVantageClient(envVars.apiKey)
	if err != nil {
		log.Fatalf("Could not create a Alpha Vantage client: %v", err)
	}

	stockCtrler, err := controller.NewStockController(av_client, cacheClient, envVars.symbol, envVars.numDays)
	if err != nil {
		log.Fatalf("Could not create stock contorller: %v", err)
	}

	// HTTP server
	server, err := server.NewServer(stockCtrler, cmdArgs.ListenAddr.Host, cmdArgs.ListenAddr.Port)
	if err != nil {
		log.Fatalf("Could not create server: %v", err)
	}
	log.Infof("HTTP server listening on %s:%d", cmdArgs.ListenAddr.Host, cmdArgs.ListenAddr.Port)
	ctx := context.Background()
	server.Start(ctx)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	err = server.Stop(ctx, time.Duration(MaxRequestDurationSeconds*time.Second))
	if err != nil {
		log.Errorf("Failed to gracefully shut down server: %v", err)
	}
}
