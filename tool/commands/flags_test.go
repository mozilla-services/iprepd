package commands

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	cli "gopkg.in/urfave/cli.v1"
)

func TestName(t *testing.T) {
	tests := []struct {
		TestName string
		F        cli.Flag
		N        string
	}{
		{
			TestName: "test: flag with 1 name",
			F: cli.StringFlag{
				Name: "testflag",
			},
			N: "testflag",
		},
		{
			TestName: "test: flag with 2 names",
			F: cli.StringFlag{
				Name: "testflag, t",
			},
			N: "testflag",
		},
		{
			TestName: "test: flag with 2+ names",
			F: cli.StringFlag{
				Name: "testflag, t, a, b, c, d, e",
			},
			N: "testflag",
		},
	}

	for _, test := range tests {
		assert.Equal(t, name(test.F), test.N, test.TestName)
	}
}

func TestWithDefault(t *testing.T) {
	def := "testflag"
	outputFlag := withDefault(payloadFlag, def)
	assert.Equal(t, outputFlag.Value, def)
}

func TestWithDefaultInt(t *testing.T) {
	def := 60
	outputFlag := withDefaultInt(suppressRecoveryFlag, def)
	assert.Equal(t, outputFlag.Value, def)
}

func TestAsMandatory(t *testing.T) {
	outputFlag := asMandatory(payloadFlag)
	assert.True(t, strings.HasPrefix(outputFlag.Usage, mandatoryTag))
}

func TestAsMandatoryInt(t *testing.T) {
	outputFlag := asMandatoryInt(suppressRecoveryFlag)
	assert.True(t, strings.HasPrefix(outputFlag.Usage, mandatoryTag))
}
