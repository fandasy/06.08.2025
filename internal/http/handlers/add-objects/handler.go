package add_objects

import (
	"errors"
	"github.com/fandasy/06.08.2025/internal/http/middlewares/logger"
	"github.com/fandasy/06.08.2025/internal/pkg/api/response"
	"github.com/fandasy/06.08.2025/internal/services/archiver"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
)

type Request struct {
	Urls []string `json:"urls"`
}

type Response struct {
	Urls []string `json:"urls"`
}

func New(archiverService archiver.Archiver, validExtension []string, log *slog.Logger) gin.HandlerFunc {
	const fn = "handler.add_objects.New"

	log = log.With("fn", fn)

	var validExtensionMap map[string]struct{}

	if validExtension != nil && len(validExtension) > 0 {
		validExtensionMap = make(map[string]struct{}, len(validExtension))
		for _, ext := range validExtension {
			validExtensionMap[ext] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		requestID, ok := c.Value(logger.RequestIDKey).(string)
		if ok {
			log = log.With("request id", requestID)
		}

		taskID, ok := c.GetQuery("id")
		if !ok {
			log.Debug("Task ID missing in request parameters")

			c.JSON(http.StatusBadRequest, response.Error("Task ID missing in request parameters"))

			return
		}

		var req Request
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Error(err.Error())

			c.JSON(http.StatusBadRequest, response.Error("request body is not valid"))

			return
		}

		if len(req.Urls) == 0 {
			log.Debug("Request URLs is empty")

			c.JSON(http.StatusBadRequest, response.Error("urls is empty"))

			return
		}

		urls := extensionValidate(req.Urls, validExtensionMap)

		if len(urls) == 0 {
			log.Debug("No valid URLs")

			c.JSON(http.StatusBadRequest, response.Error("no valid urls"))

			return
		}

		added, err := archiverService.AddObjects(taskID, urls)
		if err != nil {
			switch {
			case errors.Is(err, archiver.ErrServiceStopped):
				c.JSON(http.StatusServiceUnavailable, response.Error("Archiver service is stopped"))

				return

			case errors.Is(err, archiver.ErrTaskNotFound):
				log.Warn(err.Error(), slog.String("task id", taskID))

				c.JSON(http.StatusNotFound, response.Error("Task not found"))

				return

			case errors.Is(err, archiver.ErrTaskInProgress):
				log.Info(err.Error(), slog.String("task id", taskID))

				c.JSON(http.StatusBadRequest, response.Error("Task is in progress"))

				return

			case errors.Is(err, archiver.ErrTaskCompleted):
				log.Info(err.Error(), slog.String("task id", taskID))

				c.JSON(http.StatusBadRequest, response.Error("Task is completed"))

				return

			default:
				log.Error(err.Error())

				c.JSON(http.StatusInternalServerError, response.InternalServerError())

				return
			}
		}

		addedUrls := urls[:added]

		log.Info("Urls successfully added to task", slog.String("taskID", taskID), slog.Any("urls", addedUrls))

		c.JSON(http.StatusOK, Response{addedUrls})
	}
}

func extensionValidate(urls []string, valid map[string]struct{}) []string {
	if valid == nil {
		return urls
	}

	out := make([]string, 0, len(urls))
	for _, u := range urls {
		parsedURL, err := url.Parse(u)
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			continue
		}

		if _, ok := valid[filepath.Ext(u)]; ok {
			out = append(out, u)
		}
	}
	return out
}
