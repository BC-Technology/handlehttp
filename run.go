package handlehttp

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
)

func Run(ctx context.Context, logger Logger, srv http.Handler, host, port string) {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	httpServer := &http.Server{
		Addr:         net.JoinHostPort(host, port),
		Handler:      srv,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		logger.Infof("Listening on %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("Error listening and serving: %s", err)
		}
	}()

	var wg sync.WaitGroup

	wg.Add(1)

	go func() {
		defer wg.Done()

		<-ctx.Done()

		// make a new context for the Shutdown (thanks Alessandro Rosetti)
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Errorf("Error shutting down HTTP server: %s", err)
		}
	}()

	wg.Wait()

}
