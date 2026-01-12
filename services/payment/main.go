package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// ============================================================================
// ESTRUTURAS E CONFIGURAÇÕES
// ============================================================================

type PaymentRequest struct {
	AccountID string  `json:"accountId"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
}

type PaymentResponse struct {
	PaymentID   string    `json:"paymentId"`
	Status      string    `json:"status"`
	ProcessedAt time.Time `json:"processedAt"`
}

// Lag Controller - Para simular lag intencional e reproduzível
type LagController struct {
	mu            sync.RWMutex
	enabled       bool
	databaseDelay time.Duration
	cacheDelay    time.Duration
	externalDelay time.Duration
	lagPercentage float64 // Porcentagem de requisições afetadas (0.0 a 1.0)
}

var lagController = &LagController{
	enabled:       false,
	databaseDelay: 2 * time.Second,
	cacheDelay:    500 * time.Millisecond,
	externalDelay: 1 * time.Second,
	lagPercentage: 1.0, // 100% das requisições por padrão quando habilitado
}

func (lc *LagController) SetEnabled(enabled bool) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.enabled = enabled
}

func (lc *LagController) IsEnabled() bool {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	return lc.enabled
}

func (lc *LagController) SetDatabaseDelay(delay time.Duration) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.databaseDelay = delay
}

func (lc *LagController) SetCacheDelay(delay time.Duration) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.cacheDelay = delay
}

func (lc *LagController) SetExternalDelay(delay time.Duration) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.externalDelay = delay
}

func (lc *LagController) SetLagPercentage(percentage float64) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	if percentage < 0.0 {
		percentage = 0.0
	}
	if percentage > 1.0 {
		percentage = 1.0
	}
	lc.lagPercentage = percentage
}

func (lc *LagController) GetDatabaseDelay() time.Duration {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	return lc.databaseDelay
}

func (lc *LagController) GetCacheDelay() time.Duration {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	return lc.cacheDelay
}

func (lc *LagController) GetExternalDelay() time.Duration {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	return lc.externalDelay
}

func (lc *LagController) ShouldApplyLag() bool {
	lc.mu.RLock()
	defer lc.mu.RUnlock()
	if !lc.enabled {
		return false
	}
	return rand.Float64() < lc.lagPercentage
}

// Rate Limiter simples (backpressure)
type RateLimiter struct {
	mu         sync.Mutex
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
}

func NewRateLimiter(maxTokens int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill)
	tokensToAdd := int(elapsed / rl.refillRate)

	if tokensToAdd > 0 {
		newTokens := rl.tokens + tokensToAdd
		if newTokens > rl.maxTokens {
			rl.tokens = rl.maxTokens
		} else {
			rl.tokens = newTokens
		}
		rl.lastRefill = now
	}

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	return false
}

// Circuit Breaker simples
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

type CircuitBreaker struct {
	mu              sync.Mutex
	state           CircuitState
	failures        int
	maxFailures     int
	successes       int
	halfOpenSuccess int
	lastFailure     time.Time
	timeout         time.Duration
}

func NewCircuitBreaker(maxFailures int, timeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:       CircuitClosed,
		maxFailures: maxFailures,
		timeout:     timeout,
	}
}

func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	state := cb.state
	cb.mu.Unlock()

	if state == CircuitOpen {
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.mu.Lock()
			cb.state = CircuitHalfOpen
			cb.successes = 0
			cb.mu.Unlock()
		} else {
			return fmt.Errorf("circuit breaker is open")
		}
	}

	err := fn()
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()
		if cb.failures >= cb.maxFailures {
			cb.state = CircuitOpen
		}
		return err
	}

	cb.successes++
	if cb.state == CircuitHalfOpen {
		if cb.successes >= cb.halfOpenSuccess {
			cb.state = CircuitClosed
			cb.failures = 0
		}
	} else {
		cb.failures = 0
	}

	return nil
}

// ============================================================================
// MÉTRICAS PROMETHEUS (RED + USE)
// ============================================================================

var (
	// RED Metrics (Rate, Errors, Duration)
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
		},
		[]string{"method", "endpoint"},
	)

	httpRequestDurationPercentiles = promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "http_request_duration_percentiles",
			Help:       "HTTP request duration percentiles (p50, p90, p95, p99)",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.01, 0.99: 0.001},
		},
		[]string{"method", "endpoint"},
	)

	// USE Metrics (Utilization, Saturation, Errors)
	cpuUtilization = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "cpu_utilization_percent",
			Help: "CPU utilization percentage",
		},
	)

	memoryUtilization = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "memory_utilization_bytes",
			Help: "Memory utilization in bytes",
		},
	)

	queueDepth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "message_queue_depth",
			Help: "Current depth of message queue (saturation)",
		},
	)

	rateLimitRejected = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "rate_limit_rejected_total",
			Help: "Total requests rejected by rate limiter (backpressure)",
		},
	)

	circuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
		},
		[]string{"service"},
	)

	// Business Metrics
	paymentsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payments_processed_total",
			Help: "Total payments processed",
		},
		[]string{"status", "currency"},
	)

	paymentAmount = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "payment_amount",
			Help:    "Payment amount distribution",
			Buckets: []float64{10, 50, 100, 500, 1000, 5000, 10000},
		},
		[]string{"currency"},
	)

	// Métricas específicas para lag intencional
	lagEnabled = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "intentional_lag_enabled",
			Help: "Whether intentional lag is enabled (1=enabled, 0=disabled)",
		},
	)

	lagDatabaseDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "intentional_lag_database_duration_seconds",
			Help:    "Duration of intentional database lag",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 3.0, 5.0, 10.0},
		},
	)

	lagCacheDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "intentional_lag_cache_duration_seconds",
			Help:    "Duration of intentional cache lag",
			Buckets: []float64{0.1, 0.25, 0.5, 1.0, 2.0, 5.0},
		},
	)

	lagExternalDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "intentional_lag_external_duration_seconds",
			Help:    "Duration of intentional external call lag",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0},
		},
	)
)

// ============================================================================
// TRACING E LOGGING
// ============================================================================

var logger *zap.Logger
var tracer trace.Tracer

func initTracing() {
	jaegerEndpoint := os.Getenv("JAEGER_ENDPOINT")

	if jaegerEndpoint == "" {
		logger.Warn("No Jaeger endpoint configured, tracing disabled")
		return
	}

	// Jaeger exporter usando agent endpoint (padrão: localhost:6831)
	exporter, err := jaeger.New(jaeger.WithAgentEndpoint())
	if err != nil {
		logger.Error("Failed to create Jaeger exporter", zap.Error(err))
		return
	}
	logger.Info("Using Jaeger for tracing", zap.String("endpoint", jaegerEndpoint))

	// Sampling: 10% das requisições (head-based sampling)
	sampler := tracesdk.TraceIDRatioBased(0.1)

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exporter),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("payment-service"),
		)),
		tracesdk.WithSampler(sampler),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer = otel.Tracer("payment-service")
}

func initLogger() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}
}

// ============================================================================
// MIDDLEWARE E HANDLERS
// ============================================================================

func generateID() string {
	return fmt.Sprintf("pay-%d-%d", time.Now().UnixNano(), rand.Intn(10000))
}

func extractTraceContext(r *http.Request) (trace.SpanContext, context.Context) {
	ctx := r.Context()
	propagator := otel.GetTextMapPropagator()
	ctx = propagator.Extract(ctx, propagation.HeaderCarrier(r.Header))

	ctx, span := tracer.Start(ctx, "payment.process")
	return span.SpanContext(), ctx
}

func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Extrair correlation ID e trace ID
		correlationID := r.Header.Get("X-Correlation-ID")
		if correlationID == "" {
			correlationID = generateID()
		}

		traceCtx, ctx := extractTraceContext(r)
		traceID := traceCtx.TraceID().String()
		spanID := traceCtx.SpanID().String()

		// Adicionar ao contexto
		ctx = context.WithValue(ctx, "correlation_id", correlationID)
		ctx = context.WithValue(ctx, "trace_id", traceID)

		// Criar response writer customizado para capturar status
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Executar handler
		next(rw, r.WithContext(ctx))

		// Log estruturado
		duration := time.Since(start)
		logger.Info("http_request",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", rw.statusCode),
			zap.Duration("duration_ms", duration),
			zap.String("correlation_id", correlationID),
			zap.String("trace_id", traceID),
			zap.String("span_id", spanID),
			zap.String("service", "payment-service"),
			zap.String("level", getLogLevel(rw.statusCode)),
		)

		// Métricas RED
		httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path, fmt.Sprintf("%d", rw.statusCode)).Inc()
		httpRequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration.Seconds())
		httpRequestDurationPercentiles.WithLabelValues(r.Method, r.URL.Path).(prometheus.Summary).Observe(duration.Seconds())
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func getLogLevel(statusCode int) string {
	if statusCode >= 500 {
		return "error"
	} else if statusCode >= 400 {
		return "warn"
	}
	return "info"
}

// Simular gargalo: banco de dados lento
func simulateDatabaseDelay(ctx context.Context) error {
	start := time.Now()
	_, span := tracer.Start(ctx, "database.query")
	defer span.End()

	baseDelay := 10 * time.Millisecond

	// Lag intencional tem prioridade sobre lag aleatório
	if lagController.ShouldApplyLag() {
		delay := lagController.GetDatabaseDelay()
		span.SetAttributes(
			attribute.Bool("lag.intentional", true),
			attribute.String("lag.type", "database"),
			attribute.Int64("lag.duration_ms", delay.Milliseconds()),
		)
		time.Sleep(delay)
		duration := time.Since(start)
		lagDatabaseDuration.Observe(duration.Seconds())
		logger.Warn("intentional_lag_database",
			zap.String("lag.type", "database"),
			zap.Duration("lag.duration", delay),
			zap.Duration("actual.duration", duration),
		)
		span.RecordError(fmt.Errorf("intentional lag: database delay %v", delay))
		return nil
	}

	// Simular latência variável (cauda longa) - comportamento original
	// 1% das requisições tem latência alta (cauda)
	if rand.Float64() < 0.01 {
		delay := baseDelay + time.Duration(rand.Intn(2000))*time.Millisecond
		time.Sleep(delay)
		span.RecordError(fmt.Errorf("slow query detected: %v", delay))
	} else {
		time.Sleep(baseDelay)
	}
	return nil
}

// Simular gargalo: cache miss
func simulateCacheLookup(ctx context.Context) bool {
	start := time.Now()
	_, span := tracer.Start(ctx, "cache.lookup")
	defer span.End()

	// Lag intencional tem prioridade
	if lagController.ShouldApplyLag() {
		delay := lagController.GetCacheDelay()
		span.SetAttributes(
			attribute.Bool("lag.intentional", true),
			attribute.String("lag.type", "cache"),
			attribute.Int64("lag.duration_ms", delay.Milliseconds()),
			attribute.Bool("cache.hit", false), // Lag simula cache miss
		)
		time.Sleep(delay)
		duration := time.Since(start)
		lagCacheDuration.Observe(duration.Seconds())
		logger.Warn("intentional_lag_cache",
			zap.String("lag.type", "cache"),
			zap.Duration("lag.duration", delay),
			zap.Duration("actual.duration", duration),
		)
		return false
	}

	// 80% cache hit, 20% miss (gargalo) - comportamento original
	if rand.Float64() < 0.8 {
		span.SetAttributes(attribute.Bool("cache.hit", true))
		return true
	}
	span.SetAttributes(attribute.Bool("cache.hit", false))
	time.Sleep(50 * time.Millisecond) // Simular cache miss
	return false
}

func handlePayment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	correlationID := ctx.Value("correlation_id").(string)
	traceID := ctx.Value("trace_id").(string)

	// Rate limiting (backpressure)
	rateLimiter := NewRateLimiter(100, 1*time.Second)
	if !rateLimiter.Allow() {
		rateLimitRejected.Inc()
		logger.Warn("rate_limit_exceeded",
			zap.String("correlation_id", correlationID),
			zap.String("trace_id", traceID),
		)
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// Circuit breaker para dependências
	circuitBreaker := NewCircuitBreaker(5, 30*time.Second)
	err := circuitBreaker.Call(func() error {
		// Simular chamada a serviço externo
		if rand.Float64() < 0.05 { // 5% de falha
			return fmt.Errorf("external service error")
		}
		return nil
	})

	if err != nil {
		circuitBreakerState.WithLabelValues("external-service").Set(float64(CircuitOpen))
		logger.Error("circuit_breaker_open",
			zap.String("correlation_id", correlationID),
			zap.String("trace_id", traceID),
			zap.Error(err),
		)
		http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
		return
	}
	circuitBreakerState.WithLabelValues("external-service").Set(float64(CircuitClosed))

	// Processar pagamento
	var req PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Simular gargalos
	_ = simulateCacheLookup(ctx)
	_ = simulateDatabaseDelay(ctx)

	// Simular serviço chatty (múltiplas chamadas)
	for i := 0; i < 3; i++ {
		start := time.Now()
		_, span := tracer.Start(ctx, fmt.Sprintf("external.call.%d", i))

		// Aplicar lag intencional se habilitado
		if lagController.ShouldApplyLag() {
			delay := lagController.GetExternalDelay()
			span.SetAttributes(
				attribute.Bool("lag.intentional", true),
				attribute.String("lag.type", "external"),
				attribute.Int64("lag.duration_ms", delay.Milliseconds()),
			)
			time.Sleep(delay)
			duration := time.Since(start)
			lagExternalDuration.Observe(duration.Seconds())
		} else {
			time.Sleep(5 * time.Millisecond)
		}
		span.End()
	}

	paymentID := generateID()
	response := PaymentResponse{
		PaymentID:   paymentID,
		Status:      "PROCESSED",
		ProcessedAt: time.Now(),
	}

	// Publicar evento
	rabbitURL := os.Getenv("RABBIT_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@rabbitmq:5672/"
	}

	conn, err := amqp.Dial(rabbitURL)
	if err == nil {
		ch, _ := conn.Channel()
		ch.ExchangeDeclare("payments", "fanout", true, false, false, false, nil)

		event := map[string]interface{}{
			"event":         "PaymentCreated",
			"paymentId":     paymentID,
			"accountId":     req.AccountID,
			"amount":        req.Amount,
			"currency":      req.Currency,
			"correlationId": correlationID,
			"traceId":       traceID,
			"ts":            time.Now().UnixMilli(),
		}

		body, _ := json.Marshal(event)
		ch.Publish("payments", "", false, false, amqp.Publishing{
			Body:        body,
			ContentType: "application/json",
			Headers: amqp.Table{
				"X-Correlation-ID": correlationID,
				"X-Trace-ID":       traceID,
			},
		})
		conn.Close()
	}

	// Métricas de negócio
	paymentsProcessed.WithLabelValues("success", req.Currency).Inc()
	paymentAmount.WithLabelValues(req.Currency).Observe(req.Amount)

	// Log estruturado de negócio
	logger.Info("payment_processed",
		zap.String("payment_id", paymentID),
		zap.String("account_id", req.AccountID),
		zap.Float64("amount", req.Amount),
		zap.String("currency", req.Currency),
		zap.String("correlation_id", correlationID),
		zap.String("trace_id", traceID),
		zap.String("status", "success"),
	)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Correlation-ID", correlationID)
	w.Header().Set("X-Trace-ID", traceID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

// Handler para controlar lag intencional

// ============================================================================
// MAIN
// ============================================================================

func main() {
	initLogger()
	defer logger.Sync()

	initTracing()

	// Simular métricas USE (CPU e memória)
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for range ticker.C {
			// Simular CPU utilization (0-100%)
			cpuUtilization.Set(rand.Float64() * 100)
			// Simular memória (100MB - 500MB)
			memoryUtilization.Set(float64(100000000 + rand.Intn(400000000)))
			// Simular queue depth
			queueDepth.Set(float64(rand.Intn(1000)))
		}
	}()

	// Inicializar lag controller a partir de variáveis de ambiente
	if os.Getenv("INTENTIONAL_LAG_ENABLED") == "true" {
		lagController.SetEnabled(true)
		lagEnabled.Set(1)
		logger.Info("intentional_lag_enabled_from_env",
			zap.Bool("enabled", true),
		)
	}

	if dbDelayStr := os.Getenv("INTENTIONAL_LAG_DATABASE_MS"); dbDelayStr != "" {
		var dbDelay int
		if _, err := fmt.Sscanf(dbDelayStr, "%d", &dbDelay); err == nil {
			lagController.SetDatabaseDelay(time.Duration(dbDelay) * time.Millisecond)
		}
	}

	if cacheDelayStr := os.Getenv("INTENTIONAL_LAG_CACHE_MS"); cacheDelayStr != "" {
		var cacheDelay int
		if _, err := fmt.Sscanf(cacheDelayStr, "%d", &cacheDelay); err == nil {
			lagController.SetCacheDelay(time.Duration(cacheDelay) * time.Millisecond)
		}
	}

	if extDelayStr := os.Getenv("INTENTIONAL_LAG_EXTERNAL_MS"); extDelayStr != "" {
		var extDelay int
		if _, err := fmt.Sscanf(extDelayStr, "%d", &extDelay); err == nil {
			lagController.SetExternalDelay(time.Duration(extDelay) * time.Millisecond)
		}
	}

	http.HandleFunc("/payments", loggingMiddleware(handlePayment))
	http.HandleFunc("/health", handleHealth)
	http.Handle("/metrics", promhttp.Handler())

	logger.Info("payment-service listening on :8080",
		zap.String("service", "payment-service"),
		zap.String("version", "1.0.0"),
	)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}
