package db

import (
	"math/rand"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

var entropy = ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)

type DBID ulid.ULID

func NewDBID() DBID {
	return DBID(ulid.MustNew(ulid.Timestamp(time.Now()), entropy))
}

func ParseDBID(s string) (DBID, error) {
	id, err := ulid.Parse(strings.TrimSpace(s))
	return DBID(id), err
}

func (id DBID) String() string {
	return id.String()
}
