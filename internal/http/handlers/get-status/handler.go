package get_status

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
	Status  string    `json:"status"`
	Objects []Objects `json:"objects"`

	Zip string `json:"zip,omitempty"`
	Err string `json:"error,omitempty"`
}

type Objects struct {
	Src string `json:"src,omitempty"`
	Err string `json:"error,omitempty"`
}

// New godoc
// @Summary      Получить статус задачи архивации
// @Description  Возвращает текущий статус задачи архивации, список объектов, ошибки и ссылку на архив (если задача завершена).
// @Tags         tasks
// @Produce      json
// @Param        id   path      string  true  "ID задачи"
// @Success      200  {object}  Response  "Информация о задаче"
// @Failure      400  {object}  response.ErrorResponse "Параметр taskID отсутствует"
// @Failure      404  {object}  response.ErrorResponse "Задача не найдена"
// @Failure      503  {object}  response.ErrorResponse "Сервис архивации остановлен"
// @Failure      500  {object}  response.ErrorResponse "Внутренняя ошибка сервера"
// @Example      {json}  Успешный ответ:
//
//	{
//	  "status": "Done",
//	  "objects": [
//	    { "src": "https://example.com/file1.pdf" },
//	    { "src": "https://example.com/file2.jpeg", "error": "file not found" }
//	  ],
//	  "zip": "http://localhost:8080/storage/12345.zip",
//	  "error": ""
//	}
//
// @Example      {json}  Ошибка: Параметр taskID отсутствует:
//
//	{
//	  "error": "Task ID missing in request parameters"
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
// @Router       /task/{id}/status [get]
func New(archiverService archiver.Archiver, log *slog.Logger) gin.HandlerFunc {
	const fn = "handlers.get_status.New"

	log = log.With("fn", fn)

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

		taskInfo, err := archiverService.GetStatus(taskID)
		if err != nil {
			switch {
			case errors.Is(err, archiver.ErrServiceStopped):
				c.JSON(http.StatusServiceUnavailable, response.Error("Archiver service is stopped"))

				return

			case errors.Is(err, archiver.ErrTaskNotFound):
				log.Warn(err.Error(), slog.String("task id", taskID))

				c.JSON(http.StatusNotFound, response.Error("Task not found"))

				return

			default:
				log.Error(err.Error())

				c.JSON(http.StatusInternalServerError, response.InternalServerError())

				return
			}
		}

		log.Info("Information about the task has been received", slog.String("task id", taskID), slog.Any("info", taskInfo))

		objs := make([]Objects, 0, len(taskInfo.Objects))
		for _, obj := range taskInfo.Objects {
			var objErr string
			if obj.Err != nil {
				objErr = obj.Err.Error()
			}

			objs = append(objs, Objects{
				Src: obj.Src,
				Err: objErr,
			})
		}

		var taskErr string
		if taskInfo.Err != nil {
			taskErr = taskInfo.Err.Error()
		}

		resp := Response{
			Status:  taskInfo.Status.String(),
			Objects: objs,
			Zip:     taskInfo.Zip,
			Err:     taskErr,
		}

		c.JSON(http.StatusOK, resp)
	}
}
