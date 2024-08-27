package main

import (
	"fmt"
	"net/http"
)

type DefaultSuccessOutput struct {
	Message string `json:"message"`
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
