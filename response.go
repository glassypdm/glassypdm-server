package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type DefaultSuccessOutput struct {
	Message string `json:"message"`
}

func PrintResponse(w http.ResponseWriter, response string, output string) {
	fmt.Fprintf(w, `{
	"response": "%v",
	"body": %v
	}`, response, output)
}

func PrintError(w http.ResponseWriter, err string) {
	fmt.Fprintf(w, `{
			"response": "error",
			"error": "%s"
			}`,
		err)
}

func PrintSuccess(w http.ResponseWriter, output string) {
	fmt.Fprintf(w, `{
			"response": "success",
			"body": %v
			}`,
		output)
}

func PrintDefaultSuccess(w http.ResponseWriter, msg string) {
	output := DefaultSuccessOutput{Message: msg}
	output_bytes, _ := json.Marshal(output)

	PrintSuccess(w, string(output_bytes))
}
