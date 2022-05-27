package stringutils

func NilToEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func EPTThen(s string, then string) string {
	if s == "" {
		return then
	}
	return s
}
