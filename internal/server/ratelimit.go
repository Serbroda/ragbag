package server

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

// limiter is a token bucket per key (an IP address, a user or a login name), kept in memory: limits start over
// with a restart, which is fine for slowing down guessing and floods. Full buckets are forgotten.
type limiter struct {
	rate  rate.Limit
	burst int

	mu        sync.Mutex
	buckets   map[string]*rate.Limiter
	lastSweep time.Time
}

// newLimiter allows burst requests at once and then one every interval.
func newLimiter(interval time.Duration, burst int) *limiter {
	return &limiter{rate: rate.Every(interval), burst: burst, buckets: map[string]*rate.Limiter{}, lastSweep: time.Now()}
}

func (l *limiter) bucket(key string, now time.Time) *rate.Limiter {
	if now.Sub(l.lastSweep) > time.Minute {
		for k, b := range l.buckets {
			if b.TokensAt(now) >= float64(l.burst) {
				delete(l.buckets, k)
			}
		}
		l.lastSweep = now
	}
	b, ok := l.buckets[key]
	if !ok {
		b = rate.NewLimiter(l.rate, l.burst)
		l.buckets[key] = b
	}
	return b
}

// take uses up one request; when none is left it returns how long to wait instead.
func (l *limiter) take(key string) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	reservation := l.bucket(key, now).ReserveN(now, 1)
	if delay := reservation.DelayFrom(now); delay > 0 {
		reservation.CancelAt(now)
		return delay
	}
	return 0
}

// wait is how long until the next request would be allowed, without using one up.
func (l *limiter) wait(key string) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	missing := 1 - l.bucket(key, now).TokensAt(now)
	if missing <= 0 {
		return 0
	}
	return time.Duration(missing / float64(l.rate) * float64(time.Second))
}

// limits are the rate limits of the server, nil when RATE_LIMIT_ENABLED=false.
type limits struct {
	// api is every API request per IP: generous, against floods and scraping
	api *limiter
	// login, register and passwordReset are per IP: password guessing and bot sign-ups
	login, register, passwordReset *limiter
	// loginFailures is per login name: guessing one account's password from many addresses
	loginFailures *limiter
	// password is changing the password per user (it checks the current one)
	password *limiter
	// fetch is loading user supplied URLs (page details, images) per user: the server is no free proxy
	fetch *limiter
}

func newLimits() *limits {
	return &limits{
		api:           newLimiter(50*time.Millisecond, 600),
		login:         newLimiter(6*time.Second, 10),
		register:      newLimiter(10*time.Minute, 5),
		passwordReset: newLimiter(time.Minute, 10),
		loginFailures: newLimiter(3*time.Minute, 10),
		password:      newLimiter(time.Minute, 5),
		fetch:         newLimiter(2*time.Second, 60),
	}
}

// clientKey is the client's IP address; IPv6 by its /64, which usually belongs to one connection.
func clientKey(c echo.Context) string {
	ip := net.ParseIP(c.RealIP())
	if ip == nil {
		return c.RealIP()
	}
	if ip.To4() == nil {
		return ip.Mask(net.CIDRMask(64, 128)).String()
	}
	return ip.String()
}

// refuse answers 429 with a Retry-After header.
func refuse(c echo.Context, wait time.Duration, detail string) error {
	c.Response().Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
	return tooManyRequests("%s", detail)
}

func (l *limits) check(c echo.Context, lim *limiter, key, detail string) error {
	if wait := lim.take(key); wait > 0 {
		return refuse(c, wait, detail)
	}
	return nil
}

// rateLimit applies the per IP limits; per user and per account ones are checked in the handlers.
func (s *Server) rateLimit(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		l, req := s.limits, c.Request()
		if l == nil || req.Method == http.MethodOptions {
			return next(c)
		}
		ip := clientKey(c)
		if err := l.check(c, l.api, ip, "Too many requests, please slow down"); err != nil {
			return err
		}
		var err error
		switch path := req.URL.Path; {
		case req.Method == http.MethodPost && path == "/api/auth/login":
			err = l.check(c, l.login, ip, "Too many login attempts, try again in a minute")
		case req.Method == http.MethodPost && path == "/api/auth/register":
			err = l.check(c, l.register, ip, "Too many new accounts from your network, try again later")
		case strings.HasPrefix(path, "/api/auth/password-reset/"):
			err = l.check(c, l.passwordReset, ip, "Too many attempts, try again in a minute")
		}
		if err != nil {
			return err
		}
		return next(c)
	}
}

// fetchAllowed is false when the user loaded too many URLs lately; links are then saved without loading details.
func (s *Server) fetchAllowed(c echo.Context) bool {
	return s.limits == nil || s.limits.fetch.take(currentUser(c)) == 0
}

// limitFetch refuses loading a user supplied URL when the user loaded too many lately.
func (s *Server) limitFetch(c echo.Context) error {
	if s.limits == nil {
		return nil
	}
	return s.limits.check(c, s.limits.fetch, currentUser(c), "Too many pages loaded, try again in a minute")
}

func (s *Server) limitPasswordChange(c echo.Context) error {
	if s.limits == nil {
		return nil
	}
	return s.limits.check(c, s.limits.password, currentUser(c), "Too many attempts, try again in a minute")
}

// loginBlocked refuses logins to an account after too many failed ones, wherever they came from.
func (s *Server) loginBlocked(c echo.Context, login string) error {
	if s.limits == nil {
		return nil
	}
	if wait := s.limits.loginFailures.wait(login); wait > 0 {
		return refuse(c, wait, "Too many failed logins for this account, try again in a few minutes")
	}
	return nil
}

func (s *Server) loginFailed(login string) {
	if s.limits != nil {
		s.limits.loginFailures.take(login)
	}
}
