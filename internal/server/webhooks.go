package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/labstack/echo/v4"
)

// Discord webhook of a space: every link somebody saves is posted to the channel of the webhook, so a group sees
// what is new without opening Ragbag. Deliberately narrow:
//   - only addresses of Discord's webhook API are taken, so the server never calls an address somebody made up
//   - links in private collections (or below one) are never posted: the channel is read by people who cannot
//     see those; imports do not post either, they would flood the channel
//   - the address is a secret (who has it can post there): only members with EDIT_SPACE see it, API tokens
//     never, and it is not logged
//   - posting happens in the background and never fails saving a link; when Discord says the webhook is gone,
//     posting stops until a new address is set

const (
	webhookTimeout  = 10 * time.Second
	discordEmbedCol = 0xf59e0b
)

// discordHosts are the hosts webhooks of Discord live on.
var discordHosts = map[string]bool{
	"discord.com": true, "discordapp.com": true, "canary.discord.com": true, "ptb.discord.com": true,
}

// WebhookSender posts a JSON payload and returns the HTTP status.
type WebhookSender func(ctx context.Context, address string, payload []byte) (int, error)

// UseWebhookSender replaces the HTTP sender (tests).
func (s *Server) UseWebhookSender(sender WebhookSender) { s.webhookSender = sender }

// postJSON is the sender outside of tests. Redirects are not followed: the address was checked, where it would
// lead to was not.
func postJSON(ctx context.Context, address string, payload []byte) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, address, bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	return res.StatusCode, nil
}

// discordWebhookURL checks an address typed by a user: https://discord.com/api/webhooks/<id>/<token>.
func discordWebhookURL(value string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "https" || !discordHosts[strings.ToLower(parsed.Host)] || parsed.User != nil {
		return "", badRequest("That is not a Discord webhook: it starts with https://discord.com/api/webhooks/")
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 4 || parts[0] != "api" || parts[1] != "webhooks" || parts[2] == "" || parts[3] == "" {
		return "", badRequest("That is not a Discord webhook: it starts with https://discord.com/api/webhooks/")
	}
	// Without query or fragment: Discord's options (wait, thread_id) are not ours to pass on
	return "https://" + strings.ToLower(parsed.Host) + "/" + strings.Join(parts, "/"), nil
}

type discordMessage struct {
	Username string         `json:"username"`
	Embeds   []discordEmbed `json:"embeds"`
	// Empty on purpose: a title like "@everyone" must not ping anybody
	AllowedMentions struct {
		Parse []string `json:"parse"`
	} `json:"allowed_mentions"`
}

type discordEmbed struct {
	Title       string         `json:"title"`
	URL         string         `json:"url,omitempty"`
	Description string         `json:"description,omitempty"`
	Color       int            `json:"color"`
	Author      *discordNamed  `json:"author,omitempty"`
	Fields      []discordField `json:"fields,omitempty"`
	Footer      *discordFooter `json:"footer,omitempty"`
	Image       *discordImage  `json:"image,omitempty"`
	Timestamp   string         `json:"timestamp,omitempty"`
}

type discordNamed struct {
	Name string `json:"name"`
}

type discordField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type discordFooter struct {
	Text string `json:"text"`
}

type discordImage struct {
	URL string `json:"url"`
}

// plainText keeps Discord from reading text of others as Markdown (links with another target, headings).
var plainText = strings.NewReplacer(`\`, `\\`, "`", "\\`", "*", `\*`, "_", `\_`, "~", `\~`, "|", `\|`, "[", `\[`, "]", `\]`,
	"(", `\(`, ")", `\)`, "<", `\<`, ">", `\>`, "#", `\#`)

// discordMessageFor is the card of a new link. collectionURL leads to its collection in Ragbag: the last thing
// of the card that can be a link (footers cannot), so it sits in a field below the description.
func discordMessageFor(bookmark *store.Bookmark, spaceName, collectionName, collectionURL string) discordMessage {
	embed := discordEmbed{
		Title:       truncate(firstNonEmpty(deref(bookmark.Title), bookmark.URL), 250),
		URL:         bookmark.URL,
		Description: plainText.Replace(truncate(deref(bookmark.Description), 300)),
		Color:       discordEmbedCol,
		Author:      &discordNamed{Name: truncate(bookmark.CreatedByName+" added a link", 250)},
		Fields: []discordField{{
			Name:  "Collection",
			Value: "[" + plainText.Replace(truncate(collectionName, 100)) + "](" + collectionURL + ")",
		}},
		Footer:    &discordFooter{Text: truncate(spaceName, 250)},
		Timestamp: bookmark.CreatedAt.UTC().Format(time.RFC3339),
	}
	// The preview as the app shows it, if Discord can load it: an own address, else the page's image
	if !bookmark.ImageHidden {
		if image := httpURLOrNil(firstPtr(bookmark.ImageURL, bookmark.OgImage)); image != nil {
			embed.Image = &discordImage{URL: *image}
		}
	}
	message := discordMessage{Username: "Ragbag", Embeds: []discordEmbed{embed}}
	message.AllowedMentions.Parse = []string{}
	return message
}

func firstPtr(values ...*string) *string {
	for _, value := range values {
		if value != nil && *value != "" {
			return value
		}
	}
	return nil
}

// announceLink posts a link somebody just saved to the Discord channel of its space, in the background.
func (s *Server) announceLink(c echo.Context, bookmarkID string) {
	ctx := ctxOf(c)
	bookmark, err := s.store.BookmarkByID(ctx, bookmarkID)
	if err != nil {
		return
	}
	webhook, err := s.store.SpaceWebhook(ctx, bookmark.SpaceID)
	if err != nil || webhook.URL == nil || webhook.FailedAt != nil {
		return
	}
	_, byID, err := s.spaceCollections(c, bookmark.SpaceID)
	if err != nil {
		return
	}
	collection := byID[bookmark.CollectionID]
	if collection == nil || privateInPath(byID, collection) {
		return
	}
	space, err := s.store.SpaceByID(ctx, bookmark.SpaceID)
	if err != nil {
		return
	}
	// Like the link previews: PUBLIC_URL, else the address the app was just used at
	base := s.cfg.PublicURL
	if base == "" {
		base = c.Scheme() + "://" + c.Request().Host
	}
	collectionURL := fmt.Sprintf("%s/space/%s/collections/%s", base, space.ID, collection.ID)
	s.postToDiscord(space.ID, *webhook.URL, discordMessageFor(bookmark, space.Name, collection.Name, collectionURL))
}

func (s *Server) postToDiscord(spaceID, address string, message discordMessage) {
	payload, err := json.Marshal(message)
	if err != nil {
		return
	}
	sender := s.webhookSender
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), webhookTimeout)
		defer cancel()
		status, err := sender(ctx, address, payload)
		switch {
		case err != nil:
			// Never the error itself: it holds the address
			slog.Warn("Discord webhook could not be reached", "space", spaceID)
		case status == http.StatusNotFound || status == http.StatusUnauthorized:
			slog.Warn("Discord webhook is gone, posting stops", "space", spaceID, "status", status)
			_ = s.store.FailSpaceWebhook(ctx, spaceID, address)
		case status >= 300:
			slog.Warn("Discord refused a message", "space", spaceID, "status", status)
		}
	}()
}

func (s *Server) webhookView(c echo.Context, spaceID string) error {
	webhook, err := s.store.SpaceWebhook(ctxOf(c), spaceID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, api.SpaceWebhookDto{Url: webhook.URL, FailedAt: webhook.FailedAt})
}

func (s *Server) GetSpaceWebhook(c echo.Context, spaceID api.ID) error {
	if _, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermEditSpace); err != nil {
		return err
	}
	return s.webhookView(c, spaceID)
}

func (s *Server) SetSpaceWebhook(c echo.Context, spaceID api.ID) error {
	if _, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermEditSpace); err != nil {
		return err
	}
	var req api.SetSpaceWebhookRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	address, err := discordWebhookURL(req.Url)
	if err != nil {
		return err
	}
	if err := s.store.SetSpaceWebhook(ctxOf(c), spaceID, &address); err != nil {
		return err
	}
	audit(c, slog.LevelInfo, "Discord webhook set", "space", spaceID)
	return s.webhookView(c, spaceID)
}

func (s *Server) DeleteSpaceWebhook(c echo.Context, spaceID api.ID) error {
	if _, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermEditSpace); err != nil {
		return err
	}
	if err := s.store.SetSpaceWebhook(ctxOf(c), spaceID, nil); err != nil {
		return err
	}
	audit(c, slog.LevelInfo, "Discord webhook removed", "space", spaceID)
	return c.NoContent(http.StatusNoContent)
}

// TestSpaceWebhook posts a test message right away and answers with what Discord said, so a wrong address shows.
func (s *Server) TestSpaceWebhook(c echo.Context, spaceID api.ID) error {
	if _, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermEditSpace); err != nil {
		return err
	}
	webhook, err := s.store.SpaceWebhook(ctxOf(c), spaceID)
	if err != nil {
		return err
	}
	if webhook.URL == nil {
		return badRequest("Set a Discord webhook first")
	}
	space, err := s.store.SpaceByID(ctxOf(c), spaceID)
	if err != nil {
		return err
	}
	message := discordMessage{Username: "Ragbag", Embeds: []discordEmbed{{
		Title:       "It works",
		Description: plainText.Replace(fmt.Sprintf("New links of %s will show up here.", space.Name)),
		Color:       discordEmbedCol,
	}}}
	message.AllowedMentions.Parse = []string{}
	payload, _ := json.Marshal(message)
	ctx, cancel := context.WithTimeout(ctxOf(c), webhookTimeout)
	defer cancel()
	status, err := s.webhookSender(ctx, *webhook.URL, payload)
	switch {
	case err != nil:
		return badRequest("Discord could not be reached, try again later")
	case status == http.StatusNotFound || status == http.StatusUnauthorized:
		return badRequest("Discord does not know this webhook: was it deleted? Create a new one in the channel settings")
	case status >= 300:
		return badRequest("Discord refused the message (status %d)", status)
	}
	// It works (again)
	if webhook.FailedAt != nil {
		if err := s.store.SetSpaceWebhook(ctxOf(c), spaceID, webhook.URL); err != nil {
			return err
		}
	}
	return c.NoContent(http.StatusNoContent)
}
