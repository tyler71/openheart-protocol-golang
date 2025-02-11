package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dmolesUC/emoji"
	"io"
	"net/http"
	"net/url"
	"openheart.tylery.com/internal/response"
	"unicode/utf8"
)

const maxPayloadByteSize = 32

type urlIdColumn int
type emojiTable struct {
	Id     int  `db:"id"`
	SiteId int  `db:"site_id"`
	Emoji  rune `db:"emoji"`
	Count  int  `db:"count"`
}

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
	urlPathValue := r.PathValue("url")

	var urlId urlIdColumn
	var emojiRecords []emojiTable

	// We look for the site id record. If none exists, we return 404
	err := app.db.Get(&urlId, "SELECT id FROM site WHERE url=?", urlPathValue)
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
		data[string(emojiRecords[i].Emoji)] = emojiRecords[i].Count
	}

	err = response.JSON(w, http.StatusOK, data)
	if err != nil {
		app.serverError(w, r, err)
	}
}

// Returns emoji count for a specific url and emoji
func (app *application) getOne(w http.ResponseWriter, r *http.Request) {
	urlPathValue, emojiPathValue := r.PathValue("url"), r.PathValue("emoji")
	var emojiRune rune
	var urlId urlIdColumn
	var emojiRecord emojiTable

	for _, r := range emojiPathValue {
		emojiRune = r
		break
	}

	// We look for the site id record. If none exists, we return 404
	err := app.db.Get(&urlId, "SELECT id FROM site WHERE url=?", urlPathValue)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		app.serverError(w, r, err)
	}
	if errors.Is(err, sql.ErrNoRows) {
		w.WriteHeader(http.StatusNotFound)
		_, err = w.Write([]byte("NOT FOUND"))
		return
	}

	err = app.db.Get(&emojiRecord, "SELECT id, site_id, emoji, count FROM emoji WHERE site_id=? AND emoji=?", urlId, emojiRune)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		app.serverError(w, r, err)
	}
	if errors.Is(err, sql.ErrNoRows) {
		w.WriteHeader(http.StatusNotFound)
		_, err = w.Write([]byte("NOT FOUND"))
		return
	}

	// On the happy path here, we have the record and return the count to the user
	data := map[string]int{
		string(emojiRecord.Emoji): emojiRecord.Count,
	}
	err = response.JSON(w, http.StatusOK, data)
	if err != nil {
		app.serverError(w, r, err)
	}
}

// Increment the count for a specific emoji by 1
func (app *application) createOne(w http.ResponseWriter, r *http.Request) {
	var emojiRune rune
	if r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
		reader := io.LimitReader(r.Body, maxPayloadByteSize)
		encodedValue := make([]byte, maxPayloadByteSize)
		reader.Read(encodedValue)
		escapedValue, err := url.QueryUnescape(string(encodedValue))
		if err != nil {
			app.serverError(w, r, err)
		}
		emojiRune, _ = utf8.DecodeRuneInString(escapedValue)
	} else if r.Header.Get("Content-Type") == "application/json" {
		reader := io.LimitReader(r.Body, maxPayloadByteSize)
		encodedValue := make([]byte, maxPayloadByteSize)
		byteLength, err := reader.Read(encodedValue)
		var request = struct {
			Emoji string `json:"emoji"`
		}{}
		err = json.Unmarshal(encodedValue[:byteLength], &request)
		if err != nil {
			app.serverError(w, r, err)
		}
		emojiRune, _ = utf8.DecodeRuneInString(request.Emoji)
	} else {
		reader := bufio.NewReader(io.LimitReader(r.Body, maxPayloadByteSize))
		e, emojiRuneByteSize, err := reader.ReadRune()
		emojiRune = e
		if err != nil || emojiRuneByteSize == 0 {
			app.serverError(w, r, err)
		}
	}

	var err error
	if !emoji.IsEmoji(emojiRune) {
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte("BAD REQUEST"))
		return
	}

	urlPathValue := r.PathValue("url")
	var urlId urlIdColumn

	var emojiRecord emojiTable

	// First, we get the site id based on the url. We should probably try to parse the url to try and only get
	// relevant data.
	err = app.db.Get(&urlId, "SELECT id FROM site WHERE url=?", urlPathValue)
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
		tx.MustExec("INSERT INTO site (url) VALUES (?)", urlPathValue)
		err := tx.Get(&urlId, "SELECT id FROM site WHERE url=?", urlPathValue)
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

	if noEmojiRecord == true {
		w.WriteHeader(201)
	} else {
		w.WriteHeader(200)
	}

	app.logger.Info(fmt.Sprintf("%s just got a %s reaction!", urlPathValue, string(emojiRune)))
	_, err = w.Write([]byte("OK"))
	if err != nil {
		app.serverError(w, r, err)
	}
}
