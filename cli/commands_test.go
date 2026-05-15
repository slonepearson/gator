package main

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"gator/internal/config"
	"gator/internal/database"
	"slices"
	"strings"
	"testing"
)

func TestParseCliArgs(t *testing.T) {
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

func TestHandlerLogin(t *testing.T) {
	cfg, err := config.Read()
	if err != nil {
		fmt.Printf("Error: %v", err)
	}
	db, err := sql.Open("postgres", cfg.DbUrl)
	if err != nil {
		fmt.Printf("Error: %v", err)
	}
	if err := db.Ping(); err != nil {
		fmt.Printf("Error: %v", err)
	}
	defer db.Close()

	dbQueries := database.New(db)
	state := &State{
		Db:     dbQueries,
		Config: &cfg,
	}

	if err != nil {
		t.Fatal(err)
	}
	handlers := NewRegistry()
	handlers.Register("login", HandlerLogin)

	cases := []struct {
		input       []string
		expected    string
		expectedErr error
		wantErr     bool
	}{
		{
			input:       []string{"banana"},
			expectedErr: ErrInvalidCmd,
			wantErr:     true,
		},
		{
			input:       []string{"login"},
			expectedErr: ErrNotEnoughArgs,
			wantErr:     true,
		},
		{
			input:    []string{"login", "alice"},
			expected: "login successful!",
		},
		{
			input:       []string{"login", "alice", "bob"},
			expectedErr: ErrTooManyArgs,
			wantErr:     true,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("Test %d:", i+1), func(t *testing.T) {
			cmd, err := NewCommand(tc.input...)
			var buf bytes.Buffer
			err = handlers.Run(&buf, state, cmd)

			if tc.wantErr && err == nil {
				t.Fatal("expected an error got <nil>")
			}
			if tc.wantErr && !errors.Is(err, tc.expectedErr) {
				t.Fatalf("expected %v, got: %v", tc.expectedErr, err)
			}
			if !tc.wantErr && !strings.Contains(buf.String(), tc.expected) {
				t.Fatalf("expeted: %v, got: %v", tc.expected, buf.String())
			}
		})
	}
}
