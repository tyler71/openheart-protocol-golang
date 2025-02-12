package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dmolesUC/emoji"
	"io"
	"net/http"
	"net/url"
	"openheart.tylery.com/internal/response"
	"strconv"
	"strings"
)

const maxPayloadByteSize = 4

type urlIdColumn int
type emojiTable struct {
	Id     int          `db:"id"`
	SiteId int          `db:"site_id"`
	Emoji  emojiStringT `db:"emoji"`
	Count  int          `db:"count"`
}

type emojiStringT string

type emojiRunesT []rune

func (e emojiRunesT) stringJoin(separator string) string {
	var dbString = make([]string, len(e))
	var count int
	for i := range e {
		if e[i] == 0 {
			break
		}
		a := int(e[i])
		dbString[i] = strconv.Itoa(a)
		count++
	}
	if count > 1 {
		return strings.Join(dbString, separator)
	} else {
		return dbString[0]
	}
}
func (e emojiRunesT) dbEncode() string {
	return e.stringJoin("|")
}
func (e emojiRunesT) dbDecode() string {
	return string(e)
}

func (es emojiStringT) parseRunes() (emojiRunesT, error) {
	emojiRunes := make(emojiRunesT, maxPayloadByteSize/4)
	var count int
	for _, r := range es {
		if emoji.IsEmoji(r) {
			emojiRunes[count] = r
			count++
		} else {
			break
		}
	}
	if emojiRunes[0] == 0 {
		return emojiRunes, errors.New("no emoji found")
	}
	return emojiRunes[:count], nil
}
func (es emojiStringT) decodeDb() string {
	splitStrings := strings.Split(string(es), "|")
	emojiRunes := make(emojiRunesT, len(splitStrings))
	for i, _ := range splitStrings {
		parsedInt, _ := strconv.Atoi(splitStrings[i])
		emojiRunes[i] = rune(parsedInt)
	}
	return emojiRunes.dbDecode()
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
		data[emojiRecords[i].Emoji.decodeDb()] = emojiRecords[i].Count
	}

	err = response.JSON(w, http.StatusOK, data)
	if err != nil {
		app.serverError(w, r, err)
	}
}

// Returns emoji count for a specific url and emoji
func (app *application) getOne(w http.ResponseWriter, r *http.Request) {
	urlPathValue, emojiPathValue := r.PathValue("url"), emojiStringT(r.PathValue("emoji"))
	var urlId urlIdColumn
	var emojiRecord emojiTable

	emojiRunes, err := emojiPathValue.parseRunes()
	if err != nil {
		app.serverError(w, r, err)
	}

	// We look for the site id record. If none exists, we return 404
	err = app.db.Get(&urlId, "SELECT id FROM site WHERE url=?", urlPathValue)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		app.serverError(w, r, err)
	}
	if errors.Is(err, sql.ErrNoRows) {
		w.WriteHeader(http.StatusNotFound)
		_, err = w.Write([]byte("NOT FOUND"))
		return
	}

	err = app.db.Get(&emojiRecord, "SELECT id, site_id, emoji, count FROM emoji WHERE site_id=? AND emoji=?", urlId, emojiRunes.dbEncode())
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
		string(emojiRecord.Emoji.decodeDb()): emojiRecord.Count,
	}
	err = response.JSON(w, http.StatusOK, data)
	if err != nil {
		app.serverError(w, r, err)
	}
}

// Increment the count for a specific emoji by 1
func (app *application) createOne(w http.ResponseWriter, r *http.Request) {
	emojiRunes := make(emojiRunesT, 4)
	reader := io.LimitReader(r.Body, maxPayloadByteSize)
	encodedValue := make([]byte, maxPayloadByteSize)
	byteLength, err := reader.Read(encodedValue)

	// Form submissions are url encoded, so we need to decode them before getting the
	// key rune
	if r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
		escapedValue, err := url.QueryUnescape(string(encodedValue))
		if err != nil {
			app.serverError(w, r, err)
		}

		es := emojiStringT(escapedValue)
		emojiRunes, err = es.parseRunes()
		if err != nil {
			app.serverError(w, r, err)
		}

		// JSON has a specific structure {"emoji": "🌾"}
		// So, we need to convert it to this structure first. Then we parse it
	} else if r.Header.Get("Content-Type") == "application/json" {
		var request = struct {
			Emoji string `json:"emoji"`
		}{}
		err = json.Unmarshal(encodedValue[:byteLength], &request)
		if err != nil {
			app.serverError(w, r, err)
		}

		es := emojiStringT(request.Emoji)
		emojiRunes, err = es.parseRunes()
		if err != nil {
			app.serverError(w, r, err)
		}

		//	For all other requests, we try to decode the string and get the first rune.
	} else {
		es := emojiStringT(encodedValue)
		emojiRunes, err = es.parseRunes()
		if err != nil {
			app.serverError(w, r, err)
		}
	}
	if err != nil && err.Error() != "EOF" {
		app.serverError(w, r, err)
	}

	if emojiRunes[0] == 0 {
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
		err = app.db.Get(&emojiRecord, "SELECT id, site_id, emoji, count FROM emoji WHERE site_id=? AND emoji=?", urlId, emojiRunes.dbEncode())
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
		tx.MustExec("INSERT INTO emoji (site_id, emoji) VALUES (?, ?)", urlId, emojiRunes.dbEncode())
	} else {
		updateStmt, _ := tx.Prepare("UPDATE emoji SET count=? WHERE id=?")
		_, err = updateStmt.Exec(emojiRecord.Count+1, emojiRecord.Id)
		if err != nil {
			tx.Rollback()
		}
	}
	tx.Commit()

	if noEmojiRecord == true {
		w.WriteHeader(201)
	} else {
		w.WriteHeader(200)
	}

	app.logger.Info(fmt.Sprintf("%s -> %s reaction!", urlPathValue, string(emojiRune)))
	_, err = w.Write([]byte("OK"))
	if err != nil {
		app.serverError(w, r, err)
	}
}
