package main

import (
	"errors"
	"fmt"
	"slices"
	"testing"
)

func TestNewCommand(t *testing.T) {
	cases := []struct {
		input       []string
		expected    Command
		expectedErr error
		wantErr     bool
	}{
		{
			input:       []string{},
			expectedErr: ErrNotEnoughArgs,
			wantErr:     true,
		},
		{
			input:    []string{"login"},
			expected: Command{Name: "login", Args: []string{}},
		},
		{
			input:    []string{"login", "alice", "bob"},
			expected: Command{Name: "login", Args: []string{"alice", "bob"}},
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("Test %d:", i+1), func(t *testing.T) {
			cmd, err := NewCommand(tc.input...)

			if tc.wantErr && err == nil {
				t.Fatal("expected an error got <nil>")
			}

			if tc.wantErr && !errors.Is(err, tc.expectedErr) {
				t.Fatalf("expected %v, got: %v", tc.expectedErr, err)
			}

			if !tc.wantErr && cmd.Name != tc.expected.Name && !slices.Equal(cmd.Args, tc.expected.Args) {
				t.Fatalf("Expect command: %#v\n Got command: %#v\n", tc.expected, cmd)
			}
		})
	}
}
