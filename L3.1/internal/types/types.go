package types

// Notification — сущность уведомления, которая хранится в Store.
type Notification struct {
	Message  string `json:"message"`
	SendAt   string `json:"send_at"`
	Canceled bool   `json:"canceled"`

	Attempts      int    `json:"attempts"`
	NextAttemptAt string `json:"next_attempt_at"`
	LastError     string `json:"last_error"`
}

// NotificationEnvelope — то, что отправляется в RabbitMQ.
type NotificationEnvelope struct {
	ID      string `json:"id"`
	Message string `json:"message"`
	SendAt  string `json:"send_at"`
}
