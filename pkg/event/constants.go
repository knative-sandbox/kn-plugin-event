package event

import (
	"fmt"

	"github.com/google/uuid"
	"knative.dev/kn-plugin-event/pkg"
)

const (
	// DefaultType holds a default type for a event.
	DefaultType = "dev.knative.cli.plugin.event.generic"
)

// DefaultSource holds a default source of an event.
func DefaultSource() string {
	return fmt.Sprintf("%s/%s", pkg.PluginName, pkg.Version)
}

// NewID creates a new ID for an event.
func NewID() string {
	return uuid.New().String()
}
