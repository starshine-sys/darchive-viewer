package main

import (
	"flag"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var port string

func init() {
	flag.StringVar(&port, "port", "8581", "Port to listen on")
}

func main() {
	r := chi.NewMux()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get(`/{channelID:\d+}/{attachmentID:\d+}/{attachment}`, serve)

	port = strings.TrimPrefix(port, ":")

	log.Printf("Serving on :%v", port)

	log.Fatal(http.ListenAndServe(":"+port, r))
}
