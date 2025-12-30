package main

import (
	"context"
	"html/template"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"L3.1/internal/consumer"
	"L3.1/internal/notifier"
	"L3.1/internal/store"
	"L3.1/internal/web"
	"github.com/joho/godotenv"
	"github.com/rabbitmq/amqp091-go"
	"github.com/wb-go/wbf/rabbitmq"
	"github.com/wb-go/wbf/retry"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	ctx, cancel := context.WithCancel(context.Background())
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stop
		cancel()
	}()

	// rabbitmq конфиг
	cfg := rabbitmq.ClientConfig{
		URL:            os.Getenv("RABBITMQ_URL"),
		ConnectTimeout: 5 * time.Second,
		Heartbeat:      10 * time.Second,

		PublishRetry: retry.Strategy{
			Attempts: 5,
			Delay:    200 * time.Millisecond,
			Backoff:  2.0,
		},
		ConsumeRetry: retry.Strategy{
			Attempts: 0,
			Delay:    500 * time.Millisecond,
			Backoff:  1.5,
		},
	}
	// rabbitmq клиент
	client, err := rabbitmq.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// основная exchange
	err = client.DeclareExchange("notify-exchange", "direct", true, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	// delay-exchange
	err = client.DeclareExchange("notify-delay-exchange", "direct", true, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	// error-exchange
	err = client.DeclareExchange("notify-error-exchange", "direct", true, false, false, nil)
	if err != nil {
		log.Fatal(err)
	}

	// delay-queue (для ожидания по TTL)
	err = client.DeclareQueue(
		"notify-delay-queue",
		"notify-delay-exchange",
		"notify.delay",
		true,
		false,
		true,
		amqp091.Table{
			"x-dead-letter-exchange":    "notify-exchange",
			"x-dead-letter-routing-key": "notify.key",
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	// основная очередь
	err = client.DeclareQueue(
		"notify-queue",
		"notify-exchange",
		"notify.key",
		true,
		false,
		true,
		amqp091.Table{
			"x-dead-letter-exchange":    "notify-delay-exchange", // для retry
			"x-dead-letter-routing-key": "notify.delay",
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	// помойка
	err = client.DeclareQueue(
		"notify-error-queue",
		"notify-error-exchange",
		"notify.error",
		true,
		false,
		true,
		nil, // финальная помойка
	)
	if err != nil {
		log.Fatal(err)
	}

	// consumer
	consumerCfg := rabbitmq.ConsumerConfig{
		Queue:         "notify-queue",
		ConsumerTag:   "notify-consumer",
		AutoAck:       false,
		Workers:       1,
		PrefetchCount: 1,
	}

	errorPublisher := rabbitmq.NewPublisher(client, "notify-error-exchange", "application/json")
	delayPublisher := rabbitmq.NewPublisher(client, "notify-delay-exchange", "application/json")
	str := store.NewInMemoryStore()

	timeoutStr := os.Getenv("NOTIFIER_TIMEOUT")
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		log.Printf("[WARN] invalid NOTIFIER_TIMEOUT=%q, using default 10s", timeoutStr)
		timeout = 10 * time.Second
	}
	tgNotif := notifier.NewTelegram(os.Getenv("TG_BOT_TOKEN"), os.Getenv("TG_CHAT_ID"), timeout)

	notifCons := rabbitmq.NewConsumer(client, consumerCfg, consumer.HandleNotification(str, tgNotif, errorPublisher, delayPublisher))

	go func() {
		for {
			err := notifCons.Start(ctx)
			log.Println("consumer stopped:", err)

			if ctx.Err() != nil {
				return
			}

			time.Sleep(100 * time.Millisecond)
		}
	}()

	time.Sleep(300 * time.Millisecond)

	// publisher — публикуем в delay-exchange
	publisher := rabbitmq.NewPublisher(client, "notify-delay-exchange", "application/json")
	tpl := template.Must(template.ParseFiles("templates/index.html"))

	server := web.NewServer("1235", tpl, publisher, str)
	go func() {
		if err := server.Start(); err != nil {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown")
}
