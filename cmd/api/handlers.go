package main

import (
	"fmt"
	"net/http"

	"openheart.tylery.com/internal/response"
)

func (app *application) status(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{
		"status": "OK",
	}
	err := response.JSON(w, http.StatusOK, data)
	if err != nil {
		app.serverError(w, r, err)
	}
}

// Returns all emoji's for a given url
func (app *application) get(w http.ResponseWriter, r *http.Request) {
	mockEmoji := map[string]int{
		"😀": 2,
		"🥰": 1,
	}

	url := r.PathValue("url")
	fmt.Println(url)
	var data map[string]int = mockEmoji

	err := response.JSON(w, http.StatusOK, data)
	if err != nil {
		app.serverError(w, r, err)
	}
}

// Returns emoji count for a specific url and emoji
func (app *application) getOne(w http.ResponseWriter, r *http.Request) {
	mockEmoji := map[string]int{
		"😀": 2,
		"🥰": 1,
	}
	url, emoji := r.PathValue("url"), r.PathValue("emoji")
	fmt.Println(url, emoji)

	data := mockEmoji[emoji]

	err := response.JSON(w, http.StatusOK, data)
	if err != nil {
		app.serverError(w, r, err)
	}
}

// Increment the count for a specific emoji by 1
func (app *application) create(w http.ResponseWriter, r *http.Request) {
	url, emoji := r.PathValue("url"), r.PathValue("emoji")
	fmt.Println(url, emoji)

	w.WriteHeader(201)
	_, err := w.Write([]byte("OK"))
	if err != nil {
		app.serverError(w, r, err)
	}
}
