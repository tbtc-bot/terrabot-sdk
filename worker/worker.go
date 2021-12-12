package worker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/spf13/viper"
	"github.com/tbtc-bot/terrabot-sdk"
	"github.com/tbtc-bot/terrabot-sdk/cache"
	"github.com/tbtc-bot/terrabot-sdk/database"
	"github.com/tbtc-bot/terrabot-sdk/exchange"
	"github.com/tbtc-bot/terrabot-sdk/queue"
	"github.com/tbtc-bot/terrabot-sdk/strategy"
	"github.com/tbtc-bot/terrabot-sdk/telegram"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var tracer trace.Tracer = otel.Tracer("worker_f")

type Worker struct {
	exchange string

	qh *queue.QueueHandler
	ch *cache.RedisHandler
	dh *database.FirestoreHandler
	eh exchange.ExchangeConnector
	sh *strategy.StrategyHandler
	th *telegram.TelegramHandler

	Logger *zap.Logger

	podName         string
	applicationMode string

	tracing            bool
	TracerProvider     *sdktrace.TracerProvider
	googleCloudProject string
}

// Build a new worker instance
func NewWorker(
	exchange string,
	queueHandler *queue.QueueHandler,
	cacheHandler *cache.RedisHandler,
	databaseHandler *database.FirestoreHandler,
	exchangeConnector exchange.ExchangeConnector,
	logger *zap.Logger,
	podName string,
	applicationMode string,
	tracing bool,
	googleCloudProject string) *Worker {

	telegramHandler := &telegram.TelegramHandler{Qh: queueHandler, Logger: logger}

	strategyHandler := strategy.NewStrategyHandler(cacheHandler, databaseHandler, exchangeConnector, telegramHandler, logger)

	// Instantiate worker
	worker := &Worker{
		exchange: exchange,

		qh: queueHandler,
		ch: cacheHandler,
		dh: databaseHandler,
		eh: exchangeConnector,
		sh: strategyHandler,
		th: telegramHandler,

		Logger: logger,

		podName:         podName,
		applicationMode: applicationMode,

		tracing:            tracing,
		googleCloudProject: googleCloudProject,
	}

	// tracing
	if worker.tracing {
		logger.Info("Tracing activated")
		setupTracing(podName, logger)
	}

	return worker
}

// Starts to listen commands and events
func (w *Worker) Start() {
	/*
	 * * Some considerations to be done on QoS & Prefetch & Concurrency
	 * * https://www.rabbitmq.com/confirms.html
	 */

	if err := w.qh.StartConsumingQueueEvent(w.parseQueueEvent, w.tracing, tracer); err != nil {
		w.Logger.Fatal("Cannot listen to Events queue",
			zap.String("error", err.Error()))
	}

	if err := w.qh.StartConsumingQueueCommand(w.sh.ParseCommandEvent); err != nil {
		w.Logger.Fatal("Cannot listen to Commands queue",
			zap.String("error", err.Error()))
	}
}

// Parse the event from the exchange
func (w *Worker) parseQueueEvent(ctx context.Context, eventRaw []byte) {

	var queueEvent queue.RmqUserDataEvent

	if err := json.Unmarshal(eventRaw, &queueEvent); err != nil {
		w.Logger.Error("Error during unmarshaling of event",
			zap.String("error", err.Error()),
		)
		return
	}

	session := terrabot.NewSession(queueEvent.BotId, queueEvent.UserId, queueEvent.AccessKey, queueEvent.SecretKey, terrabot.Strategy{})
	event := queueEvent.Body

	switch event["e"] {

	case "ACCOUNT_UPDATE":
		eventRaw, _ := json.Marshal(event["a"])
		var event queue.WsAccountUpdate
		if err := json.Unmarshal(eventRaw, &event); err != nil {

			w.Logger.Error("Could not convert to WsAccountUpdate structure",
				zap.String("error", err.Error()))
			return
		}
		if event.Reason == "ORDER" {
			w.sh.HandleAccountUpdate(ctx, *session, &event)
		}

	case "ORDER_TRADE_UPDATE":
		eventRaw, _ := json.Marshal(event["o"])
		var event queue.WsOrderTradeUpdate
		if err := json.Unmarshal(eventRaw, &event); err != nil {

			w.Logger.Error("Could not convert to WsOrderUpdate structure",
				zap.String("error", err.Error()))

			return
		}
		w.sh.HandleOrderUpdate(ctx, *session, &event)

	}
}

// Shutdown the Worker gracefully
func (w *Worker) Shutdown() {

	w.qh.Consumer.StopConsuming(w.qh.Cfg.CommandsConsumerName, false)
	w.qh.Consumer.StopConsuming(w.qh.Cfg.EventsConsumerName, false)
	w.qh.Publisher.StopPublishing()
	time.Sleep(10000)

	w.qh.Consumer.Disconnect()
}

func setupTracing(podName string, logger *zap.Logger) {
	zipkinUrl := viper.GetString("zipkinUrl")
	tracingSampleRatio := viper.GetFloat64("tracingSampleRatio")

	exporter, err := zipkin.New(
		zipkinUrl,
	)
	if err != nil {
		logger.Error("Unable to connect to zipkin: " + err.Error())
	}

	batcher := sdktrace.NewBatchSpanProcessor(exporter)

	// todo recover at compile time
	// https://medium.com/geekculture/golang-app-build-version-in-containers-3d4833a55094
	// https://polyverse.com/blog/how-to-embed-versioning-information-in-go-applications-f76e2579b572/
	resources := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("futures-worker"),
		semconv.ServiceVersionKey.String("1.0.0"),
		semconv.ServiceInstanceIDKey.String(podName),
	)

	TracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(batcher),
		sdktrace.WithResource(resources),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(tracingSampleRatio))),
	)

	otel.SetTracerProvider(TracerProvider)
	propagator := propagation.NewCompositeTextMapPropagator(propagation.Baggage{}, propagation.TraceContext{})
	otel.SetTextMapPropagator(propagator)
}
