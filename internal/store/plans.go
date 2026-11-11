package store

import (
	"context"
	"time"
)

// Plan is what makes an account Pro (PLANS_ENABLED) besides being an instance admin.
type Plan struct {
	// ProGrantedAt: made Pro by an instance admin
	ProGrantedAt *time.Time `db:"pro_granted_at"`
}

const planColumns = `pro_granted_at`

const userPlanColumns = `u.pro_granted_at`

// GrantPro makes an account Pro, or takes that back.
func (s *Store) GrantPro(ctx context.Context, userID string, granted bool) error {
	var at *time.Time
	if granted {
		t := now()
		at = &t
	}
	_, err := s.exec(ctx, `UPDATE users SET pro_granted_at = ?, updated_at = ? WHERE id = ?`, at, now(), userID)
	return err
}

// SpaceOwners are the owners of a space; the space has Pro limits when one of them is Pro.
func (s *Store) SpaceOwners(ctx context.Context, spaceID string) ([]User, error) {
	var users []User
	err := s.sel(ctx, &users, `SELECT `+selectedUserColumns+` FROM users WHERE id IN
		(SELECT user_id FROM space_members WHERE space_id = ? AND role = 'OWNER') ORDER BY username`, spaceID)
	return users, err
}
