package foundation

import (
	"errors"
	"regexp"
	"strings"

	"github.com/weitecit/pkg/log"
	"github.com/weitecit/pkg/utils"
)

// type Localization struct {
// 	Language Language
// 	Repo     *i18n.Bundle
// }

// func NewLocalization(lang Language) (Localization, error) {

// 	var err error
// 	lang, err = lang.Validate()
// 	if err != nil {
// 		return Localization{}, errors.New("NewLocalization: " + err.Error())
// 	}

// 	bundle := i18n.NewBundle(language.Spanish)
// 	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

// 	lang, err = lang.Normalize()
// 	if err != nil {
// 		return Localization{}, err
// 	}

// 	file := utils.GetEnv("LANG_FOLDER") + lang.String() + ".json"

// 	_, err = bundle.LoadMessageFile(file)
// 	if err != nil {
// 		return Localization{}, err
// 	}

// 	return Localization{
// 		Language: lang,
// 		Repo:     bundle,
// 	}, nil
// }

// func (m Localization) Localize(text string, args ...string) (string, error) {

// 	language, err := m.Language.Normalize()
// 	if err != nil {
// 		return m.Language.String(), err
// 	}

// 	localizer := i18n.NewLocalizer(m.Repo, language.String())

// 	localized, err := localizer.Localize(&i18n.LocalizeConfig{
// 		MessageID: text,
// 	})
// 	if err != nil {
// 		log.Err(err)
// 		return text, err
// 	}

// 	results := strings.Split(localized, "|")
// 	if len(results) == 1 {
// 		return localized, nil
// 	}

// 	count := 1
// 	if len(args) > 0 {
// 		count = utils.StrToInt(args[0])
// 	}

// 	if count > 1 {
// 		return utils.Trim(results[1]), nil
// 	}

// 	return utils.Trim(results[0]), nil
// }

// // Concatenate strings from params function
// func (m Localization) ConcatenateStrings(params ...string) string {
// 	var result string
// 	for _, param := range params {
// 		localized, err := m.Localize(param)
// 		if err != nil {
// 			result += param
// 		} else {
// 			result += localized
// 		}
// 	}
// 	return result
// }

type Language string

var defaultLanguages = map[string]Language{
	"es": "es-ES",
}

func NewLanguage(code string) (Language, error) {

	code = strings.Replace(code, "_", "-", -1)
	language := Language(code)
	language, err := language.Validate()
	return language, err
}

func NewLanguageFromDataSourceName(fileName string) (Language, bool) {
	pattern := `([a-z]{2,3})(?:[-_]([a-zA-Z0-9]{1,8}))`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(fileName)

	if len(matches) >= 3 {
		languageString := matches[1] + "-" + matches[2]
		result, err := NewLanguage(languageString)
		if err == nil {
			return result, true
		}
	}

	return Language(""), false
}

func GetDefaultLanguages() map[string]Language {
	var defaultLanguages = map[string]Language{
		"es": "es-ES",
	}
	return defaultLanguages
}

func GetTranslatedLanguages() []Language {
	return []Language{
		"es-ES",
	}
}

func GetTemplatesLanguages() []Language {
	return []Language{
		"es-ES",
	}
}

func (m Language) Normalize() (Language, error) {

	if m == "" {
		return m, errors.New("Language.Normalize: empty language")
	}

	translatedLanguages := GetTranslatedLanguages()

	// if language is supported, return
	for _, lang := range translatedLanguages {
		if strings.EqualFold(lang.String(), m.String()) {
			return m, nil
		}
	}

	return m, errors.New("Language.Normalize: language " + m.String() + " not supported")
}

func (m Language) Validate() (Language, error) {

	defaultLanguage := Language(utils.GetEnv("WEITEC_LANGUAGE"))
	if defaultLanguage == "" {
		log.Err(errors.New("Language.Validate: ENV key WEITEC_LANGUAGE is empty"))
		defaultLanguage = Language("es-ES")
	}

	if m == "" {
		m = defaultLanguage
		return m, errors.New("Language.Validate: empty language")
	}

	code := m.String()

	if len(code) == 2 {
		defaultLanguages := GetDefaultLanguages()
		result := defaultLanguages[m.String()]
		if result != "" {
			m = Language(result)
			return m, nil
		}
	}

	if len(code) == 5 {
		code = code[:2] + "-" + strings.ToUpper(code[3:])
		m = Language(code)
	}

	matched, err := regexp.MatchString(`^[a-z]{2}-([A-Z]{2}|[0-9]{3})$`, m.String())
	if err != nil {
		return defaultLanguage, errors.New("Language.Validate: " + err.Error())
	}

	if !matched {
		return defaultLanguage, errors.New("Language.Validate: invalid language: " + m.String())
	}

	return m, nil
}

func (m Language) String() string {
	return string(m)
}
