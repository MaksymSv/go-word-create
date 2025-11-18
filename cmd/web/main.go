package main

import (
	context "context"
	flag "flag"
	log "log"
	nethttp "net/http"
	os "os"
	signal "os/signal"
	pathfile "path/filepath"
	strings "strings"
	time "time"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	staticDir := flag.String("static", "./static", "Static files directory")
	flag.Parse()

	fs := nethttp.FileServer(nethttp.Dir(*staticDir))
	mux := nethttp.NewServeMux()

	// Serve static files
	mux.Handle("/static/", nethttp.StripPrefix("/static/", fs))

	// SPA fallback: serve index.html for root and unknown paths (except /api calls)
	mux.HandleFunc("/", func(w nethttp.ResponseWriter, r *nethttp.Request) {
		p := r.URL.Path
		// API routes should not fallback to index
		if strings.HasPrefix(p, "/api") {
			nethttp.NotFound(w, r)
			return
		}
		// If path resolves to an existing file under staticDir, let file server handle it
		full := pathfile.Join(*staticDir, p)
		if p == "/" {
			httpServeFile(w, r, pathfile.Join(*staticDir, "index.html"))
			return
		}
		// Try to open the file
		if _, err := os.Stat(full); err == nil {
			httpServeFile(w, r, full)
			return
		}
		// Fallback to index
		httpServeFile(w, r, pathfile.Join(*staticDir, "index.html"))
	})

	srv := &nethttp.Server{
		Addr:    *addr,
		Handler: mux,
	}

	go func() {
		log.Printf("Starting web server on %s, serving static from %s", *addr, *staticDir)
		if err := srv.ListenAndServe(); err != nil && err != nethttp.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
	log.Println("Server exited properly")
}

// httpServeFile is a tiny wrapper to serve a specific file with correct content-type
func httpServeFile(w nethttp.ResponseWriter, r *nethttp.Request, filename string) {
	f, err := os.Open(filename)
	if err != nil {
		nethttp.NotFound(w, r)
		return
	}
	defer f.Close()
	nethttp.ServeContent(w, r, filename, time.Now(), f)
}
