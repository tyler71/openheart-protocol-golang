package main

import (
	"net/http"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /status", app.status)
	mux.HandleFunc("GET /{url}", app.get)
	mux.HandleFunc("GET /{url}/{emoji}", app.getOne)
	mux.HandleFunc("POST /{url}", app.create)

	return app.recoverPanic(mux)
}
