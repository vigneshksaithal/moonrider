package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"moonrider/backend/internal/domain"
	"moonrider/backend/internal/engine"
	"moonrider/backend/internal/httpapi"
	"moonrider/backend/internal/mqttclient"
	"moonrider/backend/internal/simulator"
	"moonrider/backend/internal/store"
)

func main() {
	if err := run(); err != nil {
		slog.Error("startup failed", "error", err)
		os.Exit(1)
	}
}
func run() error {
	demo := flag.Bool("demo", false, "in-memory cycling demo; bypasses MQTT and PostgreSQL")
	flag.Parse()
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var repo store.Repository
	if *demo {
		repo = store.NewMemory()
	} else {
		url := os.Getenv("DATABASE_URL")
		if url == "" {
			return fmt.Errorf("DATABASE_URL is required (or use --demo)")
		}
		work, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		p, err := store.Open(work, url)
		if err != nil {
			return err
		}
		repo = p
	}
	defer repo.Close()
	rules := domain.DefaultRules()

	e, err := engine.New(ctx, repo, rules, *demo)
	if err != nil {
		return err
	}
	queue := make(chan engine.Delivery, 64)
	workerDone := make(chan struct{})
	go func() { defer close(workerDone); e.Run(ctx, queue) }()
	defer func() { stop(); <-workerDone }()
	ready := func() bool { return true }
	if *demo {
		go func() {
			err := simulator.Run(ctx, "cycle", false, func(r domain.Reading) error {
				select {
				case queue <- engine.Delivery{Reading: r}:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			})
			if err != nil {
				slog.Error("demo stopped", "error", err)
			}
		}()
	} else {
		sub, err := mqttclient.Subscribe(ctx, queue)
		if err != nil {
			return err
		}
		defer sub.Close()
		ready = sub.Ready
	}
	api := &httpapi.Server{Engine: e, Repo: repo, MQTTReady: ready, Origins: strings.Split(mqttclient.Env("ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173"), ",")}
	server := &http.Server{Addr: mqttclient.Env("HTTP_ADDR", "127.0.0.1:8080"), Handler: api.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	server.BaseContext = func(net.Listener) context.Context { return ctx }
	failed := make(chan error, 1)
	go func() {
		slog.Info("Moonrider listening", "address", server.Addr, "demo", *demo)
		failed <- server.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
	case err := <-failed:
		if err != http.ErrServerClosed {
			return err
		}
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(shutdown)
}
