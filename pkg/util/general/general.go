package general

import (
	"math/rand"
	"regexp"
	"strings"
	"time"
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
