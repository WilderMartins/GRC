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

	// AppVersion é a versão da aplicação, carregada de config.Cfg.AppVersion.
	AppVersion = "unknown"
)

func init() {
	// config.Cfg já deve estar carregado devido ao import de config em outros lugares
	// ou pelo seu próprio init(). Se AppVersion estiver lá, será usado.
	// Se config.Cfg não estiver pronto aqui (improvável se a ordem de init for bem gerenciada),
	// AppVersion permanecerá "unknown" ou o valor de os.Getenv("APP_VERSION").
	// A inicialização de AppVersion aqui é um fallback.
	// O ideal é que config.LoadConfig() seja chamado antes de qualquer lógica que dependa de Cfg.

	// Acessar config.Cfg diretamente aqui pode ser problemático se a ordem de init() dos pacotes
	// não garantir que config.LoadConfig() já rodou.
	// Uma forma mais segura seria ter uma função GetAppVersion() em config que retorna Cfg.AppVersion.
	// Por ora, vamos confiar que config.Cfg.AppVersion estará populado.
	if config.Cfg.AppVersion != "" {
		AppVersion = config.Cfg.AppVersion
	} else {
		// Fallback para variável de ambiente direta se config.Cfg.AppVersion não estiver setado
		// (isso pode acontecer se config.LoadConfig() ainda não rodou ou APP_VERSION não estava no .env)
		envVersion := os.Getenv("APP_VERSION")
		if envVersion != "" {
			AppVersion = envVersion
		}
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
