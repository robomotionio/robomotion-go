package runtime

import "encoding/json"

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewError(code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

func (e *Error) Error() string {
	err, _ := json.Marshal(e)
	return string(err)
}
