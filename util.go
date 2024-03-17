package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func getVersion(w http.ResponseWriter, r *http.Request) {
	fmt.Println("brr")
	data := struct {
		Version string `json:"version"`
	}{}
	data.Version = "0.6.0"
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)

}
