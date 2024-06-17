package response

import "fmt"

type BaseResponse struct {
	Ok    bool   `json:"ok"`
	Error string `json:"description,omitempty"`
}

func Ok() BaseResponse {
	return BaseResponse{
		Ok: true,
	}
}

func ErrorMsg(msg string) BaseResponse {
	return BaseResponse{
		Ok:    false,
		Error: msg,
	}
}

func Error(err error) BaseResponse {
	return BaseResponse{
		Ok:    false,
		Error: fmt.Sprint(err),
	}
}
