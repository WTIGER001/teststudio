package identifier

import "regexp"

var pattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

func Valid(value string) bool { return pattern.MatchString(value) }
