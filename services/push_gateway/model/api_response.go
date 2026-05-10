package model

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type PagedData struct {
	Items    interface{} `json:"items"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

const (
	CodeOK                  = 0
	CodeResourceNotFound    = 40001
	CodeInvalidParam        = 40002
	CodeInvalidChannel      = 40003
	CodeRedisUnavailable    = 50001
	CodePostgresUnavailable = 50002
	CodeInternalError       = 50003
)

func Ok(data interface{}) Response {
	return Response{Code: CodeOK, Message: "ok", Data: data}
}

func Err(code int, msg string) Response {
	return Response{Code: code, Message: msg, Data: nil}
}
