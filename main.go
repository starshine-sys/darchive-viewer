package main

import (
	"flag"
	"fmt"
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
	r.Get(`/`, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `I can't be bothered to write some HTML for this so:
Just copy-paste the Discord attachment link, but replace https://cdn.discordapp.com/attachments/ in the link with this website.

Source: https://github.com/starshine-sys/darchive-viewer`)
	})

	port = strings.TrimPrefix(port, ":")

	log.Printf("Serving on :%v", port)

	log.Fatal(http.ListenAndServe(":"+port, r))
}
