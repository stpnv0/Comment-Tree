package middleware

import (
	"net/http"
	"time"

	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/logger"
)

func RequestLogger(log logger.Logger) ginext.HandlerFunc {
	return func(c *ginext.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		level := logger.InfoLevel
		if status >= http.StatusInternalServerError {
			level = logger.ErrorLevel
		} else if status >= http.StatusBadRequest {
			level = logger.WarnLevel
		}

		attrs := []logger.Attr{
			logger.String("method", method),
			logger.String("path", path),
			logger.Int("status", status),
			logger.Duration("latency", latency),
			logger.String("client_ip", c.ClientIP()),
		}

		if query != "" {
			attrs = append(attrs, logger.String("query", query))
		}

		if errMsg, exists := c.Get("error"); exists {
			attrs = append(attrs, logger.String("error", errMsg.(string)))
		}

		if len(c.Errors) > 0 {
			attrs = append(attrs, logger.String("gin_errors", c.Errors.String()))
		}

		log.LogAttrs(c.Request.Context(), level, "HTTP request", attrs...)
	}
}
