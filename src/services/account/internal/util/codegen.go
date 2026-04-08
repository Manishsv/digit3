package util

import "strings"

// Max length aligns with downstream services (e.g. boundary_v1.tenantId VARCHAR(64)).
const maxTenantCodeLen = 64

func GenerateCodeFromName(name string) string {
	s := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(name), " ", ""))
	if len(s) > maxTenantCodeLen {
		return s[:maxTenantCodeLen]
	}
	return s
} 