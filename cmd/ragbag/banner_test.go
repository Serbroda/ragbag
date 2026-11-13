package main

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTheStartMessageLeavesOutDatabasePasswords(t *testing.T) {
	require.Equal(t, "SQLite /data/ragbag.db", describeDatabase("sqlite:/data/ragbag.db"))
	require.Equal(t, "PostgreSQL db:5432/ragbag", describeDatabase("postgres://ragbag:secret@db:5432/ragbag?sslmode=disable"))
}

func TestTheStartMessageNamesThePort(t *testing.T) {
	require.Equal(t, "port 8080 (all interfaces)", listenAddress(&net.TCPAddr{IP: net.IPv6unspecified, Port: 8080}))
	require.Equal(t, "127.0.0.1, port 8080", listenAddress(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8080}))
}
