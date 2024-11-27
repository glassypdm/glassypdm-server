package project

import (
	"net/http"

	"github.com/go-chi/jwtauth/v5"
)

var TokenAuth *jwtauth.JWTAuth

func RouteStoreJWTRequest(w http.ResponseWriter, r *http.Request) {
	RequestStoreJWT("asdf", 19, false)
}
func RequestStoreJWT(UserID string, ProjectID int, IsDownload bool) {
	// TODO this should be in an init function

	// check if user can upload/download to/from ProjectID

	// create JWT if so, otherwise return false
	// in JWT, include userID, projectID, and intended action
}

func InitStoreJWT() {
	// grab secret from dotenv

	// initialize tokenauth
	TokenAuth = jwtauth.New("HS256", []byte("secret"), nil)
}
