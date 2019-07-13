package iprepd

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateType(t *testing.T) {
	tests := []struct {
		Name        string
		Type        string
		Object      string
		ExpectErr   bool
		ExpectedErr error
	}{
		{
			Name:      "test: validate good IP",
			Type:      "ip",
			Object:    "228.28.28.28",
			ExpectErr: false,
		},
		{
			Name:      "test: validate good Email",
			Type:      "email",
			Object:    "sstallone@mozilla.com",
			ExpectErr: false,
		},
		{
			Name:        "test: validate bad IP",
			Type:        "ip",
			Object:      "2assfa28.28",
			ExpectErr:   true,
			ExpectedErr: fmt.Errorf("invalid ip format %v", "2assfa28.28"),
		},
		{
			Name:        "test: validate bad Email",
			Type:        "email",
			Object:      "not an email",
			ExpectErr:   true,
			ExpectedErr: fmt.Errorf("invalid email format %v", "not an email"),
		},
		{
			Name:        "test: validate wrong type",
			Type:        "email",
			Object:      "228.28.28.28",
			ExpectErr:   true,
			ExpectedErr: fmt.Errorf("invalid email format %v", "228.28.28.28"),
		},
		{
			Name:        "test: validate unknown type",
			Type:        "some undefined type",
			Object:      "228.28.28.28",
			ExpectErr:   true,
			ExpectedErr: fmt.Errorf("unknown type for validation %v", "some undefined type"),
		},
	}

	for _, tst := range tests {
		err := validateType(tst.Type, tst.Object)
		if tst.ExpectErr {
			assert.Equal(t, tst.ExpectedErr, err, tst.Name)
		} else {
			assert.Nil(t, err, tst.Name)
		}
	}
}
