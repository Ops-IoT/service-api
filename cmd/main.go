package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Ops-IoT/service-api/internal/platform/db"
	"github.com/Ops-IoT/service-api/internal/platform/flag"
	"github.com/kelseyhightower/envconfig"
)

func main() {

	log := log.New(os.Stdout, "SERVICE : ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	var cfg struct {
		Web struct {
			APIHost         string        `default:"0.0.0.0:3000" envconfig:"API_HOST"`
			ReadTimeout     time.Duration `default:"5s" envconfig:"READ_TIMEOUT"`
			WriteTimeout    time.Duration `default:"5s" envconfig:"WRITE_TIMEOUT"`
			ShutdownTimeout time.Duration `default:"5s" envconfig:"SHUTDOWN_TIMEOUT"`
		}
		DB struct {
			MaxIdle   int    `default:"10" envconfig:"MAX_IDLE"`
			MaxActive int    `default:"15" envconfig:"MAX_ACTIVE"`
			Host      string `default:"127.0.0.1:3306" envconfig:"DB_HOST"`
			Name      string `envconfig:"DB_NAME"`
			User      string `envconfig:"DB_USER"`
			Pass      string `envconfig:"DB_PASS"`
		}
	}

	if err := envconfig.Process("SERVICE", &cfg); err != nil {
		log.Fatalf("main : Parsing Config : %v", err)
	}

	if err := flag.Process(&cfg); err != nil {
		if err != flag.ErrHelp {
			log.Fatalf("main : Parsing Command Line : %v", err)
		}
		return
	}

	log.Println("main : Started : Application Initializing")
	defer log.Println("main : Completed")

	cfgJSON, err := json.MarshalIndent(cfg, "", "	")
	if err != nil {
		log.Fatalf("main : Marshalling Config to JSON : %v", err)
	}
	log.Printf("main : Config : %v\n", string(cfgJSON))

	log.Println("main : Started : Initializing MySQL")
	connDB, err := db.New(cfg.DB.Host, cfg.DB.Name, cfg.DB.User, cfg.DB.Pass, cfg.DB.MaxIdle, cfg.DB.MaxActive)
	if err != nil {
		log.Fatalf("main : Register DB : %v", err)
	}
	defer connDB.Close()

	api := http.Server{
		Addr:           cfg.Web.APIHost,
		ReadTimeout:    cfg.Web.ReadTimeout,
		WriteTimeout:   cfg.WriteTimeout,
		MaxHeaderBytes: 1 << 20,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("main : API Listening")
		serverErrors <- api.ListenAndServe()
	}()

	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatalf("main : Error starting server: %v", err)
	case <-osSignals:
		log.Println("main : Start shutdown")

		ctx, cancel := context.WithTimeout(context.Background(), cfg.Web.ShutdownTimeout)
		defer cancel()

		if err := api.Shutdown(ctx); err != nil {
			log.Printf("main : Graceful shutdown did not complete in %v : %v", cfg.Web.ShutdownTimeout, err)
			if err := api.Close(); err != nil {
				log.Fatalf("main : Could not stop http server: %v", err)
			}
		}
	}
}
