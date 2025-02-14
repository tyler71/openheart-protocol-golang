package main

import (
	"database/sql"
	"embed"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	emojiLib "github.com/dmolesUC/emoji"
	"github.com/rivo/uniseg"
	"io"
	"log"
	"net/http"
	"net/url"
	"openheart.tylery.com/internal/response"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

const maxPayloadByteSize = 64

type urlIdColumn int
type emojiTable struct {
	Id     int            `db:"id"`
	SiteId int            `db:"site_id"`
	Emoji  dbEncodedEmoji `db:"emoji"`
	Count  int            `db:"count"`
}
type dbEncodedEmoji string

func (e dbEncodedEmoji) Decode() string {
	splitStrings := strings.Split(string(e), "|")
	emojiRunes := make([]rune, len(splitStrings))
	for i := range splitStrings {
		parsedInt, _ := strconv.Atoi(splitStrings[i])
		emojiRunes[i] = rune(parsedInt)
	}
	return string(emojiRunes)
}

type emojiT struct {
	Bytes     []byte
	runes     []rune
	s         string
	DbEncoded string
}

// Return rendered string if not cached, render and cache otherwise
func (e emojiT) String() string {
	if e.s != "" {
		return e.s
	}
	if len(e.Bytes) == 0 {
		log.Println("Missing bytes for String output")
	}
	e.s = string(e.Bytes)
	return e.s
}

func (e emojiT) dbEncode() string {
	if e.DbEncoded != "" {
		return e.DbEncoded
	}
	runes, _ := e.parseRunes()
	var dbString = make([]string, len(runes))
	var count int
	for i := range runes {
		if runes[i] == 0 {
			break
		}
		a := int(runes[i])
		dbString[i] = strconv.Itoa(a)
		count++
	}
	if count > 1 {
		e.DbEncoded = strings.Join(dbString, "|")
	} else {
		e.DbEncoded = dbString[0]
	}
	return e.DbEncoded
}

func (e emojiT) parseRunes() ([]rune, error) {
	if e.runes != nil {
		return e.runes, nil
	}
	if len(e.Bytes) == 0 {
		log.Println("Missing bytes for parseRunes output")
	}
	emojiRunes := make([]rune, utf8.RuneCountInString(string(e.Bytes)))
	var count int
	for _, r := range string(e.Bytes) {
		emojiRunes[count] = r
		count++
	}
	if !emojiLib.IsEmoji(emojiRunes[0]) {
		return emojiRunes, errors.New("no emoji found")
	}
	e.runes = emojiRunes[:count]
	return emojiRunes[:count], nil
}

type inputUrl string

const hostnameRegex = `^[A-Za-z0-9][A-Za-z0-9-.]*\.\D{2,4}.*`

func (u inputUrl) hostname() (string, error) {
	m := regexp.MustCompile(hostnameRegex)
	result := m.FindStringSubmatch(string(u))
	if len(result) > 0 {
		return result[0], nil
	} else {
		return "", errors.New("no hostname found")
	}
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
	urlPathValue := inputUrl(r.PathValue("url"))

	// Due to a limitation in net/http routing, we cannot do / and wildcard /*
	// To get around this, if the urlPathValue is empty or the root document /
	// We escape into the home page and return. Otherwise, return the emoji count
	if urlPathValue == "" {
		app.homePage(w, r)
		return
	}
	parsedUrl, err := urlPathValue.hostname()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte("INVALID URL"))
		return
	}

	var urlId urlIdColumn
	var emojiRecords []emojiTable

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

// Returns emoji count for a specific url and emoji
//func (app *application) getOne(w http.ResponseWriter, r *http.Request) {
//	urlPathValue, emojiPathValue := inputUrl(r.PathValue("url")), emojiStringT(r.PathValue("emoji"))
//	parsedUrl, err := urlPathValue.hostname()
//	if err != nil {
//		w.WriteHeader(http.StatusBadRequest)
//		_, err = w.Write([]byte("INVALID URL"))
//		return
//	}
//	var urlId urlIdColumn
//	var emojiRecord emojiTable
//
//	emojiRunes, err := emojiPathValue.parseRunes()
//	if err != nil {
//		app.serverError(w, r, err)
//	}
//
//	// We look for the site id record. If none exists, we return 404
//	err = app.db.Get(&urlId, "SELECT id FROM site WHERE url=?", parsedUrl)
//	if err != nil && !errors.Is(err, sql.ErrNoRows) {
//		app.serverError(w, r, err)
//	}
//	if errors.Is(err, sql.ErrNoRows) {
//		w.WriteHeader(http.StatusNotFound)
//		_, err = w.Write([]byte("NOT FOUND"))
//		return
//	}
//
//	err = app.db.Get(&emojiRecord, "SELECT id, site_id, emoji, count FROM emoji WHERE site_id=? AND emoji=?", urlId, emojiRunes.dbEncode())
//	if err != nil && !errors.Is(err, sql.ErrNoRows) {
//		app.serverError(w, r, err)
//	}
//	if errors.Is(err, sql.ErrNoRows) {
//		w.WriteHeader(http.StatusNotFound)
//		_, err = w.Write([]byte("NOT FOUND"))
//		return
//	}
//
//	// On the happy path here, we have the record and return the count to the user
//	data := map[string]int{
//		string(emojiRecord.Emoji.decodeDb()): emojiRecord.Count,
//	}
//	w.Header().Set("Cache-Control", "max-age=30")
//	err = response.JSON(w, http.StatusOK, data)
//	if err != nil {
//		app.serverError(w, r, err)
//	}
//}

// Increment the count for a specific emoji by 1
func (app *application) createOne(w http.ResponseWriter, r *http.Request) {
	reader := io.LimitReader(r.Body, maxPayloadByteSize)
	encodedValue := make([]byte, maxPayloadByteSize)
	byteLength, err := reader.Read(encodedValue)
	var emoji emojiT

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
		var request = struct {
			Emoji string `json:"emoji"`
		}{}
		err = json.Unmarshal(encodedValue[:byteLength], &request)
		if err != nil {
			app.serverError(w, r, err)
		}

		encodedValue, _, _, _ = uniseg.Step([]byte(request.Emoji), -1)
		emoji.Bytes = encodedValue
		if err != nil {
			app.serverError(w, r, err)
		}

		//	For all other requests, we try to decode the string and get the first rune.
	} else {
		encodedValue, _, _, _ = uniseg.Step(encodedValue, -1)
		emoji.Bytes = encodedValue
	}

	// Let's see if the first rune is an emoji
	emojiRunes, err := emoji.parseRunes()
	if emojiRunes[0] == 0 || err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte("BAD REQUEST"))
		return
	}

	urlPathValue := inputUrl(r.PathValue("url"))
	parsedUrl, err := urlPathValue.hostname()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, err = w.Write([]byte("INVALID URL"))
		return
	}
	var urlId urlIdColumn

	var emojiRecord emojiTable

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
		err = app.db.Get(&emojiRecord, "SELECT id, site_id, emoji, count FROM emoji WHERE site_id=? AND emoji=?", urlId, emoji.dbEncode())
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
		tx.MustExec("INSERT INTO emoji (site_id, emoji) VALUES (?, ?)", urlId, emoji.dbEncode())
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
		err = tx.Get(&emojiRecord, "SELECT id, site_id, emoji, count FROM emoji WHERE site_id=? AND emoji=?", urlId, emoji.dbEncode())
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
