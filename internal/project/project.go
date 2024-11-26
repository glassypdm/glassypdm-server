package project

import (
	"net/http"

	"github.com/go-chi/jwtauth/v5"
)

var TokenAuth *jwtauth.JWTAuth

func RouteStoreJWTRequest(w http.ResponseWriter, r *http.Request) {
	RequestStoreJWT("asdf")
}
func RequestStoreJWT(UserID string) {
	TokenAuth = jwtauth.New("HS256", []byte("secret"), nil)
}
