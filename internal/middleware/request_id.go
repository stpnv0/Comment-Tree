package middleware

import (
	"github.com/wb-go/wbf/ginext"
	"github.com/wb-go/wbf/logger"
)

const requestIDHeader = "X-Request-ID"

func RequestID() ginext.HandlerFunc {
	return func(c *ginext.Context) {
		id := c.GetHeader(requestIDHeader)
		if id == "" {
			id = logger.GenerateRequestID()
		}

		ctx := logger.SetRequestID(c.Request.Context(), id)
		c.Request = c.Request.WithContext(ctx)
		c.Header(requestIDHeader, id)

		c.Next()
	}
}
