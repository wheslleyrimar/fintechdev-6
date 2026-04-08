package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var (
	logger             *zap.Logger
	tracer             trace.Tracer
	asyncLagExtraDelay = 0 * time.Millisecond

	// Métricas RED
	messagesProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "antifraud_messages_processed_total",
			Help: "Total messages processed by antifraud service",
		},
		[]string{"status"},
	)

	processingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "antifraud_processing_duration_seconds",
			Help:    "Antifraud processing duration",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
		},
		[]string{"status"},
	)

	processingDurationPercentiles = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "antifraud_processing_duration_percentiles",
			Help:       "Processing duration percentiles",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.01, 0.99: 0.001},
		},
		[]string{"status"},
	)

	// Métricas de negócio
	fraudDetected = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "antifraud_fraud_detected_total",
			Help: "Total fraud cases detected",
		},
	)

	riskScore = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "antifraud_risk_score",
			Help:    "Risk score distribution",
			Buckets: []float64{0, 20, 40, 60, 80, 100},
		},
	)

	antifraudScenarioEnabled = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "antifraud_scenario_enabled",
			Help: "Whether an antifraud teaching scenario is enabled",
		},
		[]string{"scenario"},
	)
)

func setAsyncLagScenario(enabled bool) {
	if enabled {
		antifraudScenarioEnabled.WithLabelValues("async_lag").Set(1)
		return
	}
	antifraudScenarioEnabled.WithLabelValues("async_lag").Set(0)
}

func initTracing() {
	ctx := context.Background()
	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "antifraud-service"
	}

	var (
		exporter tracesdk.SpanExporter
		err      error
	)

	otlpEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT")
	if otlpEndpoint == "" {
		otlpEndpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	}

	if otlpEndpoint != "" {
		exporter, err = otlptracehttp.New(
			ctx,
			otlptracehttp.WithEndpointURL(otlpEndpoint),
		)
		if err != nil {
			logger.Error("Failed to create OTLP exporter", zap.Error(err), zap.String("endpoint", otlpEndpoint))
			return
		}
		logger.Info("Using OTLP for tracing", zap.String("endpoint", otlpEndpoint))
	} else if jaegerEndpoint := os.Getenv("JAEGER_ENDPOINT"); jaegerEndpoint != "" {
		collectorURL := fmt.Sprintf("http://%s/api/traces", jaegerEndpoint)
		exporter, err = jaeger.New(
			jaeger.WithCollectorEndpoint(
				jaeger.WithEndpoint(collectorURL),
			),
		)
		if err != nil {
			logger.Error("Failed to create Jaeger exporter", zap.Error(err))
			return
		}
		logger.Info("Using Jaeger for tracing", zap.String("endpoint", collectorURL))
	} else {
		logger.Warn("No tracing exporter configured, tracing disabled")
		return
	}

	// Sampling: 5% (menos crítico que payment)
	sampler := tracesdk.TraceIDRatioBased(0.05)

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		)),
		tracesdk.WithSampler(sampler),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer = otel.Tracer(serviceName)
}

func initLogger() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}
}

func extractTraceContext(headers amqp.Table) (context.Context, trace.SpanContext) {
	ctx := context.Background()
	propagator := otel.GetTextMapPropagator()

	// Converter amqp.Table para map[string]string para propagação
	carrier := make(map[string]string)
	if headers != nil {
		if traceID, ok := headers["X-Trace-ID"].(string); ok {
			carrier["traceparent"] = traceID
		}
		if correlationID, ok := headers["X-Correlation-ID"].(string); ok {
			carrier["baggage-correlation-id"] = correlationID
		}
	}

	ctx = propagator.Extract(ctx, propagation.MapCarrier(carrier))
	return ctx, trace.SpanContext{}
}

func processPayment(ctx context.Context, msgBody []byte, headers amqp.Table) {
	start := time.Now()

	// Extrair correlation ID e trace ID
	correlationID := ""
	traceID := ""
	if headers != nil {
		if cid, ok := headers["X-Correlation-ID"].(string); ok {
			correlationID = cid
		}
		if tid, ok := headers["X-Trace-ID"].(string); ok {
			traceID = tid
		}
	}

	ctx, span := tracer.Start(ctx, "antifraud.process")
	defer span.End()

	var event map[string]interface{}
	if err := json.Unmarshal(msgBody, &event); err != nil {
		logger.Error("failed_to_unmarshal",
			zap.Error(err),
			zap.String("correlation_id", correlationID),
			zap.String("trace_id", traceID),
		)
		messagesProcessed.WithLabelValues("error").Inc()
		return
	}

	paymentID, _ := event["paymentId"].(string)
	amount, _ := event["amount"].(float64)

	// Simular processamento antifraud (com latência variável)
	processingTime := 50*time.Millisecond + time.Duration(rand.Intn(100))*time.Millisecond
	// 1% tem latência muito alta (cauda)
	if rand.Float64() < 0.01 {
		processingTime += 500 * time.Millisecond
	}
	if asyncLagExtraDelay > 0 {
		processingTime += asyncLagExtraDelay
		span.SetAttributes(attribute.Int64("scenario.async_lag_ms", asyncLagExtraDelay.Milliseconds()))
	}
	time.Sleep(processingTime)

	// Calcular risk score
	riskScoreValue := rand.Float64() * 100
	riskScore.Observe(riskScoreValue)

	// Detectar fraude (5% de chance)
	isFraud := riskScoreValue > 80
	if isFraud {
		fraudDetected.Inc()
		span.SetAttributes(attribute.Bool("fraud.detected", true))
		span.RecordError(fmt.Errorf("fraud detected: risk_score=%.2f", riskScoreValue))
	}

	duration := time.Since(start)
	status := "success"
	if isFraud {
		status = "fraud"
	}

	// Métricas
	messagesProcessed.WithLabelValues(status).Inc()
	processingDuration.WithLabelValues(status).Observe(duration.Seconds())
	processingDurationPercentiles.WithLabelValues(status).(prometheus.Summary).Observe(duration.Seconds())

	// Log estruturado
	logger.Info("payment_processed",
		zap.String("service", "antifraud-service"),
		zap.String("payment_id", paymentID),
		zap.Float64("amount", amount),
		zap.Float64("risk_score", riskScoreValue),
		zap.Bool("fraud_detected", isFraud),
		zap.Duration("duration_ms", duration),
		zap.String("correlation_id", correlationID),
		zap.String("trace_id", traceID),
		zap.String("level", "info"),
		zap.String("status", status),
	)
}

func handleScenarioAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int64{
			"asyncLagMs": asyncLagExtraDelay.Milliseconds(),
		})
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		AsyncLagMs int `json:"asyncLagMs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	asyncLagExtraDelay = time.Duration(payload.AsyncLagMs) * time.Millisecond
	setAsyncLagScenario(asyncLagExtraDelay > 0)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func main() {
	initLogger()
	defer logger.Sync()

	initTracing()

	rabbitURL := os.Getenv("RABBIT_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@rabbitmq:5672/"
	}

	conn, err := connectRabbitMQWithRetry(rabbitURL)
	if err != nil {
		logger.Fatal("failed_to_connect_rabbitmq", zap.Error(err))
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		logger.Fatal("failed_to_open_channel", zap.Error(err))
	}
	defer ch.Close()

	ch.ExchangeDeclare("payments", "fanout", true, false, false, false, nil)

	q, err := ch.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		logger.Fatal("failed_to_declare_queue", zap.Error(err))
	}

	ch.QueueBind(q.Name, "", "payments", false, nil)

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		logger.Fatal("failed_to_register_consumer", zap.Error(err))
	}

	// Expor métricas
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.HandleFunc("/admin/scenarios", handleScenarioAdmin)
		http.ListenAndServe(":8080", nil)
	}()

	logger.Info("antifraud-service ready",
		zap.String("service", "antifraud-service"),
		zap.String("version", "1.0.0"),
	)

	for msg := range msgs {
		ctx, _ := extractTraceContext(msg.Headers)
		processPayment(ctx, msg.Body, msg.Headers)
	}
}

func connectRabbitMQWithRetry(rabbitURL string) (*amqp.Connection, error) {
	var lastErr error
	for attempt := 1; attempt <= 30; attempt++ {
		conn, err := amqp.Dial(rabbitURL)
		if err == nil {
			return conn, nil
		}

		lastErr = err
		logger.Warn("failed_to_connect_rabbitmq_retrying",
			zap.Int("attempt", attempt),
			zap.Error(err),
		)
		time.Sleep(2 * time.Second)
	}

	return nil, fmt.Errorf("rabbitmq unavailable after retries: %w", lastErr)
}
