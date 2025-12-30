package notifier

import (
	"context"

	"L3.1/internal/types"
)

// Notifier отправляет уведомление.
type Notifier interface {
	Send(ctx context.Context, ntf types.Notification) error
}
