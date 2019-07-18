package iprepd

import (
	"fmt"
	"net"
	"regexp"
)

var validators = map[string]func(string) error{
	"ip":    validateTypeIP,
	"email": validateTypeEmail,
}

func validateTypeIP(val string) error {
	if net.ParseIP(val) == nil {
		return fmt.Errorf("invalid ip format %v", val)
	}
	return nil
}

func validateTypeEmail(val string) error {
	re := regexp.MustCompile("^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$")
	if !re.MatchString(val) {
		return fmt.Errorf("invalid email format %v", val)
	}
	return nil
}

func validateType(t string, val string) error {
	if fn, ok := validators[t]; ok {
		return fn(val)
	}
	return fmt.Errorf("unknown type for validation %v", t)
}
