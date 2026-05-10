package main

import (
	"context"
	"dns-manager/config"
	"dns-manager/internal/dns"
	"dns-manager/pkg/logger"
	"dns-manager/pkg/middleware"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := config.LoadConfig()

	log, cleanup, err := logger.New(cfg.LogLevel, cfg.LogFilePath)
	if err != nil {
		fmt.Printf("failed to init logger: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	router := http.NewServeMux()

	// Repository
	dnsRepository := dns.NewFileRepository(cfg.ResolvConfPath)

	// Services
	dnsService := dns.NewService(dnsRepository)

	//Handlers
	dns.NewHandler(router, dns.HandlerDeps{
		Service: dnsService,
	})

	// Middlewares
	stack := middleware.Chain(
		middleware.Recover(log),
		middleware.Logging(log),
		middleware.CORS,
	)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           stack(router),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// буфер 1 нужен, чтобы горутина могла записать ошибку, если main-горутина еще не готова прочитать ее
	errCh := make(chan error, 1)

	go func() {
		log.Info("server started",
			"port", cfg.HTTPAddr,
			"resolv_conf_path", cfg.ResolvConfPath,
			"log_level", cfg.LogLevel,
			"log_file_path", cfg.LogFilePath,
		)
		//fmt.Printf("Listening on %s\n", cfg.HTTPAddr)
		//fmt.Printf("Using resolv.conf path: %s\n", cfg.ResolvConfPath) // pwd file

		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		log.Info("Shutting down server\n")
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("Server stopped unexpectedly: %v\n", err)
			os.Exit(1)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("Server shutdown error", "error", err)
		os.Exit(1)
	}

	log.Info("Server stopped")
}
