package main

import (
	"flag"
	"net"
	"net/http"

	"github.com/davecgh/go-spew/spew"
)

var (
	port = flag.String("port", "8080", "port to listen on")
)

func main() {
	http.HandleFunc("/", dump)
	http.ListenAndServe(net.JoinHostPort("", *port), nil)
}

func dump(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(200)

	spew.Fdump(w, r)
}
