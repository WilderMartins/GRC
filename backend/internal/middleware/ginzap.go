package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// GinZap retorna um gin.HandlerFunc (middleware) que loga requisições usando o logger zap.
// Baseado em https://github.com/gin-contrib/zap (mas simplificado e adaptado)
func GinZap(logger *zap.Logger, timeFormat string, utc bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		// Processar requisição
		c.Next()

		// Logar após a requisição ser processada
		end := time.Now()
		latency := end.Sub(start)
		if utc {
			end = end.UTC()
		}

		fields := []zapcore.Field{
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Duration("latency", latency),
		}

		if timeFormat != "" {
			fields = append(fields, zap.String("time", end.Format(timeFormat)))
		}

		// Adicionar erros do Gin, se houver
		if len(c.Errors) > 0 {
			// Apenas logar o último erro para simplicidade, ou iterar e logar todos
			// Para múltiplos erros, c.Errors.ByType(gin.ErrorTypePrivate) pode ser usado.
			// Ou apenas c.Errors.String()
			for _, e := range c.Errors.Errors() {
				logger.Error("Request error", append(fields, zap.String("error", e))...)
			}
		} else {
			if c.Writer.Status() >= 500 {
				logger.Error("Server error", fields...)
			} else if c.Writer.Status() >= 400 {
				logger.Warn("Client error", fields...)
			} else {
				logger.Info("Request processed", fields...)
			}
		}
	}
}

// GinRecovery retorna um gin.HandlerFunc (middleware) para recovery de panics com zap.
// Loga o panic com stacktrace e retorna um erro 500.
func GinRecovery(logger *zap.Logger, timeFormat string, utc bool, recovery bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Checar se a conexão foi quebrada, nesse caso não fazer nada (o cliente fechou)
				// TODO: Esta checagem pode precisar ser mais robusta dependendo do OS e network conditions.
				// var brokenPipe bool
				// if ne, ok := err.(*net.OpError); ok {
				// 	if se, ok := ne.Err.(*os.SyscallError); ok {
				// 		if strings.Contains(strings.ToLower(se.Error()), "broken pipe") || strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
				// 			brokenPipe = true
				// 		}
				// 	}
				// }

				// httpRequest, _ := httputil.DumpRequest(c.Request, false)
				// headers := strings.Split(string(httpRequest), "\r\n")
				// for idx, header := range headers {
				// 	current := strings.Split(header, ":")
				// 	if current[0] == "Authorization" {
				// 		headers[idx] = current[0] + ": ***REDACTED***"
				// 	}
				// }
				// headersToStr := strings.Join(headers, "\r\n")

				// Simplificado por agora:
				logger.Error("Panic recovered",
					zap.Any("error", err),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("ip", c.ClientIP()),
					zap.String("user_agent", c.Request.UserAgent()),
					zap.Stack("stacktrace"), // Adiciona o stack trace
				)

				// Retornar um erro 500 genérico se a flag de recovery estiver ativa
				if recovery {
					c.AbortWithStatusJSON(500, gin.H{"error": "Internal Server Error - Panic Recovered"})
				}
			}
		}()
		c.Next()
	}
}
