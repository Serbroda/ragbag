package fetch

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

func TestRefusesNonPublicTargets(t *testing.T) {
	client := NewClient(false)
	for _, target := range []string{
		"http://localhost:8080/actuator/health", "http://127.0.0.1/", "http://10.0.0.1/", "http://192.168.1.1/",
		"http://169.254.169.254/latest/meta-data/", "http://[::1]/", "http://[fd00::1]/", "file:///etc/passwd",
	} {
		_, err := client.Get(context.Background(), target, 1024)
		require.Error(t, err, target)
		require.False(t, client.Metadata(context.Background(), target).Found(), target)
	}
}

func TestPrivateHostsAllowedStillRefusesLoopbackAndLinkLocal(t *testing.T) {
	client := NewClient(true)
	for _, target := range []string{"http://localhost:8080/", "http://127.0.0.1/", "http://169.254.169.254/", "http://[::1]/"} {
		_, err := client.Get(context.Background(), target, 1024)
		require.True(t, errors.Is(err, ErrForbidden), "%s: %v", target, err)
	}
	_, err := NewClient(false).Get(context.Background(), "http://192.168.1.1/", 1024)
	require.True(t, errors.Is(err, ErrPrivate))
	_, err = NewClient(false).Get(context.Background(), "http://does-not-exist.invalid/", 1024)
	require.True(t, errors.Is(err, ErrUnresolvable))
}

func parse(t *testing.T, page, base string) Metadata {
	doc, err := html.Parse(strings.NewReader(page))
	require.NoError(t, err)
	baseURL, err := url.Parse(base)
	require.NoError(t, err)
	return ParseMetadata(doc, baseURL)
}

func TestParsePrefersOpenGraphAndResolvesRelativeURLs(t *testing.T) {
	metadata := parse(t, `<html><head>
		<title>Svelte docs | Svelte</title>
		<meta property="og:title" content="Svelte docs">
		<meta property="og:description" content="Learn Svelte">
		<meta property="og:image" content="/og.png">
		<meta name="keywords" content="Svelte, JavaScript , svelte, ,Compiler">
		<link rel="icon" href="favicon.svg">
		<link rel="canonical" href="javascript:alert(1)">
		</head></html>`, "https://svelte.dev/docs/intro")

	require.Equal(t, "Svelte docs", metadata.Title)
	require.Equal(t, "Learn Svelte", metadata.Description)
	require.Equal(t, "https://svelte.dev/og.png", metadata.OgImage)
	require.Equal(t, "https://svelte.dev/docs/favicon.svg", metadata.Favicon)
	require.Equal(t, []string{"svelte", "javascript", "compiler"}, metadata.Keywords)
	require.Empty(t, metadata.Canonical, "non-http URLs are dropped")
}

func TestParseFallsBackToTitleTagAndDefaultFavicon(t *testing.T) {
	metadata := parse(t, `<html><head><title> Plain </title><meta name=description content=Desc></head></html>`,
		"https://example.com/a/b")
	require.Equal(t, "Plain", metadata.Title)
	require.Equal(t, "Desc", metadata.Description)
	require.Equal(t, "https://example.com/favicon.ico", metadata.Favicon)
	require.Empty(t, metadata.Keywords)
}

func TestNormalizeURL(t *testing.T) {
	for input, expected := range map[string]string{
		" example.com/docs ": "https://example.com/docs",
		"http://example.com": "http://example.com",
	} {
		normalized, err := NormalizeURL(input)
		require.NoError(t, err)
		require.Equal(t, expected, normalized)
	}
	for _, input := range []string{"", "javascript:alert(1)", "ftp://example.com", "data:text/html,x", "https://"} {
		_, err := NormalizeURL(input)
		require.Error(t, err, input)
	}
}
