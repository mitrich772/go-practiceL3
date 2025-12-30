package producer

import (
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/wb-go/wbf/rabbitmq"
)

// WithTTL возвращает опцию публикации, устанавливающую TTL сообщения
func WithTTL(d time.Duration) rabbitmq.PublishOption {
	return func(p *amqp091.Publishing) {
		p.Expiration = fmt.Sprintf("%d", d.Milliseconds())
	}
}
