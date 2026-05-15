package main

import (
	"bytes"
	"errors"
	"fmt"
	"gator/internal/config"
	"gator/internal/database"
	mockdb "gator/internal/database/mock"
	"slices"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
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

func TestNonRegisteredCommands(t *testing.T) {
	handlers := NewRegistry()
	handlers.Register("login", HandlerLogin)
	handlers.Register("register", HandlerRegister)

	cases := []struct {
		name        string
		expectedErr error
	}{
		{
			name:        "banana",
			expectedErr: ErrInvalidCmd,
		},
		{
			name:        "abc",
			expectedErr: ErrInvalidCmd,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("Test %v:", i+1), func(t *testing.T) {
			var buf bytes.Buffer
			state := State{Config: &config.Config{}}
			cmd, err := NewCommand(tc.name)
			if err != nil {
				t.Fatalf("expected no error got: %v", err)
			}

			err = handlers.Run(&buf, &state, cmd)
			if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("expected error %v, got: %v", tc.expectedErr, err)
			}
		})
	}
}

func TestHandlerLogin(t *testing.T) {
	handlers := NewRegistry()
	handlers.Register("login", HandlerLogin)

	cases := []struct {
		input          []string
		expectedRes    string
		expectedErr    error
		expectedSqlErr error
		shouldMock     bool
	}{
		{
			input:       []string{"login"},
			expectedErr: ErrNotEnoughArgs,
		},
		{
			input:       []string{"login", "alice"},
			expectedRes: "Login successful!",
			shouldMock:  true,
		},
		{
			input:          []string{"login", "alice"},
			expectedErr:    ErrUserNotRegistered,
			expectedSqlErr: errors.New("sql: no rows in result set"),
			shouldMock:     true,
		},
		{
			input:       []string{"login", "alice", "bob"},
			expectedErr: ErrTooManyArgs,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("Test %v", i+1), func(t *testing.T) {

			cmd, err := NewCommand(tc.input...)
			if err != nil {
				t.Fatalf("expected no error got: %v", err)
			}

			cfg := config.Config{}
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := mockdb.NewMockQuerier(ctrl)

			if tc.shouldMock {
				mockDB.EXPECT().
					GetUser(gomock.Any(), cmd.Args[0]).
					Return(database.User{Name: cmd.Args[0]}, tc.expectedSqlErr)
			}

			state := State{Db: mockDB, Config: &cfg}
			var buf bytes.Buffer
			err = handlers.Run(&buf, &state, cmd)

			// Evaluate function execution results
			if tc.expectedErr != nil {
				if err == nil {
					t.Fatal("expected an error got <nil>")
				}
				if !errors.Is(err, tc.expectedErr) {
					t.Fatalf("expected error %v, got: %v", tc.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				if !strings.Contains(buf.String(), tc.expectedRes) {
					t.Fatalf("expected output to contain: %q, got: %q", tc.expectedRes, buf.String())
				}
			}
		})
	}
}

func TestHandlerRegister(t *testing.T) {
	handlers := NewRegistry()
	handlers.Register("register", HandlerRegister)

	cases := []struct {
		input       []string
		expectedRes string
		expectedErr error
		SqlErr      error
		shouldMock  bool
	}{
		{
			input:       []string{"register"},
			expectedErr: ErrNotEnoughArgs,
		},
		{
			input:       []string{"register", "bob", "alice"},
			expectedErr: ErrTooManyArgs,
		},
		{
			input:       []string{"register", "bob"},
			expectedRes: "bob was successfully registered",
			shouldMock:  true,
		},
		{
			input:       []string{"register", "bob"},
			expectedErr: ErrAlreadyRegistered,
			SqlErr:      errors.New("ERROR: duplicate key value violates unique constraint"),
			shouldMock:  true,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("Test %v", i+1), func(t *testing.T) {
			cmd, err := NewCommand(tc.input...)
			if err != nil {
				t.Fatalf("expected no error got: %v", err)
			}

			cfg := config.Config{}
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDb := mockdb.NewMockQuerier(ctrl)

			if tc.shouldMock {
				expectedUser := strings.ToLower(cmd.Args[0])
				mockDb.EXPECT().
					CreateUser(gomock.Any(), gomock.Cond(func(u any) bool {
						params, ok := u.(database.CreateUserParams)
						if !ok {
							return false
						}
						return params.Name == expectedUser
					})).
					Return(database.User{Name: expectedUser}, tc.SqlErr)
			}

			var buf bytes.Buffer
			state := State{Db: mockDb, Config: &cfg}
			err = handlers.Run(&buf, &state, cmd)

			if tc.expectedErr != nil {
				if err == nil {
					t.Fatal("expected an error got <nil>")
				}
				if !errors.Is(err, tc.expectedErr) {
					t.Fatalf("expected error %v, got: %v", tc.expectedErr, err)
				}
			} else {
				if err != nil {
					t.Fatalf("expected no error got: %v", err)
				}

				if !strings.Contains(buf.String(), tc.expectedRes) {
					t.Fatalf("expected output to contain: %q, got: %q", tc.expectedRes, buf.String())
				}
			}
		})
	}
}
