package response

type ErrorResponse struct {
	Err string `json:"error"`
}

func Error(msg string) ErrorResponse {
	return ErrorResponse{msg}
}

const internalServerErrorMsg = "Internal Server Error"

func InternalServerError() ErrorResponse {
	return ErrorResponse{internalServerErrorMsg}
}
