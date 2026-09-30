package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nativepane/internal/server"
)

//go:embed web/*
var assets embed.FS

func main() {
	cfg, e := server.EnvConfig()
	if e != nil {
		log.Fatal(e)
	}
	web, _ := fs.Sub(assets, "web")
	app, e := server.New(cfg, web)
	if e != nil {
		log.Fatal(e)
	}
	bind := os.Getenv("BIND_ADDR")
	if bind == "" {
		bind = "127.0.0.1"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{Addr: net.JoinHostPort(bind, port), Handler: app, ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 60 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16384}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				app.Cleanup()
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	if cfg.Auth == "none" {
		log.Print("AUTH=none: local demonstration only; never expose this mode publicly")
	}
	log.Printf("NativePane listening on %s", srv.Addr)
	if e = srv.ListenAndServe(); e != nil && e != http.ErrServerClosed {
		log.Fatal(e)
	}
}
