package machinestats

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

// StartHTTPServer starts an HTTP server on the specified port and returns a stop function.
func StartHTTPServer(port int) (mux *http.ServeMux, stop func() error, err error) {
	mux = http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	// Create a listener
	var listener net.Listener
	listener, err = net.Listen("tcp", server.Addr)
	if err != nil {
		err = fmt.Errorf("failed to listen on port %d: %w", port, err)
		return
	}

	// Channel to signal server shutdown
	stopChan := make(chan struct{})

	// Start the server in a new goroutine
	go func() {
		defer close(stopChan)
		if err := server.Serve(listener); err != http.ErrServerClosed {
			fmt.Printf("HTTP server ListenAndServe: %v\n", err)
		}
	}()

	// Return a stop function
	stop = func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := server.Shutdown(ctx)
		<-stopChan
		return err
	}
	return
}
