package server_test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Serbroda/ragbag/internal/config"
	"github.com/Serbroda/ragbag/internal/mail"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// mailbox stands in for SMTP and keeps what the server sent (in the background).
type mailbox struct {
	mu       sync.Mutex
	messages []mail.Message
}

func (m *mailbox) Send(_ context.Context, msg mail.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mailbox) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

var mailLink = regexp.MustCompile(`https://ragbag\.test/app/(verify-email|reset-password)/(\S+)`)

// await waits for the n-th mail (1-based) and returns its link's kind and token.
func (m *mailbox) await(t *testing.T, n int, to string) (kind, token string) {
	t.Helper()
	require.Eventually(t, func() bool { return m.count() >= n }, 5*time.Second, 10*time.Millisecond, "mail %d", n)
	m.mu.Lock()
	defer m.mu.Unlock()
	msg := m.messages[n-1]
	require.Equal(t, to, msg.To)
	match := mailLink.FindStringSubmatch(msg.Text)
	require.NotNil(t, match, msg.Text)
	return match[1], match[2]
}

func newMailEnv(t *testing.T, options ...func(*config.Config)) (*env, *mailbox) {
	options = append(options, func(cfg *config.Config) { cfg.PublicURL = "https://ragbag.test" })
	e := newEnv(t, options...)
	box := &mailbox{}
	e.srv.UseMailer(box)
	return e, box
}

func (e *env) registerPending() (username, email string) {
	e.t.Helper()
	username = "u" + strings.ReplaceAll(uuid.NewString(), "-", "")[:8]
	email = username + "@example.com"
	e.req("POST", "/api/auth/register").
		json(fmt.Sprintf(`{"username":%q,"email":%q,"password":%q}`, username, email, testPassword)).expect(200)
	return username, email
}

func TestNewAccountsConfirmTheirEmailAddressFirst(t *testing.T) {
	e, box := newMailEnv(t)
	e.req("GET", "/api/app/config/features").expect(200).has("email", "true")
	username, email := e.registerPending()
	kind, token := box.await(t, 1, email)
	require.Equal(t, "verify-email", kind)

	// Signing in waits for the confirmation and sends a new link, which replaces the first one
	res := e.req("POST", "/api/auth/login").json(loginBody(username, testPassword)).expect(403)
	require.Contains(t, res.str("detail"), email)
	_, newToken := box.await(t, 2, email)
	e.req("POST", "/api/auth/verify-email/%s", token).expect(400)
	// A wrong password tells nothing and sends nothing
	e.req("POST", "/api/auth/login").json(loginBody(username, "wrong-password")).expect(401)

	res = e.req("POST", "/api/auth/verify-email/%s", newToken).expect(200)
	require.NotEmpty(t, res.str("access_token"))
	require.NotNil(t, res.cookieNamed("refresh_token"))
	user := &session{Token: res.str("access_token")}
	e.req("GET", "/api/v1/spaces").as(user).expect(200).has("#", 1)
	// The link works once; afterwards the account signs in normally
	e.req("POST", "/api/auth/verify-email/%s", newToken).expect(400)
	e.login(username, testPassword)
	require.Equal(t, 2, box.count())
}

func TestConfirmationLinksExpire(t *testing.T) {
	e, box := newMailEnv(t)
	_, email := e.registerPending()
	_, token := box.await(t, 1, email)
	_, err := e.db.Exec(e.db.Rebind(`UPDATE email_verifications SET expires_at = ?`), time.Now().UTC().Add(-time.Minute))
	require.NoError(t, err)
	e.req("POST", "/api/auth/verify-email/%s", token).expect(400)
}

func TestUnconfirmedAccountsAreDeletedAfterAWeek(t *testing.T) {
	e, box := newMailEnv(t)
	old, oldEmail := e.registerPending()
	box.await(t, 1, oldEmail)
	recent, recentEmail := e.registerPending()
	box.await(t, 2, recentEmail)
	confirmed, confirmedEmail := e.registerPending()
	_, verify := box.await(t, 3, confirmedEmail)
	e.req("POST", "/api/auth/verify-email/%s", verify).expect(200)
	_, err := e.db.Exec(e.db.Rebind(`UPDATE users SET created_at = ? WHERE username IN (?, ?)`),
		time.Now().UTC().Add(-8*24*time.Hour), old, confirmed)
	require.NoError(t, err)

	e.srv.DeleteUnverifiedAccounts(context.Background())
	e.req("POST", "/api/auth/login").json(loginBody(old, testPassword)).expect(401)
	e.req("POST", "/api/auth/login").json(loginBody(recent, testPassword)).expect(403)
	e.login(confirmed, testPassword)
}

func TestAccountsWaitingWhenEmailIsTurnedOffCanSignIn(t *testing.T) {
	e, box := newMailEnv(t)
	username, email := e.registerPending()
	box.await(t, 1, email)
	e.srv.UseMailer(nil)
	user := e.login(username, testPassword)
	e.req("GET", "/api/v1/spaces").as(user).expect(200).has("#", 1)
}

func TestPasswordResetByEmail(t *testing.T) {
	e, box := newMailEnv(t)
	username, email := e.registerPending()
	_, verify := box.await(t, 1, email)
	e.req("POST", "/api/auth/verify-email/%s", verify).expect(200)

	// Unknown accounts get the same answer, and no mail
	e.req("POST", "/api/auth/password-reset").json(`{"login":"nobody@example.com"}`).expect(204)
	e.req("POST", "/api/auth/password-reset").json(fmt.Sprintf(`{"login":%q}`, email)).expect(204)
	kind, token := box.await(t, 2, email)
	require.Equal(t, "reset-password", kind)
	e.req("GET", "/api/auth/password-reset/%s", token).expect(200).has("valid", true).has("username", username)
	e.req("POST", "/api/auth/password-reset/%s", token).json(`{"newPassword":"another-secret"}`).expect(204)
	e.req("POST", "/api/auth/login").json(loginBody(username, testPassword)).expect(401)
	e.login(username, "another-secret")
	// Used up
	e.req("GET", "/api/auth/password-reset/%s", token).expect(200).has("valid", false)
	require.Equal(t, 2, box.count())
}

func TestEmailedResetLinksExpireAfterAnHour(t *testing.T) {
	e, box := newMailEnv(t)
	username, email := e.registerPending()
	_, verify := box.await(t, 1, email)
	e.req("POST", "/api/auth/verify-email/%s", verify).expect(200)
	e.req("POST", "/api/auth/password-reset").json(fmt.Sprintf(`{"login":%q}`, username)).expect(204)
	box.await(t, 2, email)
	var expiresAt time.Time
	require.NoError(t, e.db.Get(&expiresAt, `SELECT expires_at FROM password_resets`))
	require.WithinDuration(t, time.Now().Add(time.Hour), expiresAt, time.Minute)
}

func TestMailsPerAccountArePaced(t *testing.T) {
	e, box := newMailEnv(t)
	username, email := e.registerPending()
	_, verify := box.await(t, 1, email)
	e.req("POST", "/api/auth/verify-email/%s", verify).expect(200)
	for range 5 {
		e.req("POST", "/api/auth/password-reset").json(fmt.Sprintf(`{"login":%q}`, username)).expect(204)
	}
	// The confirmation and three resets, the rest is dropped quietly
	box.await(t, 4, email)
	time.Sleep(100 * time.Millisecond)
	require.Equal(t, 4, box.count())
}

func TestWithoutEmailNothingChanges(t *testing.T) {
	e := newEnv(t)
	e.req("GET", "/api/app/config/features").expect(200).has("email", "false")
	user := e.register()
	e.req("GET", "/api/v1/spaces").as(user).expect(200).has("#", 1)
	e.req("POST", "/api/auth/password-reset").json(fmt.Sprintf(`{"login":%q}`, user.Username)).expect(400)
}

// refusingMailer stands in for an SMTP server that does not take mail.
type refusingMailer struct{}

func (refusingMailer) Send(context.Context, mail.Message) error {
	return errors.New("535 authentication failed")
}

func TestAdminsSendATestEmail(t *testing.T) {
	e := newEnv(t)
	admin := e.login("admin", "test123")
	user := e.register()
	box := &mailbox{}
	e.srv.UseMailer(box)
	e.req("POST", "/api/v1/admin/mail/test").as(user).expect(403)

	// Sent while the request waits, to the admin's own address
	e.req("POST", "/api/v1/admin/mail/test").as(admin).expect(204)
	require.Equal(t, 1, box.count())
	require.Equal(t, "admin@example.com", box.messages[0].To)

	// The mail server's error is shown
	e.srv.UseMailer(refusingMailer{})
	e.req("POST", "/api/v1/admin/mail/test").as(admin).expect(502).has("detail", "The mail server did not take the email: 535 authentication failed")

	e.srv.UseMailer(nil)
	e.req("POST", "/api/v1/admin/mail/test").as(admin).expect(400)
}
