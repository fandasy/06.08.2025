package new_task

import (
	"errors"
	"github.com/fandasy/06.08.2025/internal/http/middlewares/logger"
	"github.com/fandasy/06.08.2025/internal/pkg/api/response"
	"github.com/fandasy/06.08.2025/internal/services/archiver"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
)

type Response struct {
	ID string `json:"id"`
}

func New(archiverService archiver.Archiver, log *slog.Logger) gin.HandlerFunc {
	const fn = "handlers.new_task.New"

	log = log.With("fn", fn)

	return func(c *gin.Context) {
		requestID, ok := c.Value(logger.RequestIDKey).(string)
		if ok {
			log = log.With("request id", requestID)
		}

		id, err := archiverService.NewTask()
		if err != nil {
			switch {
			case errors.Is(err, archiver.ErrServiceStopped):
				c.JSON(http.StatusServiceUnavailable, response.Error("Archiver service is stopped"))

				return

			case errors.Is(err, archiver.ErrMaxTasksExceeded):
				log.Warn("Maximum number of tasks exceeded")

				c.JSON(http.StatusServiceUnavailable, response.Error("Max tasks exceeded"))

				return

			default:
				log.Error(err.Error())

				c.JSON(http.StatusInternalServerError, response.InternalServerError())

				return
			}
		}

		log.Info("New task started", slog.String("task id", id))

		c.JSON(http.StatusOK, Response{ID: id})
	}
}
