package types

import (
	"time"

	"github.com/spf13/cobra"
)

// ConvertibleNotifier is a notifier capable of creating an Apprise URL
type ConvertibleNotifier interface {
	GetURL(c *cobra.Command) (string, error)
}

// DelayNotifier is a notifier that might need to be delayed before sending notifications
type DelayNotifier interface {
	GetDelay() time.Duration
}
