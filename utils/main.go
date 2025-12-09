package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/saintfish/chardet"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const YYYY_MM_DD = "2006-01-02"
const YYYY_M_D = "2006-1-2"
const YYYY_MM_DD_HH = "2006-01-02 15:04:05 -0700 MST"

func DATE_FORMAT() []string {
	return []string{YYYY_MM_DD}
}
func DATE_TIME_FORMAT() []string {
	return []string{YYYY_MM_DD_HH}
}
func ALL_DATE_FORMATS() []string {
	return []string{
		"02/01/2006",
		"01/02/2006",
		"02/01/06",
		"01/02/06",
		"2006/02/01",
		"2006/01/02",
		"02-01-2006",
		"01-02-2006",
		"02-01-06",
		"01-02-06",
		"2006-02-01",
		"2006-01-02",
		"2/1/2006",
		"1/2/2006",
		"2/1/06",
		"1/2/06",
		"2006/2/1",
		"2006/1/2",
		"2-1-2006",
		"1-2-2006",
		"2-1-21",
		"1-2-21",
		"2006-2-1",
		"2006-1-2",
		"020106",
		"010206",
		"02012006",
		"01022006",
		"20060201",
		"20060102",
		"02 Jan",
		"Jan 02",
		"02 Jan 06",
		"Jan 02 06",
		"02 Jan 2006",
		"Jan 02 2006",
		"02/Jan/2006",
		"Jan/02/2006",
		"02/Jan/06",
		"Jan/02/06",
		"02-Jan-2006",
		"Jan-02-2006",
		"02-Jan-06",
		"Jan-02-06",
		"02.01.2006",
		"01.02.2006",
		"02.01.06",
		"01.02.06",
		"2006.01.02",
		"2006.02.01",
	}
}

type Enum string

type Dictionary struct {
	Key   string
	Value string
}

type OpType int

const (
	OpTypeAdd OpType = iota + 1
	OpTypeRemove
	OpTypeSwitch
)

func MakeOperationInArray(list []string, item string, opType OpType) []string {
	switch opType {
	case OpTypeAdd:
		// add if not exists
		list, _ = FindOrAppendStr(list, item)
	case OpTypeRemove:
		list, _ = RemoveStringInArray(list, item)
	case OpTypeSwitch:
		list, exists := RemoveStringInArray(list, item)
		if !exists {
			list = append(list, item)
		}
	default:
		return list
	}
	return list
}

func RemoveLastItem(items []string) []string {
	if items == nil {
		return []string{}
	}
	if len(items) == 0 {
		return items
	}
	return items[:len(items)-1]
}

func Bool(b interface{}) bool {
	switch b.(type) {
	case bool:
		return b.(bool)
	default:
		return false
	}
}
func NewID() *primitive.ObjectID {
	id := primitive.NewObjectID()
	return &id
}

func NewIDHex() string {
	newID := primitive.NewObjectID()
	return newID.Hex()
}

func Int64(number int64) *int64 {
	return &number
}

func Now() *time.Time {
	t := time.Now()
	return &t
}

func Date(year int, month int, day int) *time.Time {
	m := time.Month(month)
	t := time.Date(year, m, day, 0, 0, 0, 0, time.UTC)
	return &t
}

func WeekNumber(date *time.Time) int {
	_, week := date.ISOWeek()
	return week
}

func GetFirstDateOfTheWeek(t *time.Time) *time.Time {
	weekday := t.Weekday()
	daysToMonday := int(weekday - time.Monday)
	if daysToMonday < 0 {
		daysToMonday += 7
	}
	firstDay := t.AddDate(0, 0, -daysToMonday).Truncate(24 * time.Hour)
	return &firstDay
}

func GetToday() *time.Time {
	today := time.Now().Truncate(24 * time.Hour)
	return &today
}
func Normalize(str string) string {
	str = strings.ReplaceAll(str, " ", "")
	str = strings.ToLower(str)
	str = RemoveAccents(str)
	str = strings.TrimFunc(str, func(r rune) bool {
		return !unicode.IsGraphic(r)
	})
	return str
}

func NormalizeKey(str string) string {
	// Pasar a minúsculas
	s := strings.ToLower(str)

	// Separar acentos (NFD) y eliminar marcas de acento
	s = norm.NFD.String(s)
	s = strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) {
			return -1
		}
		return r
	}, s)

	// Reemplazar espacios por guion bajo
	s = strings.ReplaceAll(s, " ", "_")

	// Quitar caracteres que no sean alfanuméricos o guion bajo
	re := regexp.MustCompile(`[^\w_]`)
	s = re.ReplaceAllString(s, "")

	// Normalizar guiones bajos múltiples
	reMultiUnderscore := regexp.MustCompile(`_+`)
	s = reMultiUnderscore.ReplaceAllString(s, "_")

	// Quitar guion bajo inicial o final
	s = strings.Trim(s, "_")

	return s
}

func NormalizeFile(str string) string {
	str = Normalize(str)
	str = strings.ReplaceAll(str, "/", " ")
	return str
}

func NormalizeForTag(str string) string {
	value := strings.TrimSpace(str)
	key := Normalize(value)
	str = RemovePunctuation(key)

	return str
}

func Replace(str string, from string, to string) string {
	return strings.ReplaceAll(str, from, to)
}

func RemovePunctuation(str string) string {
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		return "RemovePunctuation error"
	}
	processedString := reg.ReplaceAllString(str, "")
	return processedString
}

func NormalizeToAlphanumeric(str string) string {
	str = strings.ReplaceAll(str, " ", "")
	str = strings.ToLower(str)
	str = RemoveAccents(str)
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		return "NormalizeToAlphanumeric error"
	}
	processedString := reg.ReplaceAllString(str, "")
	return processedString
}

func GetSuffixUntilNonAlphanumeric(str string) string {
	// Define the set of characters to search for
	chars := ".,-_/"

	// Find the index of the last occurrence of any character in the set
	index := strings.LastIndexAny(str, chars)

	// If no character is found, return the original string
	if index == -1 {
		return str
	}

	// Return the substring from the last occurrence to the end of the string
	return str[index+1:]
}

func NormalizeToAlphanumericNotLower(str string) string {
	str = strings.ReplaceAll(str, " ", "")
	str = RemoveAccents(str)
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		return "NormalizeToAlphanumericNotLower error"
	}
	processedString := reg.ReplaceAllString(str, "")
	return processedString
}

func RemoveAccents(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, s)
	return result
}

func StrToInt64(str string) (int64, error) {
	if str == "" {
		return 0, nil
	}
	result, err := strconv.Atoi(str)
	if err != nil {
		return 0, err
	}
	return int64(result), nil
}

func Int64ToStr(n int64) string {
	return strconv.FormatInt(n, 10)
}

func IntToStr(n int) string {
	return strconv.Itoa(n)
}

func PositionAlphabetToLetter(position int) (string, error) {
	if position < 1 {
		return "", fmt.Errorf("position must be a positive integer")
	}

	if position < 27 {
		firstLetter := string('A' + rune(position-1))
		return firstLetter, nil
	}

	firstPos := (position-1)/26 - 1
	secondPos := (position - 1) % 26

	firstLetter := 'A' + rune(firstPos)
	secondLetter := 'A' + rune(secondPos)

	doubleLetter := fmt.Sprintf("%c%c", firstLetter, secondLetter)
	return doubleLetter, nil
}

func Float64ToInt(f float64) int {
	return int(f)
}

func AddDecimalStr(n string) string {
	if len(n) > 2 {
		return n[:len(n)-2] + "." + n[len(n)-2:]
	}
	if len(n) == 2 {
		return "0." + n
	}
	if len(n) == 1 {
		return "0.0" + n
	}
	return "0"
}

func IntIDToStr(id int) (string, error) {
	if id < 1 {
		return "", fmt.Errorf("Utils.IntIDToStr: id must be grater to 1")
	}
	return strconv.Itoa(id), nil
}

func StrToIntID(id string) (int, error) {
	if id == "" {
		return 0, fmt.Errorf("Utils.StrToIntID: empty string")
	}
	result, err := strconv.Atoi(id)
	if err != nil {
		return result, err
	}
	if result < 1 {
		return 0, fmt.Errorf("Utils.StrToIntID: id must be grater to 1")
	}
	return result, nil
}

func HasValidStrIntID(id string) bool {
	_, err := StrToIntID(id)
	return err == nil
}

func Uint64ToStr(n uint64) string {
	return strconv.FormatUint(n, 10)
}

func StrToByte(str string) (byte, error) {
	if str == "" {
		return 0, nil
	}

	result, err := strconv.Atoi(str)
	if err != nil {

		return 0, err
	}
	return byte(result), nil
}

func IntToByte(integer int) byte {
	return byte(integer)
}

func StrToBool(str string) (bool, error) {
	result, err := strconv.ParseBool(str)
	if err != nil {
		return false, err
	}
	return result, nil
}

func Abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func IntMillisecondsToTime(milliseconds int) time.Time {
	return time.Unix(0, int64(milliseconds)*int64(time.Millisecond))
}

func StrMillisecondsToDate(milliseconds string) *time.Time {
	m, err := StrToInt64(milliseconds)
	if err != nil {
		return nil
	}

	result := time.UnixMilli(m)
	if result.IsZero() {
		return nil
	}

	date := time.Date(result.Year(), result.Month(), result.Day(), 0, 0, 0, 0, time.UTC)

	return &date
}

func TimeToDate(t time.Time) time.Time {
	date := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	return date
}

func TimeToPointerDate(t time.Time) *time.Time {
	return &t
}

func GetCharSet(file *os.File, codification string) (reader *csv.Reader, newCodification string) {
	newCodification = codification
	part := make([]byte, 1000)
	_, err := file.Read(part)
	if err != nil {
		return csv.NewReader(file), newCodification
	}
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return csv.NewReader(file), newCodification
	}
	detector := chardet.NewTextDetector()
	bestChar, err := detector.DetectBest(part)
	if err != nil {
		return csv.NewReader(file), newCodification
	}
	if codification == "" {
		if strings.Contains(bestChar.Charset, "8859-1") {
			newCodification = "ISO-8859-1"
		}
		if strings.Contains(bestChar.Charset, "UTF") {
			newCodification = "UTF-8"
		}
	}

	var csvr *csv.Reader
	switch newCodification {
	case "ISO-8859-1":
		csvr = csv.NewReader(charmap.ISO8859_1.NewDecoder().Reader(file))
	case "UTF-8":
		csvr = csv.NewReader(file)
	default:
		err = errors.New("Utils.GetCharSet: charset not found.. " + bestChar.Charset)
		csvr = csv.NewReader(file)
	}

	csvr.FieldsPerRecord = -1
	return csvr, newCodification
}

func StringInArray(a string, list []string) bool {

	if len(list) == 0 {
		return false
	}

	if a == "" {
		return false
	}

	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func IntInArray(a int, list []int) bool {

	if len(list) == 0 {
		return false
	}

	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func GetHigherValue(list []int) int {

	higherValue := 0
	if len(list) == 0 {
		return 0
	}

	for _, b := range list {
		if b > higherValue {
			higherValue = b
		}
	}
	return higherValue
}

func GetLowerValue(list []int) int {

	if len(list) == 0 {
		return 0
	}
	lowerValue := list[0]

	for _, b := range list {
		if b < lowerValue {
			lowerValue = b
		}
	}
	return lowerValue
}

func IDInArray(a *primitive.ObjectID, list []*primitive.ObjectID) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func RemoveStringsFromArray(list []string, words ...string) ([]string, []int) {
	if list == nil || len(list) == 0 {
		return []string{}, []int{}
	}

	indexes := []int{}

	for _, word := range words {
		for i, item := range list {
			if item == word {
				indexes = append(indexes, i)
				list = append((list)[:i], (list)[i+1:]...)
				continue
			}
		}
	}
	return list, indexes
}

func RemoveStringsFromArrayByIndex(list []string, indexes []int) []string {
	if list == nil || len(list) == 0 {
		return []string{}
	}

	sort.Ints(indexes)
	for i := len(indexes) - 1; i >= 0; i-- {
		list = append((list)[:indexes[i]], (list)[indexes[i]+1:]...)
	}
	return list
}

func AddStringsInArray(list []string, words ...string) []string {
	for _, word := range words {
		if word == "" {
			continue
		}
		if !StringInArray(word, list) {
			list = append(list, word)
		}
	}
	return list
}

func RemoveStringInArray(list []string, word string) ([]string, bool) {

	for i, item := range list {
		if item == word {
			list = append((list)[:i], (list)[i+1:]...)
			return list, true
		}
	}
	return list, false
}

func ReplaceStringsInArray(list []string, old string, new string) []string {
	for i, item := range list {
		if item == old {
			list[i] = new
		}
	}
	return list
}

func ArrayContentStr(list []string, str string) bool {
	for _, item := range list {
		if item == str {
			return true
		}
	}
	return false
}

func ReplaceStringsInArrayIDs(list []string, oldID *primitive.ObjectID, newID *primitive.ObjectID) []string {
	old := oldID.Hex()
	for i, item := range list {
		if item == old {
			list[i] = newID.Hex()
		}
	}
	return list
}

func SameArray(list1 []string, list2 []string) bool {
	if len(list1) != len(list2) {
		return false
	}
	for _, item1 := range list1 {
		if !StringInArray(item1, list2) {
			return false
		}
	}
	return true
}

func AddDateInArray(list []time.Time, dates ...time.Time) ([]time.Time, bool) {

	changed := false

	for _, d := range dates {
		d = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
		if !TimeInArray(d, list) {
			list = append(list, d)
			changed = true
		}
	}
	return list, changed
}

func AddTimeInArray(list []time.Time, times ...time.Time) ([]time.Time, bool) {

	changed := false

	for _, t := range times {
		if !TimeInArray(t, list) {
			list = append(list, t)
			changed = true
		}
	}
	return list, changed
}

func RemoveDateInArray(list []time.Time, dates ...time.Time) ([]time.Time, bool) {

	changed := false

	for _, d := range dates {
		d = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
		for i, item := range list {
			item = time.Date(item.Year(), item.Month(), item.Day(), 0, 0, 0, 0, time.UTC)
			if item == d {
				list = append((list)[:i], (list)[i+1:]...)
				changed = true
			}
		}
	}
	return list, changed
}

func RemoveTimeInArray(list []time.Time, times ...time.Time) ([]time.Time, bool) {

	changed := false

	for _, t := range times {
		for i, item := range list {
			if item == t {
				list = append((list)[:i], (list)[i+1:]...)
				changed = true
			}
		}
	}
	return list, changed
}

func TimeInArray(a time.Time, list []time.Time) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func FindOrAppendInt(list []int, item int) []int {
	for _, x := range list {
		if x == item {
			return list
		}
	}
	return append(list, item)
}

func FindOrAppendStrRaw(list []string, item string) []string {
	for _, x := range list {
		if x == item {
			return list
		}
	}
	return append(list, item)
}

func FindOrAppendStr(list []string, item string) ([]string, error) {
	if item == "" {
		return list, errors.New("Utils.FindOrAppendStr: empty string")
	}
	for _, x := range list {
		if x == item {
			return list, errors.New("Utils.FindOrAppendStr: string already exists")
		}
	}
	return append(list, item), nil
}

func FindOrAppendID(list []*primitive.ObjectID, item *primitive.ObjectID) ([]*primitive.ObjectID, error) {
	if item == nil {
		return list, nil
	}
	for _, x := range list {
		if x == nil {
			continue
		}
		if HaveSameIDs(x, item) {
			return list, errors.New("Utils.FindOrAppendID: ID already exists")
		}
	}
	return append(list, item), nil
}

func RemoveID(list []*primitive.ObjectID, item *primitive.ObjectID) ([]*primitive.ObjectID, error) {
	index := -1
	for i, x := range list {
		if x == nil {
			continue
		}
		if HaveSameIDs(x, item) {
			index = i
		}
	}

	if index == -1 {
		return list, errors.New("Utils.RemoveID: id not found")
	}

	result := list[:index]
	if len(list) == len(result)+1 {
		return result, nil
	}

	result = append(result, list[index+1:]...)

	return result, nil
}

func HaveSameIDs(id1, id2 *primitive.ObjectID) bool {
	if id1 == nil || id2 == nil {
		return false
	}

	return id1.Hex() == id2.Hex()
}
func HaveSameStrIDs(id1, id2 string) bool {
	if id1 == "" || id2 == "" {
		return false
	}

	return id1 == id2
}

func FindOrAppendStringsArray(list []string, items []string) []string {
	for _, item := range items {
		list = FindOrAppendStrRaw(list, item)
	}
	return list
}

func NameFromPath(path string) string {
	name := filepath.Base(path)
	extension := filepath.Ext(name)
	return name[0 : len(name)-len(extension)]
}

// TODO: realmente va en Models o en UTILS?
var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	snake += "s"
	return strings.ToLower(snake)
}

func StringToDataType(text string, dataType string, needReformatDate bool, params ...string) (value interface{}, err error) {
	text = strings.TrimSpace(text)
	switch dataType {
	case "string":
		value = text
	case "bool":
		value, err = StringToBoolean(text)
	case "int":
		value = StrToInt(text)
	case "uint64":
		value, err = StrToUint64(text)
	case "float64":
		value, err = StringToFloat64(text)
	case "time.Time":
		value, err = StringToTime(text, needReformatDate, params...)
	case "*time.Time":
		value, err = StringToPointerTime(text, needReformatDate, params...)
	case "money":
		value, err = StringToMoney(text)
	case "percent":
		value, err = StringToPercent(text)
	case "array.String":
		value = StringToArrayString(text)
	case "array.Array.String":
		value = StringToArrayArrayString(text)
	default:
		err = errors.New("ConvertStringToType: invalid data type")
	}
	return
}

func String(value interface{}) string {
	switch value.(type) {
	case string:
		return value.(string)
	default:
		return ""
	}
}

func IsIntefaceNil(inter interface{}) bool {
	return reflect.ValueOf(inter).IsNil()
}

func StrToDate(text string, dateFormat []string) (value time.Time, err error) {

	for _, layout := range dateFormat {
		if layout != "" {
			x, err := time.Parse(layout, text)
			if err == nil {
				return x, nil
			}
		}
	}
	x, err := StringToTime(text, false)
	if err != nil {
		return time.Time{}, errors.New("ImportedDataSketch.GetDate: invalid string date provided")
	}

	return *x, nil
}

// If format ir empty, it will use the default format "2006-01-02"
func DateToStr(date *time.Time, format string) string {

	if date == nil {
		return ""
	}

	if format == "" {
		format = YYYY_MM_DD
	}
	return date.Format(format)
}

func Int64ToStrWithDecimals(value int64, decimals int) string {
	if decimals < 0 {
		decimals = 0
	}
	if value == 0 {
		return fmt.Sprintf("%."+strconv.Itoa(decimals)+"f", float64(value))
	}
	return fmt.Sprintf("%."+strconv.Itoa(decimals)+"f", float64(value)/math.Pow10(decimals))
}

func StrArrayToIDArray(strArray []string) ([]*primitive.ObjectID, error) {
	var result []*primitive.ObjectID
	for _, str := range strArray {
		id, err := primitive.ObjectIDFromHex(str)
		if err != nil {
			return nil, err
		}
		result = append(result, &id)
	}
	return result, nil
}

func IDArrayToStrArray(idArray []*primitive.ObjectID) []string {
	var result []string
	for _, id := range idArray {
		result = append(result, id.Hex())
	}
	return result
}

func DateToMilliseconds(date *time.Time) string {
	if date == nil {
		return "0"
	}
	return strconv.FormatInt(date.UnixNano()/int64(time.Millisecond), 10)
}

// 2023-01-22T00:00:00Z
func MillisecondsToTimestamp(milliseconds string) (time.Time, error) {
	if milliseconds == "" {
		return time.Time{}, errors.New("Utils.MillisecondsToTimestamp: invalid millisecond")
	}
	intMilli := StrToInt(milliseconds)
	if intMilli == 0 {
		return time.Time{}, errors.New("Utils.MillisecondsToTimestamp: invalid millisecond")
	}

	ts := time.Unix(0, int64(intMilli)*int64(time.Millisecond)).UTC()
	return ts, nil
}

func StrToInt(text string) int {
	if text == "" {
		return 0
	}
	value, err := strconv.Atoi(text)
	if err != nil {
		return 0
	}
	return value
}

func StrToInt64Raw(text string) int64 {
	if text == "" {
		return 0
	}
	value, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return 0
	}
	return value
}

func StrToUint64(text string) (value uint64, err error) {
	if text == "" {
		return 0, nil
	}
	value = uint64(0)
	value, err = strconv.ParseUint(text, 10, 64)
	if err != nil {
		value = uint64(0)
	}
	return
}

func decodeFloatString(text string) string {
	posPeriod := strings.LastIndex(text, ".")
	posComma := strings.LastIndex(text, ",")
	if posPeriod == -1 && posComma == -1 {
		return text
	}
	decimalSeparator := "."
	if posComma > posPeriod {
		decimalSeparator = ","
	}

	// remove thousands separator
	if decimalSeparator == "." {
		text = strings.ReplaceAll(text, ",", "")
	} else {
		text = strings.ReplaceAll(text, ".", "")
		text = strings.ReplaceAll(text, ",", ".")
	}

	// if no more than 2 digits return
	posSeparator := strings.LastIndex(text, decimalSeparator)
	if len(text)-posSeparator <= 3 {
		return text
	}

	// round to 2 digits
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return text
	}

	value = math.Round(value*100) / 100
	text = strconv.FormatFloat(value, 'f', -1, 64)

	return text

}

func StringToFloat64(text string) (value float64, err error) {
	value = 0.00
	if text == "" {
		return
	}
	text = decodeFloatString(text)
	// return max 2 decimals
	value, err = strconv.ParseFloat(text, 64)
	if err != nil {
		value = 0.00
	}

	return
}

func StringToTimeWithLayout(text string, needReformatDate bool, params ...string) (*time.Time, error) {
	if len(params) == 0 {
		return nil, errors.New("StringToTimeWithLayout: no layouts")
	}

	for _, layout := range params {

		if layout == "" {
			return nil, errors.New("StringToTimeWithLayout: empty layout")
		}

		date, err := time.Parse(layout, text)
		if err == nil {
			return &date, nil
		}

		if !needReformatDate {
			continue
		}

		formatedText, err := ReformatDate(text)
		if err != nil {
			return nil, err
		}

		date, err = time.Parse(layout, formatedText)
		if err == nil {
			return &date, nil
		}
		continue
	}

	return nil, errors.New("StringToTimeWithLayout: error in conversion of " + text)

}

func ReformatDate(value string) (string, error) {
	if value == "" {
		return value, errors.New("Main.Reformat: empty date")
	}
	for _, x := range value {
		if unicode.IsLetter(x) {
			value = TranslateMonthsToEnglish(value)
			return value, nil
		}
	}
	if len(value) < 5 {
		return value, errors.New("Main.Reformat: invalid date")
	}
	if len(value) == 5 {
		return ("0" + value), nil
	}
	no_separators := true
	for i := 0; i < len(value); i++ {
		_, err := strconv.Atoi(string(value[i]))
		if err != nil {
			no_separators = false
			break
		}
	}
	if no_separators {
		return value, nil
	}
	starts_with_year := true
	for i := 0; i < 4; i++ {
		_, err := strconv.Atoi(string(value[i]))
		if err != nil {
			starts_with_year = false
			break
		}
	}
	if !starts_with_year {
		_, err := strconv.Atoi(string(value[1]))
		if err != nil {
			value = "0" + value
		}
		_, err = strconv.Atoi(string(value[4]))
		if err != nil {
			value = value[0:3] + "0" + value[3:(len(value))]
		}
	} else {
		if len(value) < 8 {
			return value, errors.New("Main.Reformat: invalid date")
		}
		_, err := strconv.Atoi(string(value[6]))
		if err != nil {
			value = value[0:5] + "0" + value[5:(len(value))]
		}
		if len(value) < 9 {
			return value, errors.New("Main.Reformat: invalid date")
		}
		if len(value) == 9 {
			value = value[0:8] + "0" + value[8:9]
		}
	}
	return value, nil
}

func TranslateMonthsToEnglish(date string) string {
	spanish := []string{"Ene", "Feb", "Mar", "Abr", "May", "Jun", "Jul", "Ago", "Sep", "Oct", "Nov", "Dic"}
	portuguese := []string{"Jan", "Fev", "Mar", "Abr", "Mai", "Jun", "Jul", "Ago", "Set", "Out", "Nov", "Dez"}
	english := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	for i := range spanish {
		date = strings.Replace(date, spanish[i], english[i], -1)
	}
	for i := range spanish {
		date = strings.Replace(date, portuguese[i], english[i], -1)
	}
	return date
}

func StringToPointerTime(text string, needReformatDate bool, params ...string) (**time.Time, error) {
	if text == "" {
		return nil, errors.New("StringToPointerTime: empty string")
	}
	result, err := StringToTime(text, needReformatDate, params...)
	if err != nil {
		return nil, err
	}
	return &result, nil

}

// get layouts function
func GetLayouts(language string) []string {
	return ALL_DATE_FORMATS()
}

// find layouts function
func FindLayouts(text string, language string, totalResultLayouts int) []string {
	layouts := GetLayouts(language)
	result := []string{}
	text = Trim(text)
	for _, layout := range layouts {
		_, err := time.Parse(layout, text)
		if err == nil {
			result = append(result, layout)
		}
		if totalResultLayouts != 0 && len(result) == totalResultLayouts {
			return result
		}
	}
	return result

}

func Trim(str string) string {
	return strings.TrimSpace(str)
}

func StringToTime(text string, needReformatDate bool, params ...string) (*time.Time, error) {
	// Si se añade algún layout añadir también en Infrastructure Service - LoadConfig

	if text == "" {
		return nil, errors.New("StringToTime: empty string")
	}

	// return year and first month and first day
	if len(text) == 2 {
		year := time.Now().Year()
		year = year - (year % 100) + StrToInt(text)
		text = "01/01/" + IntToStr(year)
	}

	if len(text) == 4 {
		text = "01/01/" + text
	}

	result, err := StringToTimeWithLayout(text, needReformatDate, params...)
	if err == nil {
		return result, nil
	}

	if len(text) > 10 {
		text = text[0:10]
	}

	layouts := FindLayouts(text, "", 1)
	if len(layouts) == 0 {
		err = errors.New("StringToTime: not matching layout for " + text)
		return nil, err
	}

	response, err := time.Parse(layouts[0], text)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func StringToBoolean(text string) (value bool, err error) {
	value = false
	var slice = [11]string{"True", "true", "Yes", "yes", "Sí", "sí", "Si", "si", "x", "X", "1"}
	for _, item := range slice {
		if item == text {
			return true, nil
		}
	}
	return false, nil
}

func StringToArrayString(text string) (value []string) {
	value = []string{}
	if text == "" {
		return
	}
	// no podemos quitar los puntos porque se usan en algunas cuentas
	// text = strings.ReplaceAll(text, ".", ",")
	text = strings.ReplaceAll(text, " ", "")
	value = strings.Split(text, ",")
	value = RemoveEmptyStrings(value)
	return
}

func StringToArrayArrayString(text string) (value [][]string) {
	value = [][]string{}
	if text == "" {
		return
	}
	text = strings.ReplaceAll(text, " ", "")
	for _, v := range strings.Split(text, "/") {
		value = append(value, strings.Split(v, ","))
	}

	return
}

func ArrayToString(array []string) string {
	return strings.Join(array, ",")
}

func RemoveEmptyStrings(arr []string) []string {
	var result []string
	for _, str := range arr {
		if str != "" {
			result = append(result, str)
		}
	}
	return result
}

func CompareStringArrays(array1, array2 []string) bool {
	if len(array1) != len(array2) {
		return false
	}
	for _, item1 := range array1 {
		if !StringInArray(item1, array2) {
			return false
		}
	}
	return true
}

func InsertObjectInArray(array interface{}, object interface{}, index int) {
	// Handle nil array
	if array == nil {
		return
	}

	// Use reflection to handle any slice type
	v := reflect.ValueOf(array)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Slice {
		return // Not a pointer to a slice
	}

	slice := v.Elem()

	if index > slice.Len() {
		return // Index out of bounds
	}

	if index < 0 {
		index = slice.Len()
	}

	// Create a new element with the same type as the slice elements
	elemType := slice.Type().Elem()
	newElem := reflect.ValueOf(object)
	if !newElem.Type().AssignableTo(elemType) {
		return // Type mismatch
	}

	// Grow the slice by one element
	slice.Set(reflect.Append(slice, reflect.Zero(elemType)))

	// Shift elements to make room for the new element
	reflect.Copy(slice.Slice(index+1, slice.Len()), slice.Slice(index, slice.Len()-1))

	// Set the new element at the specified index
	slice.Index(index).Set(newElem)
}

func AddObjectInArray(array interface{}, object interface{}) {
	InsertObjectInArray(array, object, -1)
}

func StringToMoney(text string) (int64, error) {
	m := int64(0)
	if text == "" {
		return 0, nil
	}

	text = RemoveMoneySimbolsFromStr(text)
	text = strings.ReplaceAll(text, " ", "")
	if text == "-" {
		return 0, nil
	}

	text = NormalizeDecimals(text, 2)

	text = strings.Replace(text, ".", "", 1)

	m, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return m, err
	}

	return m, nil
}

func RemoveMoneySimbolsFromStr(text string) string {

	symbols := []string{"$", "€"}
	for _, symbol := range symbols {
		text = strings.ReplaceAll(text, symbol, "")
	}

	return text
}

func RemovePercentSimbolsFromStr(text string) string {

	symbols := []string{"$", "€", "£", "¥", "₱", "₹", "₽", "C$", "₡"}
	for _, symbol := range symbols {
		text = strings.ReplaceAll(text, symbol, "")
	}

	return text
}

func NormalizeDecimals(text string, decimals int) string {
	text = NormalizeCommaDot(text)
	value, err := ParseFloatScientificNotation(text)
	if err != nil {
		return text
	}
	decimalRound := math.Pow(10, float64(decimals))
	value = math.Round(value*decimalRound) / decimalRound
	formatString := fmt.Sprintf("%%.%df", decimals)
	return fmt.Sprintf(formatString, value)
	//return fmt.Sprintf("%.2f", value)
}

func NormalizeCommaDot(text string) string {
	numberCommas := strings.Count(text, ",")
	numberDots := strings.Count(text, ".")
	commaIndex := strings.Index(text, ",")
	dotIndex := strings.Index(text, ".")

	if numberCommas != 0 && numberDots != 0 {
		if numberCommas > numberDots {
			text = strings.Replace(text, ",", "", -1)
		}
		if numberDots > numberCommas {
			text = strings.Replace(text, ".", "", -1)
		}
		if numberCommas == numberDots {
			if commaIndex < dotIndex {
				text = strings.Replace(text, ",", "", -1)
			} else {
				text = strings.Replace(text, ".", "", -1)
			}
		}
	}

	if (numberCommas == 0 && numberDots > 1) || (numberDots == 0 && numberCommas > 1) {
		text = strings.Replace(text, ",", "", -1)
		text = strings.Replace(text, ".", "", -1)
	}

	text = strings.Replace(text, ",", ".", -1)

	return text

}

func ParseFloatScientificNotation(str string) (float64, error) {
	val, err := strconv.ParseFloat(str, 64)
	if err == nil {
		return val, nil
	}

	//Some number may be seperated by comma, for example, 23,120,123, so remove the comma firstly
	str = strings.Replace(str, ",", "", -1)

	//Some number is specifed in scientific notation
	pos := strings.IndexAny(str, "eE")
	if pos < 0 {
		return strconv.ParseFloat(str, 64)
	}

	var baseVal float64
	var expVal int64

	baseStr := str[0:pos]
	baseVal, err = strconv.ParseFloat(baseStr, 64)
	if err != nil {
		return 0, err
	}

	expStr := str[(pos + 1):]
	expVal, err = strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		return 0, err
	}

	return baseVal * math.Pow10(int(expVal)), nil
}
func StringToPercent(text string) (int64, error) {
	m := int64(0)
	if text == "" {
		return 0, nil
	}

	text = strings.ReplaceAll(text, "%", "")
	text = strings.ReplaceAll(text, " ", "")
	if text == "-" {
		return 0, nil
	}

	text = NormalizeDecimals(text, 4)
	text = strings.Replace(text, ".", "", 1)

	m, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return m, err
	}

	return m, nil
}

// TODO: ver monedas que no son utf-8

func ContainsStr(phrase string, value string) bool {
	return strings.Contains(phrase, value)
}

func ContainsStrInList(list []string, value string) bool {
	for _, text := range list {
		if strings.Contains(value, text) {
			return true
		}
	}
	return false
}

func NumberIn(vector []int, value int) bool {
	for _, x := range vector {
		if x == value {
			return true
		}
	}
	return false
}

func WordIn(vector []string, value string) bool {
	for _, x := range vector {
		if x == value {
			return true
		}
	}
	return false
}

func IsEmptyStr(str string) bool {
	str = strings.TrimSpace(str)
	if str == "" {
		return true
	}

	return false
}

func ContainsNumbersAndLetters(s string) bool {
	// Define the regular expression pattern to match any string that contains both letters and numbers
	pattern := `[a-zA-Z].*[0-9]|[0-9].*[a-zA-Z]`

	// Compile the regular expression pattern
	regex, _ := regexp.Compile(pattern)

	// Use the regular expression to match against the input string
	return regex.MatchString(s)
}

func Matches(row []string, items []string) bool {
	for _, item := range items {
		for _, x := range row {
			if x == item {
				return true
			}
		}
	}
	return false
}

func DeleteEmpty(row []string) []string {
	notEmpty := []string{}
	for _, x := range row {
		if x != "" {
			notEmpty = append(notEmpty, x)
		}
	}
	return notEmpty
}

func MinInt(x int, y int) int {
	if y < x {
		return y
	}
	return x
}

func MaxFloat(x float64, y float64) float64 {
	if y < x {
		return x
	}
	return y
}

func DateIsZero(date time.Time) bool {

	if date.Year() == 1 {
		return true
	}

	return false
}

func DateIsNilOrZero(date *time.Time) bool {
	if date == nil {
		return true
	}
	if date.Year() == 1 {
		return true
	}
	return false
}

func DateIsValid(date *time.Time) bool {
	valid := !DateIsNilOrZero(date)
	if !valid {
		return false
	}
	return date.Year() > 1 && date.Year() < 10000
}

func Max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

func Max64(x, y int64) int64 {
	if x < y {
		return y
	}
	return x
}

func Min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

func CompareDatesFromDatetimes(time1 time.Time, time2 time.Time) bool {
	y1, m1, d1 := time1.Date()
	y2, m2, d2 := time2.Date()
	if y1 != y2 {
		return false
	}
	if m1 != m2 {
		return false
	}
	if d1 != d2 {
		return false
	}
	return true
}

func CompareStrToDate(date *time.Time, strDate string) bool {
	if date == nil {
		return false
	}
	if len(strDate) > 10 {
		strDate = strDate[:10]
	}
	convertedDate, err := time.Parse(YYYY_MM_DD, strDate)
	if err != nil {
		return false
	}
	return CompareDatesFromDatetimes(*date, convertedDate)
}

func TwoDigits(value int) string {
	if value < 10 {
		return "0" + fmt.Sprint(value)
	}
	return fmt.Sprint(value)
}

func ZeroDate() *time.Time {
	date := time.Time{}
	return &date
}

func DateFromYear(year int) *time.Time {
	date := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	return &date
}

func ContainNumbers(value string) bool {
	for _, x := range value {
		_, err := strconv.Atoi(string(x))
		if err == nil {
			return true
		}
	}
	return false
}

func CloneID(id *primitive.ObjectID) *primitive.ObjectID {
	if id == nil {
		return nil
	}
	return GetObjectIdFromStringRaw(id.Hex())
}

func GetObjectIdFromString(id string) (*primitive.ObjectID, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("GetObjectIdFromString: id is empty")
	}
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return &objID, errors.New("GetObjectIdFromString: id is not valid")
	}
	return &objID, nil
}

func GetObjectIdFromStringRaw(id string) *primitive.ObjectID {
	id = strings.TrimSpace(id)
	objID, _ := primitive.ObjectIDFromHex(id)
	return &objID
}

func GetObjectIdsFromStringRawArray(ids []string) []*primitive.ObjectID {
	objIDs := []*primitive.ObjectID{}
	for _, id := range ids {
		realID, err := GetObjectIdFromString(id)
		if err == nil {
			objIDs = append(objIDs, realID)
		}
	}
	return objIDs
}

func GetObjectIdsFromStringArray(ids []string) ([]*primitive.ObjectID, error) {
	objIDs := []*primitive.ObjectID{}
	for _, id := range ids {
		realID, err := GetObjectIdFromString(id)
		if err != nil {
			return objIDs, err
		}
		objIDs = append(objIDs, realID)
	}
	return objIDs, nil
}

func GetObjectIdsFromInterface(ids interface{}) ([]*primitive.ObjectID, error) {
	if ids == nil {
		return []*primitive.ObjectID{}, errors.New("GetObjectIdsFromInterface: ids is nil")
	}

	switch ids.(type) {
	case []string:
		return GetObjectIdsFromStringArray(ids.([]string))
	case []*primitive.ObjectID:
		return ids.([]*primitive.ObjectID), nil
	default:
		return []*primitive.ObjectID{}, errors.New("GetObjectIdsFromInterface: invalid type")
	}
}

func HasValidID(id *primitive.ObjectID) bool {
	if id == nil {
		return false
	}
	if id.IsZero() {
		return false
	}
	return true
}

func HasValidIDStr(id string) bool {
	result := GetObjectIdFromStringRaw(id)
	return HasValidID(result)
}

func RemoveTrailingZeroes(str string) string {
	return strings.TrimRight(str, "0")
}

func GetValueNumToStr(container map[string]interface{}, key string) string {
	value, ok := container[key]
	if !ok {
		return ""
	}

	switch value.(type) {
	case int:
		return fmt.Sprint(value.(int))
	case float64:
		return fmt.Sprintf("%.0f", value)
	case string:
		return value.(string)
	}

	return ""

}

func GetValueNumToInt(container map[string]interface{}, key string) int {
	value, ok := container[key]
	if !ok {
		return 0
	}

	switch value.(type) {
	case int:
		return value.(int)
	case float64:
		return int(value.(float64))
	case string:
		num, err := strconv.Atoi(value.(string))
		if err != nil {
			return 0
		}
		return num
	}
	return 0
}

func GetValueToArrayID(container map[string]interface{}, key string) []*primitive.ObjectID {
	response := []*primitive.ObjectID{}
	value, ok := container[key]
	if !ok {
		return response
	}

	s := reflect.ValueOf(value)
	if s.Kind() != reflect.Slice {
		return response
	}

	if s.IsNil() {
		return response
	}

	for i := 0; i < s.Len(); i++ {
		idStr := s.Index(i).Interface().(string)
		id, err := GetObjectIdFromString(idStr)
		if err != nil {
			return response
		}
		response = append(response, id)
	}

	return response
}

func GetValueToArrayStr(container map[string]interface{}, key string) []string {
	response := []string{}
	value, ok := container[key]
	if !ok {
		return response
	}

	s := reflect.ValueOf(value)
	if s.Kind() != reflect.Slice {
		return response
	}

	if s.IsNil() {
		return response
	}

	for i := 0; i < s.Len(); i++ {
		str := s.Index(i).Interface().(string)
		response = append(response, str)
	}

	return response
}

func GetValueToStr(container map[string]interface{}, key string) string {
	value, ok := container[key]
	if !ok {
		return ""
	}
	if value == nil {
		return ""
	}

	switch value.(type) {
	case string:
		return value.(string)
	}

	return ""
}

func GetValueToObjectId(container map[string]interface{}, key string) *primitive.ObjectID {
	value, ok := container[key].(string)
	if !ok {
		return nil
	}
	value = strings.TrimSpace(value)
	objID, err := primitive.ObjectIDFromHex(value)
	if err != nil {
		return nil
	}

	return &objID
}

func GetStrFromObjecIDRaw(id *primitive.ObjectID) string {
	if id == nil {
		return ""
	}
	return id.Hex()
}

func GetStrFromObjecID(id *primitive.ObjectID) (string, error) {
	if id == nil {
		return "", errors.New("GetStrFromObjecID: id is nil")
	}
	return id.Hex(), nil
}

func ToRaw(model interface{}) []byte {
	raw, _ := json.Marshal(model)
	return raw
}

func ToBuffer(model map[string]interface{}) *bytes.Buffer {
	// test v1.4
	jsonBody, err := json.Marshal(model)
	if err != nil {
		return &bytes.Buffer{}
	}

	return bytes.NewBuffer(jsonBody)
}

func FromRaw(raw []byte, model interface{}) error {
	err := json.Unmarshal(raw, model)
	if err != nil {
		return err
	}
	return nil
}

func ToJSON(model interface{}) string {
	o, err := json.MarshalIndent(model, "", "\t")
	if err != nil {
		return "Error in conversion"
	}
	return string(o)
}

func ToJSONRaw(model interface{}) string {
	o, err := json.Marshal(model)
	if err != nil {
		return "Error in conversion"
	}
	return string(o)
}

func StrIsEqual(str1, str2 string) bool {
	return strings.ToLower(str1) == strings.ToLower(str2)
}

func ToLower(str string) string {
	return strings.ToLower(str)
}

func InterfaceIsNil(i interface{}) bool {

	if i == nil {
		return true
	}

	if reflect.ValueOf(i).IsNil() {
		return true
	}

	return false
}

func Encode(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

func Encrypt(text, secretKey string) (string, error) {
	block, err := aes.NewCipher([]byte(secretKey))
	if err != nil {
		return "", err
	}
	plainText := []byte(text)

	bytes := []byte{35, 46, 57, 24, 85, 35, 24, 74, 87, 35, 88, 98, 66, 32, 14, 05}
	cfb := cipher.NewCFBEncrypter(block, bytes)
	cipherText := make([]byte, len(plainText))
	cfb.XORKeyStream(cipherText, plainText)
	return Encode(cipherText), nil
}

func Decode(s string) []byte {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return data
}

func Decrypt(text, secretKey string) (string, error) {
	block, err := aes.NewCipher([]byte(secretKey))
	if err != nil {
		return "", err
	}
	cipherText := Decode(text)
	bytes := []byte{35, 46, 57, 24, 85, 35, 24, 74, 87, 35, 88, 98, 66, 32, 14, 05}
	cfb := cipher.NewCFBDecrypter(block, bytes)
	plainText := make([]byte, len(cipherText))
	cfb.XORKeyStream(plainText, cipherText)
	return string(plainText), nil
}

func ConverToSliceInterface(arg interface{}) []interface{} {
	slice, success := takeArg(arg, reflect.Slice)
	if !success {
		return nil
	}
	c := slice.Len()
	out := make([]interface{}, c)
	for i := 0; i < c; i++ {
		out[i] = slice.Index(i).Interface()
	}
	return out
}

func takeArg(arg interface{}, kind reflect.Kind) (val reflect.Value, ok bool) {
	val = reflect.ValueOf(arg)
	if val.Kind() == kind {
		ok = true
	}
	return
}

func AddObjectToArray(list interface{}, model interface{}) error {

	resultsVal := reflect.ValueOf(list)

	if resultsVal.Kind() != reflect.Ptr {
		return fmt.Errorf("results argument must be a pointer to a slice, but was a %s", resultsVal.Kind())
	}

	sliceVal := resultsVal.Elem()
	if sliceVal.Kind() == reflect.Interface {
		sliceVal = sliceVal.Elem()
	}

	if sliceVal.Kind() != reflect.Slice {
		return fmt.Errorf("results argument must be a pointer to a slice, but was a pointer to %s", sliceVal.Kind())
	}

	len := sliceVal.Len()

	if model == nil {
		return nil
	}

	switch model.(type) {
	case *primitive.ObjectID:
		if model.(*primitive.ObjectID) == nil {
			return nil
		}
	case string:
		if model.(string) == "" {
			return nil
		}
	}

	// Check if model is already in slice
	for i := 0; i < len; i++ {
		if sliceVal.Index(i).Interface() == model {
			return nil
		}
	}

	sliceVal = reflect.Append(sliceVal, reflect.ValueOf(model))

	len++
	resultsVal.Elem().Set(sliceVal.Slice(0, len))

	return nil
}

func RemoveObjectFromArray(model interface{}, list interface{}) error {
	resultsVal := reflect.ValueOf(list)
	if resultsVal.Kind() != reflect.Ptr {
		return fmt.Errorf("results argument must be a pointer to a slice, but was a %s", resultsVal.Kind())
	}

	sliceVal := resultsVal.Elem()
	if sliceVal.Kind() == reflect.Interface {
		sliceVal = sliceVal.Elem()
	}

	if sliceVal.Kind() != reflect.Slice {
		return fmt.Errorf("results argument must be a pointer to a slice, but was a pointer to %s", sliceVal.Kind())
	}

	len := sliceVal.Len()

	for i := 0; i < len; i++ {
		if sliceVal.Index(i).Interface() == model {
			sliceVal = reflect.AppendSlice(sliceVal.Slice(0, i), sliceVal.Slice(i+1, len))
			len--
			i--
		}
	}

	resultsVal.Elem().Set(sliceVal.Slice(0, len))

	return nil
}

func AddObjectIDToArray(list []*primitive.ObjectID, newID *primitive.ObjectID) []*primitive.ObjectID {
	if newID == nil {
		return list
	}

	for _, id := range list {
		if !HaveSameIDs(id, newID) {
			continue
		}
		return list
	}

	list = append(list, newID)
	return list
}
func GetInitialsFromString(text string) string {

	if text == "" {
		return ""
	}

	text = strings.ToUpper(text)

	// remove all except leters, spaces and numbers
	text = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsSpace(r) || unicode.IsNumber(r) {
			return r
		}
		return -1
	}, text)

	text = RemoveAccents(text)
	text = strings.TrimFunc(text, func(r rune) bool {
		return !unicode.IsGraphic(r)
	})

	// remove multiple spaces
	text = strings.Join(strings.Fields(text), " ")

	if text == "" {
		return ""
	}

	if text == " " {
		return ""
	}

	if len(text) < 3 {
		return text
	}

	if text == "" {
		return ""
	}

	words := strings.Split(text, " ")
	initials := ""
	for _, word := range words {
		initials += string(word[0])
	}

	// if len(initials) > 3 take first 3 from name
	if len(initials) < 3 {
		text = strings.Replace(text, " ", "", -1)
		initials = GetSlice(text, 3)
	}
	if len(initials) > 3 {
		initials = GetSlice(initials, 3)
	}

	return initials
}

func IsEmptyIntefaceList(list interface{}) bool {
	if list == nil {
		return false
	}
	resultsVal := reflect.ValueOf(list)
	sliceVal := resultsVal.Elem()
	if sliceVal.Kind() == reflect.Interface {
		sliceVal = sliceVal.Elem()
	}

	if sliceVal.Kind() != reflect.Slice {
		return false
	}

	if sliceVal.Len() < 1 {
		return false
	}

	return true
}

func ReplaceCharInStringSliceBetweenSymbols(s string, symbol string, old string, new string) string {
	if strings.Count(s, symbol) <= 1 {
		return s
	}

	count := 0
	sCopy := s
	firstSymbol := 0

	for strings.Contains(sCopy, symbol) {
		count += 1
		if count%2 == 1 {
			firstSymbol = strings.Index(sCopy, symbol)
		} else if count%2 == 0 {
			secondSymbol := strings.Index(sCopy, symbol)
			s = s[:firstSymbol] + strings.Replace(s[firstSymbol:secondSymbol+1], old, new, -1) + s[secondSymbol+1:]
		}
		sCopy = strings.Replace(sCopy, symbol, " ", 1)
	}

	return s
}

func SortStringArray(list []string) []string {
	sort.Strings(list)
	return list
}

func GetSlice(text string, lenght int) string {
	return GetSliceFromIndex(text, 0, lenght)
}

func GetSliceFromIndex(text string, index int, lenght int) string {
	if text == "" {
		return text
	}
	if lenght < 1 {
		return text
	}
	if lenght > len(text) {
		return text
	}
	return string(text[index : index+lenght])

}

func IndexOfStrArray(slice []string, element string) int {
	for i, v := range slice {
		if v == element {
			return i
		}
	}
	return -1
}

func CreateDateCode(prefix string) string {
	now := time.Now()
	return prefix + now.Format("200601021504")
}

func CloneStrArray(array []string) []string {
	result := []string{}
	for _, item := range array {
		result = append(result, item)
	}
	return result
}

func RemoveDots(text string) string {
	return strings.ReplaceAll(text, ".", "")
}

func CompareStringAndHash(text string, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(text)) == nil
}

func GenerateHash(text string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(text), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func IsEmptyIDArray(array []*primitive.ObjectID) bool {
	for _, item := range array {
		if item != nil {
			return false
		}
	}

	return true
}

func IsEmptyStrArray(array []string) bool {
	for _, item := range array {
		if item != "" {
			return false
		}
	}

	return true
}

func GetEnv(key string) string {
	return os.Getenv(key)
}

func GetEnvInt(key string, defaultValue int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return defaultValue
	}
	return value
}

func StructToStr(s interface{}) string {
	return fmt.Sprintf("%+v", s)
}

// SplitLevelArray divide una cadena separada por comas en un array de strings para niveles de log
func SplitLevelArray(text string) []string {
	if text == "" {
		return []string{}
	}
	// Eliminar espacios en blanco
	text = strings.TrimSpace(text)
	// Dividir por comas
	parts := strings.Split(text, ",")
	// Eliminar espacios en blanco y elementos vacíos
	result := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func StringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}
