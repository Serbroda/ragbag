package server

import (
	"context"
	"errors"
	"log/slog"

	"github.com/Serbroda/ragbag/internal/auth"
	"github.com/Serbroda/ragbag/internal/store"
)

const (
	demoUser     = "admin"
	demoPassword = "test123"
)

// SeedDemo creates the demo account admin/test123 with a few collections. Only used with RAGBAG_DEV=true.
func (s *Server) SeedDemo(ctx context.Context) error {
	if _, err := s.store.UserByLogin(ctx, demoUser); err == nil {
		return nil
	} else if !errors.Is(err, store.ErrNotFound) {
		return err
	}
	hash, err := auth.HashPassword(demoPassword)
	if err != nil {
		return err
	}
	err = s.store.InTx(ctx, func(tx *store.Store) error {
		user, err := tx.CreateUser(ctx, demoUser, "admin@example.com", hash)
		if err != nil {
			return err
		}
		space, err := tx.CreateSpace(ctx, "Admin's Space", nil, user.ID)
		if err != nil {
			return err
		}
		if err := tx.AddMember(ctx, space.ID, user.ID, string(RoleOwner)); err != nil {
			return err
		}
		create := func(name string, parent *store.Collection) (*store.Collection, error) {
			c := &store.Collection{SpaceID: space.ID, Name: name, CreatedBy: user.ID}
			if parent != nil {
				c.ParentID = &parent.ID
			}
			return c, tx.CreateCollection(ctx, c, nil)
		}
		food, err := create("Food", nil)
		if err != nil {
			return err
		}
		for _, name := range []string{"Fruits", "Vegetables"} {
			if _, err := create(name, food); err != nil {
				return err
			}
		}
		meat, err := create("Meat", food)
		if err != nil {
			return err
		}
		for _, name := range []string{"Beef", "Pork"} {
			if _, err := create(name, meat); err != nil {
				return err
			}
		}
		_, err = create("Books", nil)
		return err
	})
	if err == nil {
		slog.Info("demo user created", "username", demoUser, "password", demoPassword)
	}
	return err
}
