package main

import (
	"context"
	"errors"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger := log.New(os.Stdout, "", log.LstdFlags)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	srv := http.Server{
		Addr: ":" + port,
	}

	http.HandleFunc("/", handler)

	// Start the server
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatalf("Server failed to start: %v", err)
		}
	}()

	logger.Printf("Server started on %s", srv.Addr)

	// Wait for the signal
	<-ctx.Done()

	timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer timeoutCancel()

	// Gracefully shutdown the server
	if err := srv.Shutdown(timeoutCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Fatalf("Server failed to shutdown: %v", err)
	}

	logger.Println("Server shutdown")
}

var responseTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<title>{{.Host}}</title>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex, nofollow">
<style>
body {
    font-family: Arial, sans-serif;
    margin: 2em;
}

.notice {
  text-align: center;
}
</style>
</head>
<body>
<div class="notice">
<h1>{{.Host}}</h1>
<p>You can reach me at <a href="mailto:{{.Mail}}"><code>{{.Mail}}</code></a>.</p>
</div>
</body>
</html>`

func handler(w http.ResponseWriter, r *http.Request) {
	var host string
	host = r.Header.Get("X-Forwarded-Host")

	if host == "" {
		host = r.Host
	}

	data := struct {
		Host string
		Mail string
	}{
		Host: host,
		Mail: "vojtech@mares.cz",
	}

	t, err := template.New("response").Parse(responseTemplate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := t.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
