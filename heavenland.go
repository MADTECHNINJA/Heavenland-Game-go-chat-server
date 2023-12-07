package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt"
)

type GetAccountResp struct {
	Id                    string `json:"id"`
	Wallet                string `json:"wallet"`
	Nickname              string `json:"nickname"`
	Username              string `json:"username"`
	ReferralCode          string `json:"referralCode"`
	ReferralId            string `json:"referralId"`
	CreatedAt             int64  `json:"createdAt"`
	IsReadyForGameLogin   bool   `json:"isReadyForGameLogin"`
	EarlyAccessToGameInfo GetAccountEarlyAccessInfo
}

type GetAccountEarlyAccessInfo struct {
	HasEarlyAccess bool   `json:"hasEarlyAccess"`
	Type           string `json:"type"`
}

type KeyClaims struct {
	UserId string `json:"sub"`
	jwt.StandardClaims
}

// embed public key when compiling the chat server
//go:embed pub.pem
var f embed.FS

type Heavenland struct {
	// url of the API
	url string

	// public key to validate access token
	pubKey []byte

	// authentication status
	auth bool

	// user id unpacked from the access token claims
	userId string
}

func newHeavneland() *Heavenland {
	pubKey, err := f.ReadFile("pub.pem")
	if err != nil {
		fmt.Println("raise error that the public could not be found")
	}

	return &Heavenland{
		url:    os.Getenv("HEAVENLAND_URL"),
		pubKey: pubKey,
		auth:   false,
		userId: "",
	}
}

// function to check the validity of the provided auth token and login the connected user
func (h *Heavenland) authenticate(loginToken string) bool {
	pubKey, err := f.ReadFile("pub.pem")

	if err != nil {
		fmt.Println("raise error that the public could not be found")
	}

	token, _ := jwt.ParseWithClaims(loginToken, &KeyClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwt.ParseRSAPublicKeyFromPEM(pubKey)
	})
	if claims, ok := token.Claims.(*KeyClaims); ok && token.Valid {
		h.auth = true
		h.userId = claims.UserId
	}
	return token.Valid

}

func (h *Heavenland) fetchUsername(loginToken string) (bool, GetAccountResp) {
	fmt.Println("fetching username")
	bearer := "Bearer " + loginToken
	fmt.Println(h.url)
	req, _ := http.NewRequest("GET", h.url+"accounts/"+h.userId, nil)
	req.Header.Add("Authorization", bearer)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error on response.\n[ERROR] -", err)
	}
	if resp.StatusCode != 200 {
		return false, GetAccountResp{}
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var data GetAccountResp
	err = decoder.Decode(&data)

	if err != nil {
		log.Println("Error while reading the response bytes:", err)
	}
	log.Println(data.Nickname)
	return true, data
}
