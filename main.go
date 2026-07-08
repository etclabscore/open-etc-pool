package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"

	"github.com/yvasiyarov/gorelic"

	"github.com/etclabscore/open-etc-pool/api"
	"github.com/etclabscore/open-etc-pool/payouts"
	"github.com/etclabscore/open-etc-pool/proxy"
	"github.com/etclabscore/open-etc-pool/storage"
)

var cfg proxy.Config
var backend *storage.RedisClient

func startProxy() {
	s := proxy.NewProxy(&cfg, backend)
	s.Start()
}

func startApi() {
	s := api.NewApiServer(&cfg.Api, backend)
	s.Start()
}

func startBlockUnlocker(ctx context.Context) {
	u := payouts.NewBlockUnlocker(&cfg.BlockUnlocker, backend, &cfg.Network)
	u.Start(ctx)
}

func startPayoutsProcessor(ctx context.Context) {
	u := payouts.NewPayoutsProcessor(&cfg.Payouts, backend)
	u.Start(ctx)
}

func startNewrelic() {
	if cfg.NewrelicEnabled {
		nr := gorelic.NewAgent()
		nr.Verbose = cfg.NewrelicVerbose
		nr.NewrelicLicense = cfg.NewrelicKey
		nr.NewrelicName = cfg.NewrelicName
		nr.Run()
	}
}

func readConfig(cfg *proxy.Config) {
	configFileName := "config.json"
	if len(os.Args) > 1 {
		configFileName = os.Args[1]
	}
	configFileName, _ = filepath.Abs(configFileName)
	log.Printf("Loading config: %v", configFileName)

	configFile, err := os.Open(configFileName)
	if err != nil {
		log.Fatal("File error: ", err.Error())
	}
	defer configFile.Close()
	jsonParser := json.NewDecoder(configFile)
	if err := jsonParser.Decode(&cfg); err != nil {
		log.Fatal("Config error: ", err.Error())
	}
}

func main() {
	readConfig(&cfg)

	if cfg.Threads > 0 {
		runtime.GOMAXPROCS(cfg.Threads)
		log.Printf("Running with %v threads", cfg.Threads)
	}

	startNewrelic()

	backend = storage.NewRedisClient(&cfg.Redis, cfg.Coin)
	pong, err := backend.Check()
	if err != nil {
		log.Printf("Can't establish connection to backend: %v", err)
	} else {
		log.Printf("Backend check reply: %v", pong)
	}

	// Shut down cleanly on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// The proxy and API are network servers, torn down when the process exits.
	// The block unlocker and payouts processor mutate balances, so we wait for
	// them to finish the current cycle before exiting.
	var wg sync.WaitGroup

	if cfg.Proxy.Enabled {
		go startProxy()
	}
	if cfg.Api.Enabled {
		go startApi()
	}
	if cfg.BlockUnlocker.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			startBlockUnlocker(ctx)
		}()
	}
	if cfg.Payouts.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			startPayoutsProcessor(ctx)
		}()
	}

	<-ctx.Done()
	stop() // a second signal terminates immediately
	log.Println("Shutting down; waiting for unlocker/payouts to finish the current cycle...")
	wg.Wait()
	log.Println("Shutdown complete")
}
