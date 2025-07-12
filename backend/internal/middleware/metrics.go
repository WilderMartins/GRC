package middleware

import (
	"strconv"
	"time"

	phxmetrics "phoenixgrc/backend/pkg/metrics" // Importar o pacote de métricas

	"github.com/gin-gonic/gin"
)

// Metrics é um middleware Gin para coletar métricas Prometheus para requisições HTTP.
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Processar a requisição
		c.Next()

		// Coletar métricas após a requisição ser processada
		status := c.Writer.Status()
		method := c.Request.Method

		// Usar c.FullPath() para obter o template da rota, o que é melhor para cardinalidade de labels.
		// Se FullPath() estiver vazio (ex: rota não encontrada), usar o Path.
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		latency := time.Since(start)

		// Incrementar contador de requisições
		if phxmetrics.HTTPRequestCounter != nil {
			phxmetrics.HTTPRequestCounter.WithLabelValues(method, path, strconv.Itoa(status)).Inc()
		}

		// Observar duração da requisição
		if phxmetrics.HTTPRequestDuration != nil {
			phxmetrics.HTTPRequestDuration.WithLabelValues(method, path).Observe(latency.Seconds())
		}
	}
}
