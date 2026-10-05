package i18n

import (
	"errors"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	_ "looz.ws/typstify/i18n/translations"
)

// Define a Localizer type which stores the relevant locale ID (as used
// in our URLs) and a (deliberately unexported) message.Printer instance
// for the locale.
type Localizer struct {
	ID      string
	Name    string
	printer *message.Printer
}

// Initialize a slice which holds the initialized Localizer types for
// each of our supported locales.
var Locales = []Localizer{
	{
		// United States
		ID:      "en-us",
		Name:    "English",
		printer: message.NewPrinter(language.MustParse("en-US")),
	},

	{
		// Chinese
		ID:      "zh-cn",
		Name:    "中文",
		printer: message.NewPrinter(language.MustParse("zh-CN")),
	},

	{
		// Germany
		ID:      "de",
		Name:    "Deutschland",
		printer: message.NewPrinter(language.MustParse("de-DE")),
	},

	{
		// Vietnam
		ID:      "vi-vn",
		Name:    "Tiếng Việt",
		printer: message.NewPrinter(language.MustParse("vi-VN")),
	},
}

// The Get() function accepts a locale ID and returns the corresponding
// Localizer for that locale. If the locale ID is not supported then
// this returns `false` as the second return value.
func Get(id string) (Localizer, bool) {
	norm := strings.ToLower(strings.TrimSpace(id))
	for _, locale := range Locales {
		if norm == locale.ID {
			return locale, true
		}
	}

	return Localizer{printer: message.NewPrinter(language.MustParse("en-US"))}, false
}

func (l Localizer) Translate(key message.Reference, args ...interface{}) string {
	return l.printer.Sprintf(key, args...)
}

var defaultLocalizer Localizer

func SetLocale(id string) error {
	var found bool
	defaultLocalizer, found = Get(id)
	if !found {
		defaultLocalizer, _ = Get("en-us")
		return errors.New("Locales not supported: " + id)
	}
	return nil
}

func Translate(key message.Reference, args ...interface{}) string {
	return defaultLocalizer.Translate(key, args...)
}

func init() {
	defaultLocalizer, _ = Get("en-us")
}
