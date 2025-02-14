package request

import (
	"errors"
	emojiLib "github.com/dmolesUC/emoji"
	"log"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

type UrlIdColumn int
type EmojiTable struct {
	Id     int            `db:"id"`
	SiteId int            `db:"site_id"`
	Emoji  DbEncodedEmoji `db:"emoji"`
	Count  int            `db:"count"`
}
type DbEncodedEmoji string

func (e DbEncodedEmoji) Decode() string {
	splitStrings := strings.Split(string(e), "|")
	emojiRunes := make([]rune, len(splitStrings))
	for i := range splitStrings {
		parsedInt, _ := strconv.Atoi(splitStrings[i])
		emojiRunes[i] = rune(parsedInt)
	}
	return string(emojiRunes)
}

type EmojiT struct {
	Bytes     []byte
	runes     []rune
	s         string
	DbEncoded string
}

// Return rendered string if not cached, render and cache otherwise
func (e EmojiT) String() string {
	if e.s != "" {
		return e.s
	}
	if len(e.Bytes) == 0 {
		log.Println("Missing bytes for String output")
	}
	e.s = string(e.Bytes)
	return e.s
}

func (e EmojiT) DbEncode() string {
	if e.DbEncoded != "" {
		return e.DbEncoded
	}
	runes, _ := e.ParseRunes()
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

func (e EmojiT) ParseRunes() ([]rune, error) {
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

type InputUrl string

const hostnameRegex = `[A-Za-z0-9][\sA-Za-z0-9-.]*[\s/A-Za-z0-9-]+\.[a-z]+`

func (u InputUrl) Parse() (string, error) {
	m := regexp.MustCompile(hostnameRegex)
	escapedValue, _ := url.QueryUnescape(string(u))
	result := m.FindStringSubmatch(escapedValue)
	if len(result) > 0 {
		f := result[0]
		return f, nil
	} else {
		return "", errors.New("no hostname found")
	}
}
