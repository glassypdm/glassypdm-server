package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type DefaultSuccessOutput struct {
	Message string `json:"message"`
}

type APIError string

const (
	GenericError           APIError = "generic error"
	BadJson                APIError = "bad json"
	IncorrectParams        APIError = "incorrect format"
	DbError                APIError = "db error"
	NoPermission           APIError = "no permission"
	insufficientPermission APIError = "insufficient permission"
)

func PrintResponse(w http.ResponseWriter, response string, output string) {
	fmt.Fprintf(w, `{
	"response": "%v",
	"body": %v
	}`, response, output)
}

func WriteCustomError(w http.ResponseWriter, err string) {
	fmt.Fprintf(w, `{
			"response": "error",
			"error": "%s"
			}`,
		err)
}

func WriteError(w http.ResponseWriter, err APIError) {
	fmt.Fprintf(w, `{
			"response": "error",
			"error": "%s"
			}`,
		err)
}

func WriteSuccess(w http.ResponseWriter, output string) {
	fmt.Fprintf(w, `{
			"response": "success",
			"body": %v
			}`,
		output)
}

func WriteDefaultSuccess(w http.ResponseWriter, msg string) {
	output := DefaultSuccessOutput{Message: msg}
	output_bytes, _ := json.Marshal(output)

	WriteSuccess(w, string(output_bytes))
}
