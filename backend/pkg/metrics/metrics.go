package metrics

import (
	"os"
	"phoenixgrc/backend/internal/config" // Para obter a versão da app, se disponível

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTPRequestCounter conta o total de requisições HTTP.
	HTTPRequestCounter *prometheus.CounterVec

	// HTTPRequestDuration observa a duração das requisições HTTP.
	HTTPRequestDuration *prometheus.HistogramVec

	// AppInfo expõe informações sobre a aplicação.
	AppInfo *prometheus.GaugeVec

	// AppVersion é um placeholder para a versão da aplicação.
	// Idealmente, isso seria injetado durante o build.
	AppVersion = "0.1.0-snapshot" // TODO: Injetar via ldflags no build
)

func init() {
	// Carregar a versão da app de uma variável de ambiente se existir,
	// caso contrário, usar o placeholder.
	versionFromEnv := os.Getenv("APP_VERSION")
	if versionFromEnv != "" {
		AppVersion = versionFromEnv
	}
	// Alternativamente, se a config.Cfg já estiver carregada e tiver um campo de versão:
	if config.Cfg.AppVersion != "" { // Supondo que AppVersion exista em config.Cfg
		AppVersion = config.Cfg.AppVersion
	}


	HTTPRequestCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "phoenixgrc_http_requests_total",
			Help: "Total number of HTTP requests processed.",
		},
		[]string{"method", "path", "status_code"}, // Labels
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "phoenixgrc_http_request_duration_seconds",
			Help: "Histogram of HTTP request latencies.",
			// Buckets podem ser ajustados conforme necessário
			Buckets: prometheus.DefBuckets, // prometheus.ExponentialBuckets(0.001, 2, 15) etc.
		},
		[]string{"method", "path"}, // Labels
	)

	AppInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "phoenixgrc_app_info",
			Help: "Information about the Phoenix GRC application.",
		},
		[]string{"version"}, // Label para a versão
	)
	AppInfo.With(prometheus.Labels{"version": AppVersion}).Set(1)

	// Registrar métricas padrão do Go (opcional, mas útil)
	// promauto.MustRegister(collectors.NewBuildInfoCollector()) // Requer import de collectors
	// promauto.MustRegister(collectors.NewGoCollector())
	// Nota: promauto já registra automaticamente no DefaultRegisterer.
	// Se usar NewGoCollector, etc., diretamente, eles precisam ser registrados
	// com prometheus.MustRegister(prometheus.NewGoCollector())
}
