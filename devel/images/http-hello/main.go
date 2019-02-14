package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
)

var (
	port     = flag.String("port", "8080", "port to listen on")
	hostname = "http-hello"
)

func main() {
	host, err := os.Hostname()
	if err != nil {
		log.Printf("Error reading hostname: %v", err)
	} else {
		hostname = host
	}

	http.HandleFunc("/", hello)
	http.ListenAndServe(net.JoinHostPort("", *port), nil)
}

func hello(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(200)
	fmt.Fprintf(w, "Hello!\n")
	fmt.Fprintf(w, "hostname=%s\n", hostname)
}
