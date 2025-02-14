package main

import (
	"database/sql"
	"embed"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rivo/uniseg"
	"io"
	"net/http"
	"net/url"
	"openheart.tylery.com/internal/request"
	"openheart.tylery.com/internal/response"
)

const maxPayloadByteSize = 64

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
func (app *application) getAll(w http.ResponseWriter, r *http.Request) {
	urlPathValue := request.InputUrl(r.PathValue("url"))

	// Due to a limitation in net/http routing, we cannot do / and wildcard /*
	// To get around this, if the urlPathValue is empty or the root document /
	// We escape into the home page and return. Otherwise, return the emoji count
	if urlPathValue == "" {
		app.homePage(w, r)
		return
	}
	parsedUrl, err := urlPathValue.Parse()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte("INVALID URL"))
		return
	}

	var urlId request.UrlIdColumn
	var emojiRecords []request.EmojiTable

	// We look for the site id record. If none exists, we return 404
	err = app.db.Get(&urlId, "SELECT id FROM site WHERE url=?", parsedUrl)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		app.serverError(w, r, err)
	}
	if errors.Is(err, sql.ErrNoRows) {
		w.WriteHeader(http.StatusNotFound)
		_, err = w.Write([]byte("NOT FOUND"))
		return
	}

	// We look for the all emoji's with this site urlPathValue. If none exists, we return 404
	err = app.db.Select(&emojiRecords, "SELECT id, site_id, emoji, count FROM emoji WHERE site_id=? ORDER BY count DESC", urlId)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		app.serverError(w, r, err)
	}
	if errors.Is(err, sql.ErrNoRows) {
		w.WriteHeader(http.StatusNotFound)
		_, err = w.Write([]byte("NOT FOUND"))
		return
	}

	// We're not interested in revealing all information. We only return the emoji and the count for it
	data := make(map[string]int, len(emojiRecords))
	for i := range emojiRecords {
		data[emojiRecords[i].Emoji.Decode()] = emojiRecords[i].Count
	}

	w.Header().Set("Cache-Control", "max-age=30")
	err = response.JSON(w, http.StatusOK, data)
	if err != nil {
		app.serverError(w, r, err)
	}
}

// Increment the count for a specific emoji by 1
func (app *application) createOne(w http.ResponseWriter, r *http.Request) {
	reader := io.LimitReader(r.Body, maxPayloadByteSize)
	encodedValue := make([]byte, maxPayloadByteSize)
	byteLength, err := reader.Read(encodedValue)
	var emoji request.EmojiT

	// Form submissions are url encoded, so we need to decode them before getting the
	// key rune
	if r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
		escapedValue, err := url.QueryUnescape(string(encodedValue))
		encodedValue, _, _, _ = uniseg.Step([]byte(escapedValue), -1)
		if err != nil {
			app.serverError(w, r, err)
		}
		emoji.Bytes = encodedValue

		// JSON has a specific structure {"emoji": "🌾"}
		// So, we need to convert it to this structure first. Then we parse it
	} else if r.Header.Get("Content-Type") == "application/json" {
		var jInput = struct {
			Emoji string `json:"emoji"`
		}{}
		err = json.Unmarshal(encodedValue[:byteLength], &jInput)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, err = w.Write([]byte("BAD REQUEST"))
			return
		}

		encodedValue, _, _, _ = uniseg.Step([]byte(jInput.Emoji), -1)
		emoji.Bytes = encodedValue

		//	For all other requests, we try to decode the string and get the first rune.
	} else {
		encodedValue, _, _, _ = uniseg.Step(encodedValue, -1)
		emoji.Bytes = encodedValue
	}

	// Let's see if the first rune is an emoji
	emojiRunes, err := emoji.ParseRunes()
	if emojiRunes[0] == 0 || err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte("BAD REQUEST"))
		return
	}

	urlPathValue := request.InputUrl(r.PathValue("url"))
	parsedUrl, err := urlPathValue.Parse()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte("INVALID URL"))
		return
	}
	var urlId request.UrlIdColumn

	var emojiRecord request.EmojiTable

	// First, we get the site id based on the url. We should probably try to parse the url to try and only get
	// relevant data.
	err = app.db.Get(&urlId, "SELECT id FROM site WHERE url=?", parsedUrl)
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
		err = app.db.Get(&emojiRecord, "SELECT id, site_id, emoji, count FROM emoji WHERE site_id=? AND emoji=?", urlId, emoji.DbEncode())
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
		tx.MustExec("INSERT INTO site (url) VALUES (?)", urlPathValue)
		err := tx.Get(&urlId, "SELECT id FROM site WHERE url=?", urlPathValue)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			app.serverError(w, r, err)
		}
	}
	if noEmojiRecord {
		tx.MustExec("INSERT INTO emoji (site_id, emoji) VALUES (?, ?)", urlId, emoji.DbEncode())
	} else {
		updateStmt, _ := tx.Prepare("UPDATE emoji SET count=? WHERE id=?")
		_, err = updateStmt.Exec(emojiRecord.Count+1, emojiRecord.Id)
		if err != nil {
			tx.Rollback()
		}
	}

	// If Accept header is included, we will return the count in that format. Currently only json
	respondCount := r.Header.Get("Accept") == "application/json"

	if respondCount {
		err = tx.Get(&emojiRecord, "SELECT id, site_id, emoji, count FROM emoji WHERE site_id=? AND emoji=?", urlId, emoji.DbEncode())
	}
	tx.Commit()

	app.logger.Info(fmt.Sprintf("%s -> %s reaction!", urlPathValue, emoji.String()))
	var status int
	if noEmojiRecord == true {
		status = http.StatusCreated
	} else {
		status = http.StatusOK
	}
	if respondCount {
		if r.Header.Get("Accept") == "application/json" {
			data := map[string]int{
				emojiRecord.Emoji.Decode(): emojiRecord.Count,
			}
			err = response.JSONWithHeaders(w, status, data, http.Header{
				"Cache-Control": []string{"max-age=30"},
			})
		}
	} else {
		w.Header().Set("Cache-Control", "max-age=30")
		_, err = w.Write([]byte("OK"))
	}
	if err != nil {
		app.serverError(w, r, err)
	}
}

//go:embed templates/home.html
var homeHtml embed.FS

func (app *application) homePage(w http.ResponseWriter, r *http.Request) {
	content, err := homeHtml.ReadFile("templates/home.html")

	if err != nil {
		app.serverError(w, r, fmt.Errorf("error reading template: %v", err))
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(content)
	if err != nil {
		app.serverError(w, r, err)
	}
}
