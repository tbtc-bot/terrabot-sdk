package queue

import (
	"context"
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

type RmqUserDataEvent struct {
	UserId    string                 `json:"userId"`
	BotId     string                 `json:"botId"`
	Body      map[string]interface{} `json:"event"`
	AccessKey string                 `json:"accessKey"`
	SecretKey string                 `json:"secretKey"`
}

type RmqApiServerCommandEvent struct {
	BotId        string `json:"botId"`
	UserId       string `json:"userId"`
	AccessKey    string `json:"accessKey"`
	SecretKey    string `json:"secretKey"`
	Command      string `json:"command"`
	Symbol       string `json:"symbol"`
	PositionSide string `json:"positionSide"`
}

type WsAccountUpdate struct {
	Reason    string       `json:"m"`
	Balances  []WsBalance  `json:"B"`
	Positions []WsPosition `json:"P"`
}

// WsBalance define balance
type WsBalance struct {
	Asset              string `json:"a"`
	Balance            string `json:"wb"`
	CrossWalletBalance string `json:"cw"`
	ChangeBalance      string `json:"bc"`
}

// WsPosition define position
type WsPosition struct {
	Symbol                    string `json:"s"`
	Side                      string `json:"ps"`
	Amount                    string `json:"pa"`
	MarginType                string `json:"mt"`
	IsolatedWallet            string `json:"iw"`
	EntryPrice                string `json:"ep"`
	MarkPrice                 string `json:"mp"`
	UnrealizedPnL             string `json:"up"`
	AccumulatedRealized       string `json:"cr"`
	MaintenanceMarginRequired string `json:"mm"`
}

type WsOrderTradeUpdate struct {
	Symbol               string `json:"s"`
	ClientOrderID        string `json:"c"`
	Side                 string `json:"S"`
	Type                 string `json:"o"`
	TimeInForce          string `json:"f"`
	OriginalQty          string `json:"q"`
	OriginalPrice        string `json:"p"`
	AveragePrice         string `json:"ap"`
	StopPrice            string `json:"sp"`
	ExecutionType        string `json:"x"`
	Status               string `json:"X"`
	ID                   int64  `json:"i"`
	LastFilledQty        string `json:"l"`
	AccumulatedFilledQty string `json:"z"`
	LastFilledPrice      string `json:"L"`
	CommissionAsset      string `json:"N"`
	Commission           string `json:"n"`
	TradeTime            int64  `json:"T"`
	TradeID              int64  `json:"t"`
	BidsNotional         string `json:"b"`
	AsksNotional         string `json:"a"`
	IsMaker              bool   `json:"m"`
	IsReduceOnly         bool   `json:"R"`
	WorkingType          string `json:"wt"`
	OriginalType         string `json:"ot"`
	PositionSide         string `json:"ps"`
	IsClosingPosition    bool   `json:"cp"`
	ActivationPrice      string `json:"AP"`
	CallbackRate         string `json:"cr"`
	RealizedPnL          string `json:"rp"`
}
