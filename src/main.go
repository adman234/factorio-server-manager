package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/OpenFactorioServerManager/factorio-server-manager/api"
	"github.com/OpenFactorioServerManager/factorio-server-manager/bootstrap"
	"github.com/OpenFactorioServerManager/factorio-server-manager/factorio"
)

// factorioStopTimeout is how long FSM waits for Factorio to save and exit on
// shutdown before killing it. Keep it below the container stop grace period.
const factorioStopTimeout = 45 * time.Second

func main() {
	// get the all configs based on the flags
	config := bootstrap.NewConfig(os.Args[1:])

	// setup the required files for the mods
	factorio.ModStartUp()

	// Initialize Factorio Server struct
	err := factorio.NewFactorioServer()
	if err != nil {
		log.Fatalf("Error occurred during Server initialization: %v", err)
	}

	// Initialize authentication system
	api.SetupAuth()

	// Initialize HTTP router -- also initializes websocket
	router := api.NewRouter()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	httpServer := &http.Server{
		Addr:              config.ServerIP + ":" + config.ServerPort,
		Handler:           router,
		ReadHeaderTimeout: 30 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Printf("Starting server on: %s", httpServer.Addr)
		serveErr <- httpServer.ListenAndServe()
	}()

	exitCode := 0
	select {
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server failed: %s", err)
			exitCode = 1
		}
	case <-ctx.Done():
		log.Printf("Received shutdown signal")
	}
	stop()

	// Save and stop Factorio first, so a `docker stop` never loses progress.
	if err := factorio.GetFactorioServer().StopAndWait(factorioStopTimeout); err != nil {
		log.Printf("Error stopping Factorio server: %s", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error shutting down HTTP server: %s", err)
	}
	log.Printf("Factorio Server Manager stopped")
	if exitCode != 0 {
		stop()
		cancel()
		os.Exit(exitCode)
	}
}
