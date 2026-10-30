package telemetry

func MaskSecret(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 12 {
		return "..."
	}
	return s[:7] + "..." + s[len(s)-5:]
}
