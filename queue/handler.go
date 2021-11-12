package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
	"github.com/wagslane/go-rabbitmq"
	"go.opentelemetry.io/otel/trace"
)

type QueueHandler struct {
	Consumer  *rabbitmq.Consumer
	Publisher *rabbitmq.Publisher

	Cfg RabbitMqConfig
}

type RabbitMqConfig struct {
	Vhost     string
	User      string
	Password  string
	Hostnames string
	Port      string

	EventsConsumerName string
	CommandsQueueName  string
	CommandsRoutingKey string

	CommandsConsumerName string
	EventsQueueName      string
	EventsRoutingKey     string

	MdxExchange string
}

func NewQueueHandler(cfg RabbitMqConfig) (*QueueHandler, error) {

	// RabbitMQ connection
	amqpConfig := amqp091.Config{
		Vhost: cfg.Vhost,
	}
	consumer, err := rabbitmq.NewConsumer(
		"amqp://"+cfg.User+":"+cfg.Password+"@"+cfg.Hostnames+":"+cfg.Port+"/", amqpConfig,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ for Consumer: " + err.Error())
	}

	publisher, err := rabbitmq.NewPublisher(
		"amqp://"+cfg.User+":"+cfg.Password+"@"+cfg.Hostnames+":"+cfg.Port+"/", amqpConfig,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ for Publisher: " + err.Error())
	}

	return &QueueHandler{
		Consumer:  &consumer,
		Publisher: &publisher,
		Cfg:       cfg,
	}, nil
}

func (q *QueueHandler) StartConsumingQueueEvent(
	parseEventHandler func(ctx context.Context, eventRaw []byte),
	tracing bool,
	tracer trace.Tracer) error {

	rabbitMqHandler := func(d rabbitmq.Delivery) (action rabbitmq.Action) {

		ctx := context.Background()
		var genSpan trace.Span

		if tracing && d.Headers["x-opentelemetry-traceid"] != nil {

			spanid := d.Headers["x-opentelemetry-spanid"].(string)
			traceid := d.Headers["x-opentelemetry-traceid"].(string)
			traceflags := d.Headers["x-opentelemetry-traceflags"].(byte)
			tracestate := d.Headers["x-opentelemetry-tracestate"].(string)
			remote := d.Headers["x-opentelemetry-remote"].(bool)
			tid, _ := trace.TraceIDFromHex(traceid)
			sid, _ := trace.SpanIDFromHex(spanid)
			ts, _ := trace.ParseTraceState(tracestate)

			scc := &trace.SpanContextConfig{
				Remote:     remote,
				SpanID:     sid,
				TraceID:    tid,
				TraceState: ts,
				TraceFlags: trace.TraceFlags(traceflags),
			}

			sc := trace.NewSpanContext(*scc)
			ctx = trace.ContextWithSpanContext(ctx, sc) // inject current span

			ctx, genSpan = tracer.Start(ctx, "UserData Handler")

			genSpan.AddEvent("UserData Message Received")
		}

		parseEventHandler(ctx, d.Body)

		if tracing && d.Headers["x-opentelemetry-traceid"] != nil {
			genSpan.AddEvent("UserData Message Processed")
			genSpan.End()
		}

		// true to ACK, false to NACK
		return rabbitmq.Ack
	}

	return q.Consumer.StartConsuming(
		rabbitMqHandler,
		q.Cfg.EventsQueueName,            // queue name
		[]string{q.Cfg.EventsRoutingKey}, // routing keys
		rabbitmq.WithConsumeOptionsConcurrency(1),
		rabbitmq.WithConsumeOptionsQOSPrefetch(1),
		rabbitmq.WithConsumeOptionsConsumerAutoAck(true), // binance may break 1 or more API call during grid creation -> partial grid
		rabbitmq.WithConsumeOptionsConsumerName(q.Cfg.EventsConsumerName),
		rabbitmq.WithConsumeOptionsBindingExchangeDurable,
		rabbitmq.WithConsumeOptionsQueueDurable,
		rabbitmq.WithConsumeOptionsQuorum,
		rabbitmq.WithConsumeOptionsBindingExchangeName(q.Cfg.MdxExchange), // Message Deduplication Exchange
		rabbitmq.WithConsumeOptionsBindingExchangeKind("x-message-deduplication"),
		rabbitmq.WithConsumeOptionsBindingExchangeArgs(
			rabbitmq.Table{
				"x-cache-size": 1000000, // 1 million entries
				"x-cache-ttl":  10000,   // 10s
			},
		),
	)
}

func (q *QueueHandler) StartConsumingQueueCommand(parseCommandHandler func(eventRaw []byte)) error {

	rabbitMqHandler := func(d rabbitmq.Delivery) (action rabbitmq.Action) {
		parseCommandHandler(d.Body)
		// true to ACK, false to NACK
		return rabbitmq.Ack
	}

	return q.Consumer.StartConsuming(
		rabbitMqHandler,
		q.Cfg.CommandsQueueName,            // queue name
		[]string{q.Cfg.CommandsRoutingKey}, // routing keys
		rabbitmq.WithConsumeOptionsConcurrency(1),
		rabbitmq.WithConsumeOptionsQOSPrefetch(1),
		rabbitmq.WithConsumeOptionsConsumerName(q.Cfg.CommandsConsumerName),
		rabbitmq.WithConsumeOptionsBindingExchangeDurable,
		rabbitmq.WithConsumeOptionsQueueDurable,
		rabbitmq.WithConsumeOptionsQuorum,
		rabbitmq.WithConsumeOptionsBindingExchangeName("amq.direct"),
	)
}

// Telegram
type MessageType string

const (
	MsgTP      MessageType = "takeProfit"
	MsgInfo    MessageType = "info"
	MsgWarning MessageType = "warning"
	MsgError   MessageType = "error"
)

// RabbitMq
type RmqMessageEvent struct {
	BotId    string `json:"botId"`
	UserId   string `json:"userId"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // info | warn
}

type RmqTpEvent struct {
	BotId           string `json:"botId"`
	UserId          string `json:"userId"`
	Symbol          string `json:"symbol"`
	EventType       string `json:"eventType"`
	EventSide       string `json:"eventSide"`
	AveragePrice    string `json:"averagePrice"`
	FilledQty       string `json:"filledQty"`
	RealizedProfit  string `json:"realizedProfit"`
	ExecutedAt      int64  `json:"executedAt"`
	TotalGridSteps  string `json:"totalGridSteps"`
	CurrentGridStep string `json:"currentGridStep"`
}

func (q *QueueHandler) PublishTelegramMessage(body RmqMessageEvent, msgType MessageType) error {

	var routingKey string
	switch msgType {
	case MsgInfo, MsgWarning, MsgError:
		routingKey = "generic.notification.message"
	}

	reqBodyBytes := new(bytes.Buffer)
	json.NewEncoder(reqBodyBytes).Encode(&body)

	return q.Publisher.Publish(
		reqBodyBytes.Bytes(),
		[]string{routingKey},
		rabbitmq.WithPublishOptionsContentType("application/json"),
		rabbitmq.WithPublishOptionsMandatory,
		rabbitmq.WithPublishOptionsExchange("amq.direct"),
	)
}

func (q *QueueHandler) PublishTelegramTp(body RmqTpEvent) error {

	//todo readapt according to the parameters
	reqBodyBytes := new(bytes.Buffer)
	json.NewEncoder(reqBodyBytes).Encode(&body)

	return q.Publisher.Publish(
		reqBodyBytes.Bytes(),
		[]string{"generic.notification.takeprofit"},
		rabbitmq.WithPublishOptionsContentType("application/json"),
		rabbitmq.WithPublishOptionsMandatory,
		rabbitmq.WithPublishOptionsExchange("amq.direct"),
	)
}
