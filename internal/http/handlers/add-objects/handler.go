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
	Added int   `json:"added"`
	Urls  []Url `json:"urls,omitempty"`
}

type Url struct {
	Value string `json:"url"`
	Err   string `json:"error,omitempty"`
}

// New godoc
// @Summary      Добавить объекты в задачу архивации
// @Description  Добавляет один или несколько файловых URL в существующую задачу архивации.
// @Tags         tasks
// @Accept       json
// @Produce      json
// @Param        id   path      string      true  "ID задачи"
// @Param        request  body  Request     true  "Список URL-адресов для добавления"  example({"urls": ["https://example.com/file1.pdf", "https://example.com/image1.jpeg"]})
// @Success      200  {object}  Response    "Ссылки успешно добавлены в задачу"
// @Failure      400  {object}  response.ErrorResponse "Некорректный запрос"
// @Failure      400  {object}  response.ErrorResponse "Параметр taskID отсутствует"
// @Failure      400  {object}  response.ErrorResponse "Тело запроса невалидно (не JSON)"
// @Failure      400  {object}  response.ErrorResponse "Список URL пуст ('urls is empty')"
// @Failure      400  {object}  response.ErrorResponse "Нет поддерживаемых URL ('no valid urls')"
// @Failure      400  {object}  response.ErrorResponse "Задача уже в обработке ('Task is in progress')"
// @Failure      400  {object}  response.ErrorResponse "Задача уже завершена ('Task is completed')"
// @Failure      404  {object}  response.ErrorResponse "Задача не найдена ('Task not found')"
// @Failure      503  {object}  response.ErrorResponse "Сервис архивации остановлен"
// @Failure      500  {object}  response.ErrorResponse "Внутренняя ошибка сервера"
// @Example      {json}  Успешный запрос:
//
//	{
//	  "urls": [
//	    "https://example.com/file1.pdf",
//	    "https://example.com/file2.jpeg"
//	  ]
//	}
//
// @Example      {json}  Успешный ответ:
//
//	{
//	  "added": 2,
//	  "urls": [
//	    {"url": "https://example.com/file1.pdf"},
//	    {"url": "https://example.com/image1.jpeg"}
//	  ]
//	}
//
// @Example      {json}  Ошибка: Параметр taskID отсутствует:
//
//	{
//	  "error": "Task ID missing in request parameters"
//	}
//
// @Example      {json}  Ошибка: Список URL пуст:
//
//	{
//	  "error": "urls is empty"
//	}
//
// @Example      {json}  Ошибка: Нет поддерживаемых URL:
//
//	{
//	  "error": "no valid urls"
//	}
//
// @Example      {json}  Ошибка: Задача уже в обработке:
//
//	{
//	  "error": "Task is in progress"
//	}
//
// @Example      {json}  Ошибка: Задача уже завершена:
//
//	{
//	  "error": "Task is completed"
//	}
//
// @Example      {json}  Ошибка: Задача не найдена:
//
//	{
//	  "error": "Task not found"
//	}
//
// @Example      {json}  Ошибка: Сервис архивации остановлен:
//
//	{
//	  "error": "Archiver service is stopped"
//	}
//
// @Router       /task/{id}/add [post]
func New(archiverService archiver.Archiver, validExtension []string, log *slog.Logger) gin.HandlerFunc {
	const fn = "handlers.add_objects.New"

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

		taskID := c.Param("id")
		if taskID == "" {
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

		var resp Response

		urls := make([]string, 0, len(req.Urls))
		for _, u := range req.Urls {
			if err := extensionValidate(u, validExtensionMap); err != nil {
				resp.Urls = append(resp.Urls, Url{Value: u, Err: err.Error()})
				continue
			}
			resp.Urls = append(resp.Urls, Url{Value: u})
			urls = append(urls, u)
		}

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

		if added < len(urls) {
			validCount := 0
			for i := range resp.Urls {
				if resp.Urls[i].Err == "" {
					validCount++
					if validCount > added {
						resp.Urls[i].Err = ErrNoMorePlacesAvailable.Error()
					}
				}
			}
		}

		resp.Added = added

		log.Info("Urls successfully added to task", slog.String("task id", taskID), slog.Any("urls", resp.Urls))

		c.JSON(http.StatusOK, resp)
	}
}

var (
	ErrIncorrectUrl          = errors.New("incorrect url")
	ErrInvalidExtension      = errors.New("invalid extension")
	ErrNoMorePlacesAvailable = errors.New("no more places available")
)

func extensionValidate(u string, valid map[string]struct{}) error {
	parsedURL, err := url.Parse(u)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return ErrIncorrectUrl
	}

	if valid != nil {
		if _, ok := valid[filepath.Ext(u)]; !ok {
			return ErrInvalidExtension
		}
	}

	return nil
}
