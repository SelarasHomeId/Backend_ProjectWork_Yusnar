package general

import (
	"bytes"
	"io/ioutil"
	"math/rand"
	"regexp"
	"selarashomeid/internal/abstraction"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/sirupsen/logrus"
)

func IsValidEmail(email string) bool {
	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(emailRegex)
	return re.MatchString(email)
}

func IsValidPhone(phone string) bool {
	phoneNumberRegex := `^\+[1-9]\d{1,14}$`
	re := regexp.MustCompile(phoneNumberRegex)
	return re.MatchString(phone)
}

// Now ...
func Now() *time.Time {
	now := time.Now()
	return &now
}

// NowUTC ...
func NowUTC() *time.Time {
	now := time.Now().UTC()
	return &now
}

// NowLocal ...
func NowLocal() *time.Time {
	now := time.Now().UTC().Add(time.Hour * 7)
	return &now
}

// NowWithLocation ...
func NowWithLocation() *time.Time {
	now := time.Now().In(Location())
	return &now
}

// Location ...
func Location() *time.Location {
	return time.FixedZone("Asia/Jakarta", 7*60*60)
}

func Parse(layout, value string) (time.Time, error) {
	return time.ParseInLocation(layout, value, Location())
}

// LastWeek ...
func LastWeek(now time.Time) (start time.Time, end time.Time) {
	end = StartOfWeek(now).Add(-1)

	oneWeek := (24 * 6) * time.Hour
	start = StartOfDay(end.Add(-oneWeek))
	return
}

// LastMonth ...
func LastMonth(now time.Time) (time.Time, time.Time) {
	end := StartOfMonth(now).Add(-time.Nanosecond)
	return StartOfMonth(end), end
}

// StartOfMonth ...
func StartOfMonth(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
}

// StartOfWeek ...
func StartOfWeek(now time.Time) time.Time {
	wd := now.Weekday()
	if wd == time.Sunday {
		now = now.AddDate(0, 0, -6)
	} else {
		now = now.AddDate(0, 0, -int(wd)+1)
	}
	return StartOfDay(now)
}

// StartOfDay ...
func StartOfDay(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

// EndOfDay ...
func EndOfDay(now time.Time) time.Time {
	return time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, int(time.Second-1), now.Location())
}

func GeneratePassword(passwordLength, minSpecialChar, minNum, minUpperCase, minLowerCase int) string {
	var password strings.Builder
	var lowerCharSet string = "abcdedfghijklmnopqrstuvwxyz"
	var upperCharSet string = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	var specialCharSet string = "!@#$%&*"
	var numberSet string = "0123456789"
	var allCharSet string = lowerCharSet + upperCharSet + specialCharSet + numberSet

	//Set special character
	for i := 0; i < minSpecialChar; i++ {
		random := rand.Intn(len(specialCharSet))
		password.WriteString(string(specialCharSet[random]))
	}

	//Set numeric
	for i := 0; i < minNum; i++ {
		random := rand.Intn(len(numberSet))
		password.WriteString(string(numberSet[random]))
	}

	//Set uppercase
	for i := 0; i < minUpperCase; i++ {
		random := rand.Intn(len(upperCharSet))
		password.WriteString(string(upperCharSet[random]))
	}

	//Set lowercase
	for i := 0; i < minLowerCase; i++ {
		random := rand.Intn(len(lowerCharSet))
		password.WriteString(string(lowerCharSet[random]))
	}

	remainingLength := passwordLength - minSpecialChar - minNum - minUpperCase - minLowerCase
	for i := 0; i < remainingLength; i++ {
		random := rand.Intn(len(allCharSet))
		password.WriteString(string(allCharSet[random]))
	}
	inRune := []rune(password.String())
	rand.Shuffle(len(inRune), func(i, j int) {
		inRune[i], inRune[j] = inRune[j], inRune[i]
	})
	return string(inRune)
}

func SanitizeStringOfAlphabet(input string) string {
	// Menghapus karakter yang bukan huruf, underscore
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' {
			return r
		}
		return -1
	}, input)
}

func SanitizeStringOfNumber(input string) string {
	// Menghapus karakter yang bukan angka
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, input)
}

func SanitizeString(input string) string {
	// Define regex to remove special characters that could be used in SQL injection
	re := regexp.MustCompile(`[%'";()=<>` + "`" + `#\-\[\]]`)
	sanitized := re.ReplaceAllString(input, "")

	// Only allow letters, numbers, and underscores as per original function logic
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return -1
	}, sanitized)
}

func SanitizeStringDateBetween(input string) string {
	// Define regex untuk format tanggal yang diinginkan: YYYY-MM-DD_YYYY-MM-DD
	re := regexp.MustCompile(`[^0-9\-_]`)
	// Hapus semua karakter yang tidak sesuai dengan format yang diinginkan
	sanitized := re.ReplaceAllString(input, "")

	// Pastikan bahwa input sesuai dengan format 'YYYY-MM-DD_YYYY-MM-DD'
	// regex untuk mencocokkan tanggal dengan format yang benar
	dateFormat := `^\d{4}-\d{2}-\d{2}_\d{4}-\d{2}-\d{2}$`
	dateRe := regexp.MustCompile(dateFormat)

	// Jika format tidak sesuai, kembalikan string kosong atau bisa diubah sesuai kebutuhan
	if !dateRe.MatchString(sanitized) {
		return ""
	}

	return sanitized
}

func ProcessWhereParam(ctx *abstraction.Context, searchType string, whereStr string) (string, map[string]interface{}) {
	var (
		where      = "1=1 AND " + whereStr
		whereParam = map[string]interface{}{
			"false": false,
			"true":  true,
		}
	)

	if ctx.QueryParam("search") != "" {
		val := "%" + SanitizeString(ctx.QueryParam("search")) + "%"
		switch searchType {
		case "user":
			where += " AND (LOWER(name) LIKE @search_name OR LOWER(email) LIKE @search_email)"
			whereParam["search_name"] = val
			whereParam["search_email"] = val
		}
	}
	if ctx.QueryParam("id") != "" {
		val, _ := strconv.Atoi(SanitizeStringOfNumber(ctx.QueryParam("id")))
		where += " AND id = @id"
		whereParam["id"] = val
	}
	if ctx.QueryParam("name") != "" {
		val := "%" + SanitizeString(ctx.QueryParam("name")) + "%"
		where += " AND LOWER(name) LIKE @name"
		whereParam["name"] = val
	}
	if ctx.QueryParam("email") != "" {
		val := "%" + SanitizeString(ctx.QueryParam("email")) + "%"
		where += " AND LOWER(email) LIKE @email"
		whereParam["email"] = val
	}
	if ctx.QueryParam("role_id") != "" {
		val, _ := strconv.Atoi(SanitizeStringOfNumber(ctx.QueryParam("role_id")))
		where += " AND role_id = @role_id"
		whereParam["role_id"] = val
	}
	if ctx.QueryParam("divisi_id") != "" {
		val, _ := strconv.Atoi(SanitizeStringOfNumber(ctx.QueryParam("divisi_id")))
		where += " AND divisi_id = @divisi_id"
		whereParam["divisi_id"] = val
	}
	if ctx.QueryParam("is_locked") != "" {
		where += " AND is_locked = @" + SanitizeStringOfAlphabet(ctx.QueryParam("is_locked"))
	}
	if ctx.QueryParam("login_from") != "" {
		val := "%" + SanitizeString(ctx.QueryParam("login_from")) + "%"
		where += " AND LOWER(login_from) LIKE @login_from"
		whereParam["login_from"] = val
	}
	if ctx.QueryParam("created_at") != "" {
		val := SanitizeStringDateBetween(ctx.QueryParam("created_at"))
		valDate := strings.Split(val, "_")
		where += " AND created_at BETWEEN @start_created_at AND @end_created_at"
		whereParam["start_created_at"] = valDate[0]
		whereParam["end_created_at"] = valDate[1]
	}

	if where == "" {
		whereParam = nil
	}

	return where, whereParam
}

func ProcessLimitOffset(ctx *abstraction.Context) (int, int) {
	var (
		limit  = 10
		offset = 1
	)
	if ctx.QueryParam("page_size") != "" {
		ps, _ := strconv.Atoi(SanitizeStringOfNumber(ctx.QueryParam("page_size")))
		limit = ps
	}
	if ctx.QueryParam("page") != "" {
		p, _ := strconv.Atoi(SanitizeStringOfNumber(ctx.QueryParam("page")))
		offset = p
	}
	return limit, (offset - 1) * limit
}

func ProcessOrder(ctx *abstraction.Context) string {
	var (
		order string
		o     = "id"
		ob    = "ASC"
	)
	if ctx.QueryParam("order") != "" {
		o = ValidationOrder(ctx.QueryParam("order"))
	}
	if ctx.QueryParam("order_by") != "" {
		ob = ValidationOrderBy(ctx.QueryParam("order_by"))
	}
	order = o + " " + ob
	return order
}

func ValidationOrder(str string) string {
	str = SanitizeString(str)
	str = strings.ToLower(str)
	orderStack := []string{"id", "name", "email"}
	for _, item := range orderStack {
		if item == str {
			return str
		}
	}
	return "id"
}

func ValidationOrderBy(str string) string {
	str = SanitizeStringOfAlphabet(str)
	str = strings.ToUpper(str)
	orderStack := []string{"ASC", "DESC"}
	for _, item := range orderStack {
		if item == str {
			return str
		}
	}
	return "ASC"
}

func ParseTemplateEmail(templateFileName string, data interface{}) string {
	t, err := template.ParseFiles(templateFileName)
	if err != nil {
		logrus.Error("Error paring template email: ", err.Error())
		return ""
	}
	buf := new(bytes.Buffer)
	if err = t.Execute(buf, data); err != nil {
		logrus.Error("Error paring template email: ", err.Error())
		return ""
	}
	return buf.String()
}

func ProcessHTMLResponseEmail(filePath, placeholder, value string) string {
	content, _ := ioutil.ReadFile(filePath)
	return strings.Replace(string(content), placeholder, value, -1)
}
