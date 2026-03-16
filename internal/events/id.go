// Package events provides event-related utilities.
package events

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
)

// NewID generates a new event ID in the format "evt_{ulid}".
func NewID() string {
	id := ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader)
	return fmt.Sprintf("evt_%s", id.String())
}
