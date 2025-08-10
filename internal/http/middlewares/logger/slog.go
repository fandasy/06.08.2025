package logger

import (
	fast_id "github.com/fandasy/06.08.2025/pkg/fast-id"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"log/slog"
	"time"
)

const RequestIDKey = "request-id-key"

func Middleware(log *slog.Logger) gin.HandlerFunc {
	fn := func(c *gin.Context) {

		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		id := newID()

		c.Set(RequestIDKey, id)

		// Process request
		c.Next()

		// Stop timer
		TimeStamp := time.Now()
		Latency := TimeStamp.Sub(start)

		Method := c.Request.Method
		StatusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		log.Info("[SLOG]",
			slog.String("method", Method),
			slog.String("path", path),
			slog.String("request id", id),
			slog.String("time", Latency.String()),
			slog.Int("status", StatusCode),
		)
	}

	return fn
}

func newID() string {
	var id string

	UUID, err := uuid.NewRandom()
	if err != nil {
		id = fast_id.New()
	} else {
		id = UUID.String()
	}

	return id
}
