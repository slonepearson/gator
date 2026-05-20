package main

import (
	"context"
	"io"
	"time"
)

func WithLoggedIn(next handlerFuncWAuth) handlerFunc {
	return func(ctx context.Context, w io.Writer, s *State, cmd Command) error {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		user, err := s.Db.GetUserByName(ctx, s.Config.CurrentUserName)
		if err != nil {
			return err
		}
		return next(ctx, w, s, cmd, user)
	}
}
