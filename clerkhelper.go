package main

import (
	"context"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/joshtenorio/glassypdm-server/sqlcgen"
)

func GetName(UserId string) (string, error) {
	ctx := context.Background()
	key := []byte(os.Getenv("AES_KEY"))

	CacheEntry, err := queries.GetClerkCache(ctx, UserId)
	if err == nil {
		if CacheEntry.Expires > time.Now().Unix() {
			// decrypt with key
			decrypted, err := Decrypt(key, CacheEntry.Value)
			return decrypted, err
		}
		// otherwise we need to update the cache
	}

	usr, err := user.Get(ctx, UserId)
	if err != nil {
		log.Error("could not retrieve user from clerk", "userId", UserId)
		return "", err
	}

	name := *usr.FirstName + " " + *usr.LastName
	time := time.Now().Unix()
	expiry := time + 86400 // expire in 24 hours

	EncodedName, err := Encrypt(key, name)
	if err != nil {
		log.Warn("encrypting failed, skipping cache upsert", "err", err)
		return name, err
	}

	queries.UpsertClerkCache(ctx,
		sqlcgen.UpsertClerkCacheParams{
			Clerkid:        UserId,
			Type:           "name",
			Value:          EncodedName,
			Encodingscheme: "aes256",
			Encodingkey:    "AES_KEY",
			Created:        time,
			Expires:        expiry})
	return name, nil
}

/* POSTPONED
func GetEmail(UserId string) (string, error) {
	ctx := context.Background()
	// TODO check cache
	usr, err := user.Get(ctx, UserId)
	if err != nil {
		log.Warn("could not retrieve user from clerk", "userId", UserId)
		return "", err
	}
	email, err := emailaddress.Get(ctx, *usr.PrimaryEmailAddressID)
	if err != nil {
		log.Warn("could not retrieve email from clerk", "emailID", *usr.PrimaryEmailAddressID)
		return "", err
	}
	// TODO update cache
	return email.EmailAddress, nil
}
*/
