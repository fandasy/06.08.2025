package zips_download

import (
	"github.com/fandasy/06.08.2025/internal/http/middlewares/logger"
	"github.com/fandasy/06.08.2025/internal/pkg/api/response"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
)

func New(zipsDir string, log *slog.Logger) gin.HandlerFunc {
	const fn = "handlers.zips_download.New"

	log = log.With("fn", fn)

	return func(c *gin.Context) {
		requestID, ok := c.Value(logger.RequestIDKey).(string)
		if ok {
			log = log.With("request id", requestID)
		}

		filename := c.Param("filename")

		filePath := filepath.Join(zipsDir, filepath.Base(filename))

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Warn("File not found", slog.String("filename", filename))

			c.JSON(http.StatusNotFound, response.Error("File not found"))

			return
		}

		c.Header("Content-Type", "application/zip")

		c.FileAttachment(filePath, filename)
	}
}
