package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	appErorrs "L3.1/internal/errors"
	"L3.1/internal/notifier"
	"L3.1/internal/producer"
	"L3.1/internal/store"
	"L3.1/internal/types"
	"github.com/rabbitmq/amqp091-go"
	"github.com/wb-go/wbf/rabbitmq"
)

// HandleNotification обрабатывает входящие уведомления из RabbitMQ,
// выполняет отправку через notifier и управляет retry/backoff логикой.
// Возвращает функцию-обработчик для consumer.
func HandleNotification(
	s store.Store,
	nt notifier.Notifier,
	errPublisher *rabbitmq.Publisher,
	delayPublisher *rabbitmq.Publisher,
) func(ctx context.Context, msg amqp091.Delivery) error {

	return func(ctx context.Context, msg amqp091.Delivery) error {

		log.Printf("[CONSUMER] Raw message: %s", string(msg.Body))

		var env types.NotificationEnvelope
		if err := json.Unmarshal(msg.Body, &env); err != nil {
			log.Printf("[CONSUMER] Bad JSON: %v", err)
			if pubErr := errPublisher.Publish(ctx, msg.Body, "notify.error"); pubErr != nil {
				log.Printf("[ERROR] failed to publish to error queue: %v", pubErr)
			}
			return nil // ACK
		}

		ntf, err := s.Get(env.ID)
		if err != nil || ntf.Canceled {
			return nil // ACK
		}

		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		err = nt.Send(ctx, *ntf)
		if err != nil {
			if errors.Is(err, appErorrs.ErrTemporary) {
				ntf.Attempts++
				ntf.LastError = err.Error()

				delay := time.Duration(ntf.Attempts*5) * time.Second
				ntf.NextAttemptAt = time.Now().Add(delay).Format(time.RFC3339)

				if updErr := s.Update(env.ID, *ntf); updErr != nil {
					log.Printf("[ERROR] failed to update store: %v", updErr)
				}

				if pubErr := delayPublisher.Publish(ctx, msg.Body, "notify.delay", producer.WithTTL(delay)); pubErr != nil {
					log.Printf("[ERROR] failed to publish delay message: %v", pubErr)
				}

				return nil // ACK
			}

			if errors.Is(err, appErorrs.ErrFatal) {
				ntf.LastError = err.Error()
				if updErr := s.Update(env.ID, *ntf); updErr != nil {
					log.Printf("[ERROR] failed to update store: %v", updErr)
				}

				if pubErr := errPublisher.Publish(ctx, msg.Body, "notify.error"); pubErr != nil {
					log.Printf("[ERROR] failed to publish to error queue: %v", pubErr)
				}

				return nil // ACK
			}

			return appErorrs.ErrTemporary
		}

		log.Printf("[CONSUMER] SEND OK: %s", ntf.Message)

		ntf.LastError = ""
		if updErr := s.Update(env.ID, *ntf); updErr != nil {
			log.Printf("[ERROR] failed to update store: %v", updErr)
		}

		return nil // ACK
	}
}

// HandleNotificationOnlyLog — тестовый обработчик
func HandleNotificationOnlyLog(ctx context.Context, msg amqp091.Delivery) error {
	log.Printf("consumer got: %s", msg.Body)
	return nil
}
