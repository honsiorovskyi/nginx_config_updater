package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type JSONResponse struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

func http_ok(w http.ResponseWriter, data interface{}) {
	w.WriteHeader(http.StatusTeapot)
	w.Header().Set("Content-Type", "application/json")

	if data != nil {
		body, _ := json.Marshal(JSONResponse{"ok", data})
		w.Write(body)
	}
}

func http_err(w http.ResponseWriter, message string, err error) {
	if err != nil {
		log.Println(err)
	}

	w.WriteHeader(http.StatusTeapot)
	w.Header().Set("Content-Type", "application/json")

	body, _ := json.Marshal(JSONResponse{"error", message})
	w.Write(body)
}
