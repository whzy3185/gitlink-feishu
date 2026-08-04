package feishu

import "strings"

// reviewPortablePathBase treats both slash styles as separators regardless of
// the host operating system. Configuration and evidence can contain paths
// produced on a different platform than the service that reads them.
func reviewPortablePathBase(value string) string {
	value = strings.TrimSpace(value)
	trimmed := strings.TrimRight(value, `/\\`)
	if trimmed == "" || (len(trimmed) == 2 && trimmed[1] == ':') {
		return ""
	}
	if index := strings.LastIndexAny(trimmed, `/\\`); index >= 0 {
		return trimmed[index+1:]
	}
	return trimmed
}

func reviewPortablePathIsAbs(value string) bool {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, `\\`) || strings.HasPrefix(value, "//") {
		return true
	}
	return len(value) >= 3 && isReviewASCIILetter(value[0]) && value[1] == ':' && (value[2] == '/' || value[2] == '\\')
}

func isReviewASCIILetter(value byte) bool {
	return value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}
