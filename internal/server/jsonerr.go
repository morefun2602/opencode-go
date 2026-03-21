package server

import (
	"encoding/json"
	"net/http"
)

type errBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func writeJSONErr(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	var b errBody
	b.Error.Code = code
	b.Error.Message = msg
	_ = json.NewEncoder(w).Encode(b)
}
