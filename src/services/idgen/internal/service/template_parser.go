package service

import (
	"strings"
)

type segment struct {
	literal string
	token   string
}

func parseTemplate(tmpl string) []segment {
	var result []segment
	for {
		start := strings.Index(tmpl, "{")
		if start == -1 {
			if tmpl != "" {
				result = append(result, segment{literal: tmpl})
			}
			break
		}
		if start > 0 {
			result = append(result, segment{literal: tmpl[:start]})
		}
		end := strings.Index(tmpl, "}")
		if end == -1 {
			break
		}
		token := tmpl[start+1 : end]
		result = append(result, segment{token: token})
		tmpl = tmpl[end+1:]
	}
	return result
}
