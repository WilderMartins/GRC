package middleware

import (
	"strconv"
	"time"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestCounter *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	AppInfo             *prometheus.GaugeVec
	UsersCreated        *prometheus.CounterVec
	RisksCreated        prometheus.Counter
	AssessmentsUpdated  prometheus.Counter
)

func init() {
	appVersion := os.Getenv("APP_VERSION")
	if appVersion == "" {
		appVersion = "unknown"
	}

	HTTPRequestCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "phoenixgrc_http_requests_total",
			Help: "Total number of HTTP requests processed.",
		},
		[]string{"method", "path", "status_code"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "phoenixgrc_http_request_duration_seconds",
			Help:    "Histogram of HTTP request latencies.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	AppInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "phoenixgrc_app_info",
			Help: "Information about the Phoenix GRC application.",
		},
		[]string{"version"},
	)
	AppInfo.With(prometheus.Labels{"version": appVersion}).Set(1)

	UsersCreated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "phoenixgrc_users_created_total",
			Help: "Total number of users created.",
		},
		[]string{"source"},
	)

	RisksCreated = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "phoenixgrc_risks_created_total",
			Help: "Total number of risks created.",
		},
	)

	AssessmentsUpdated = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "phoenixgrc_assessments_updated_total",
			Help: "Total number of audit assessments created or updated.",
		},
	)
}

// Metrics é um middleware Gin para coletar métricas do Prometheus.
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next() // Processa a requisição

		duration := time.Since(start)
		statusCode := c.Writer.Status()

		// Ignorar métricas para o endpoint /metrics para não poluir os dados
		if c.Request.URL.Path == "/metrics" {
			return
		}

		// Usar c.FullPath() para agrupar rotas parametrizadas (ex: /users/:id)
		path := c.FullPath()
		if path == "" {
			path = "unmatched_route"
		}


		// Contadores e Histogramas
		HTTPRequestDuration.WithLabelValues(c.Request.Method, path).Observe(duration.Seconds())
		HTTPRequestCounter.WithLabelValues(c.Request.Method, path, strconv.Itoa(statusCode)).Inc()
	}
}
