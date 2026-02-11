package util

import "strings"

func GenerateCodeFromName(name string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(name), " ", ""))
} 