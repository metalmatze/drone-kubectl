package main

import (
	"encoding/base64"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	helpers = map[string]interface{}{
		"uppercase":    strings.ToUpper,
		"lowercase":    strings.ToLower,
		"replace":      strings.Replace,
		"split":        strings.Split,
		"trim":         strings.Trim,
		"trimPrefix":   strings.TrimPrefix,
		"trimSuffix":   strings.TrimSuffix,
		"toTitle":      strings.ToTitle,
		"datetime":     datetime,
		"truncate":     truncate,
		"base64encode": base64encode,
		"base64decode": base64decode,
	}
)

func datetime(timestamp float64, layout, zone string) string {
	if zone == "" {
		return time.Unix(int64(timestamp), 0).Format(layout)
	}

	loc, err := time.LoadLocation(zone)

	if err != nil {
		return time.Unix(int64(timestamp), 0).Local().Format(layout)
	}

	return time.Unix(int64(timestamp), 0).In(loc).Format(layout)
}

func truncate(s string, len int) string {
	if utf8.RuneCountInString(s) <= len {
		return s
	}

	runes := []rune(s)

	return string(runes[:len])
}

func base64encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func base64decode(s string) string {
	data, err := base64.StdEncoding.DecodeString(s)

	if err != nil {
		return s
	}

	return string(data)
}
