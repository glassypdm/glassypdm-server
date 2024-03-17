package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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

func getConfig(w http.ResponseWriter, r *http.Request) {
	fmt.Println("brr")
	data := struct {
		Key string `json:"clerk_publickey"`
	}{}
	data.Key = os.Getenv("CLERK_PUBLICKEY")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
