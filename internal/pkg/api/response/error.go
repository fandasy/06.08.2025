package response

type ErrorResponse struct {
	Err string `json:"error"`
}

func Error(msg string) ErrorResponse {
	return ErrorResponse{msg}
}
