package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/parandhamareddybommaka/kube/pkg/api/kaas"
	kaasdb "github.com/parandhamareddybommaka/kube/pkg/db"
	"github.com/parandhamareddybommaka/kube/pkg/lbmgr"
	"github.com/parandhamareddybommaka/kube/pkg/provisioner"
)

func main() {
	addr := flag.String("addr", ":8080", "API server address")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := http.NewServeMux()

	if _, err := kaasdb.Connect(ctx); err != nil {
		slog.Warn("mongodb unavailable; KaaS endpoints will return 503", "err", err)
	} else {
		slog.Info("mongodb connected")
	}

	var prov provisioner.Provisioner
	p, err := provisioner.NewDockerK3sProvisioner(kaas.DBStatusCallback())
	if err != nil {
		slog.Warn("docker provisioner unavailable; cluster endpoints will error", "err", err)
	} else {
		prov = p
	}

	var lbmgrInst *lbmgr.Manager
	if lm, err := lbmgr.New(); err != nil {
		slog.Warn("docker lb manager unavailable; lb endpoints will error", "err", err)
	} else {
		lbmgrInst = lm
	}

	kaasServer := kaas.NewServer(prov, lbmgrInst, slog.Default())
	kaasServer.Mount(mux)

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:         *addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		fmt.Printf("API server listening on %s\n", *addr)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	_ = server.Shutdown(shutdownCtx)
	if err := kaasdb.Disconnect(shutdownCtx); err != nil {
		slog.Warn("mongo disconnect", "err", err)
	}
}
