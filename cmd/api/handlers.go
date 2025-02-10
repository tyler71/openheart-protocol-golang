package main

import (
	"bufio"
	"fmt"
	"github.com/dmolesUC/emoji"
	"io"
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

var Url int

// Increment the count for a specific emoji by 1
func (app *application) create(w http.ResponseWriter, r *http.Request) {
	reader := bufio.NewReader(io.LimitReader(r.Body, 64))
	emojiRune, emojiRuneSize, err := reader.ReadRune()
	if err != nil || emojiRuneSize == 0 {
		app.serverError(w, r, err)
	}
	url := r.PathValue("url")
	fmt.Println(emojiRune, url, string(emojiRune))
	fmt.Println(emoji.IsEmoji(emojiRune))
	return
	//var urlId int
	//var emojiRecord struct {
	//	id     int
	//	siteId int
	//	emoji  string
	//	count  int
	//}
	//
	//// First, we get the site id based on the url. We should probably try to parse the url to try and only get
	//// relevant data.
	//err = app.db.Get(&urlId, "SELECT id FROM site WHERE url=$1", url)
	//if err != nil || urlId == 0 {
	//	app.serverError(w, r, err)
	//}
	//
	//// Next, we get the emoji count for this specific url. It might not exist! We need to check for this and create a new one
	//// if this is the case
	//err = app.db.Get(&emojiRecord, "SELECT id, site_id, emoji, count FROM emoji WHERE site_id=$1 AND emoji=$2", urlId, emoji)
	//if err != nil || urlId == 0 {
	//	app.serverError(w, r, err)
	//}
	//
	//// With an existing, we need to update the value by incrementing by one. If it doesn't exist, creating the record
	//// starts the count at 1, so we're good.
	//tx := app.db.MustBegin()
	//fmt.Println(url, emoji)
	//
	//w.WriteHeader(201)
	//_, err := w.Write([]byte("OK"))
	//if err != nil {
	//	app.serverError(w, r, err)
	//}
}
