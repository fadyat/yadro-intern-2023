package main

import (
	"os"
	"testing"
)

func TestParseArgs(t *testing.T) {
	testCases := []struct {
		name        string
		expectedArg string
		args        []string
		isErrExp    bool
	}{
		{
			name:     "no file arg",
			args:     []string{"exec"},
			isErrExp: true,
		},
		{
			name:        "file arg",
			expectedArg: "filename",
			args:        []string{"exec", "filename"},
			isErrExp:    false,
		},
		{
			name:        "file arg with extra args",
			expectedArg: "filename",
			args:        []string{"exec", "filename", "extra"},
			isErrExp:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Args = tc.args
			arg, err := parseArgs()
			if err != nil && !tc.isErrExp {
				t.Errorf("unexpected error: %s", err)
			}

			if arg != tc.expectedArg {
				t.Errorf("expected arg %s, got %s", tc.expectedArg, arg)
			}
		})
	}
}
