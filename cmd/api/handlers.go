package main

import (
	"bufio"
	"database/sql"
	"errors"
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
	var urlId int
	var emojiRecord struct {
		Id     int    `db:"id"`
		SiteId int    `db:"site_id"`
		Emoji  string `db:"emoji"`
		Count  int    `db:"count"`
	}

	// First, we get the site id based on the url. We should probably try to parse the url to try and only get
	// relevant data.
	err = app.db.Get(&urlId, "SELECT id FROM site WHERE url=?", url)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		app.serverError(w, r, err)
	}
	var noSiteRecord bool
	var noEmojiRecord bool
	if urlId == 0 {
		noSiteRecord = true
		noEmojiRecord = true
	} else {
		// Next, we get the emoji count for this specific url. It might not exist! We need to check for this and create a new one
		// if this is the case
		err = app.db.Get(&emojiRecord, "SELECT id, site_id, emoji, count FROM emoji WHERE site_id=? AND emoji=?", urlId, emojiRune)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			app.serverError(w, r, err)
		}
		if emojiRecord.Id == 0 {
			noEmojiRecord = true
		}
	}

	// With an existing, we need to update the value by incrementing by one. If it doesn't exist, creating the record
	// starts the count at 1, so we're good.
	tx := app.db.MustBegin()
	defer tx.Rollback()
	if noSiteRecord {
		tx.MustExec("INSERT INTO site (url) VALUES (?)", url)
		err := tx.Get(&urlId, "SELECT id FROM site WHERE url=?", url)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			app.serverError(w, r, err)
		}
	}
	if noEmojiRecord {
		tx.MustExec("INSERT INTO emoji (site_id, emoji) VALUES (?, ?)", urlId, emojiRune)
	} else {
		tx.MustExec("UPDATE emoji SET count=? WHERE id=?", emojiRecord.Count+1, emojiRecord.Id)
	}

	tx.Commit()
	fmt.Println(url, emojiRecord.Emoji)

	if noEmojiRecord == true || noSiteRecord == true {
		w.WriteHeader(201)
	} else {
		w.WriteHeader(200)
	}
	_, err = w.Write([]byte("OK"))
	if err != nil {
		app.serverError(w, r, err)
	}
}
