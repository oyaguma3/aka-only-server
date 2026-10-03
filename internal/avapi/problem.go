package avapi

import (
	"encoding/json/v2"
	"net/http"
)

// ProblemDetails の cause の値。
const (
	causeInvalidMsgFormat     = "INVALID_MSG_FORMAT"
	causeMandatoryIEMissing   = "MANDATORY_IE_MISSING"
	causeMandatoryIEIncorrect = "MANDATORY_IE_INCORRECT"
	causeOptionalIEIncorrect  = "OPTIONAL_IE_INCORRECT"
	causeAuthRejected         = "AUTHENTICATION_REJECTED"
	causeUserNotFound         = "USER_NOT_FOUND"
	causeSystemFailure        = "SYSTEM_FAILURE"
	causeAuthTypeNotSupported = "AUTH_TYPE_NOT_SUPPORTED"
)

// problemDetails は TS 29.571 の ProblemDetails のうち、このサーバーが返す項目。
type problemDetails struct {
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
	Cause  string `json:"cause"`
}

// problem はハンドラー内でエラー応答を表す。
type problem struct {
	status int
	cause  string
	detail string
}

func (p *problem) write(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(p.status)
	json.MarshalWrite(w, problemDetails{
		Title:  http.StatusText(p.status),
		Status: p.status,
		Detail: p.detail,
		Cause:  p.cause,
	})
}

// WriteForbidden は 403（AUTHENTICATION_REJECTED）の ProblemDetails を書き込む。
// クライアントを識別できなかった場合に、ハンドラーの外側から使う。
func WriteForbidden(w http.ResponseWriter, detail string) {
	(&problem{status: http.StatusForbidden, cause: causeAuthRejected, detail: detail}).write(w)
}

// WriteInternalError は 500（SYSTEM_FAILURE）の ProblemDetails を書き込む。
func WriteInternalError(w http.ResponseWriter) {
	(&problem{status: http.StatusInternalServerError, cause: causeSystemFailure}).write(w)
}

func badRequest(cause, detail string) *problem {
	return &problem{status: http.StatusBadRequest, cause: cause, detail: detail}
}
