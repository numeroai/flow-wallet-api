package main

import (
	// "bytes"
	j "encoding/json"
	"fmt"
	h "net/http"
	"os"
	"strconv"

	"github.com/google/uuid"
)

// 0xfaaecfd784e1508a
func main() {
	args := os.Args
	if args == nil || len(args) < 2 {
		fmt.Println("No account address provided provided")
		return
	}
	address := os.Args[1]
	fmt.Println("==========================")
	fmt.Println("Updating account address: ", address)
	addNewKey(address)
	revokeOldKey(address, 0)
	fmt.Println("==========================")
}

func addNewKey(accountAddress string) {
	fmt.Println("Adding a new key")

	req, reqErr := h.NewRequest("POST", "http://localhost:3005/v1/accounts/"+accountAddress+"/add-new-key", nil)
	if reqErr != nil {
		fmt.Println("Error:", reqErr)
		return
	}

	idempotencyKey := uuid.New().String()
	req.Header.Add("Idempotency-Key", idempotencyKey)
	req.Header.Add("Content-Type", "application/json")

	httpClient := &h.Client{}

	res, resErr := httpClient.Do(req)
	if resErr != nil {
		fmt.Println("Error sending http request:", reqErr)
		return
	}

	if res.StatusCode != h.StatusCreated {
		fmt.Println("status code: ", res.StatusCode)
		return
	}
	fmt.Println("Status code: ", res.StatusCode)

	var body ResponseBody
	decodeErr := j.NewDecoder(res.Body).Decode(&body)
	if decodeErr != nil {
		fmt.Println("Error decoding response:", decodeErr)
		return
	}

	newKey := body.Keys[len(body.Keys)-1]
	fmt.Println("New key added: ", newKey.PublicKey)
	fmt.Println("With index: ", newKey.Index)
	fmt.Println("-------------------------")
}

func revokeOldKey(accountAddress string, keyIndex int) {
	fmt.Println("Revoking key with index: ", keyIndex)

	req, reqErr := h.NewRequest("POST", "http://localhost:3005/v1/accounts/"+accountAddress+"/revoke-key/"+strconv.Itoa(keyIndex), nil)
	if reqErr != nil {
		fmt.Println("Error:", reqErr)
		return
	}

	idempotencyKey := uuid.New().String()
	req.Header.Add("Idempotency-Key", idempotencyKey)
	req.Header.Add("Content-Type", "application/json")

	httpClient := &h.Client{}

	res, resErr := httpClient.Do(req)
	if resErr != nil {
		fmt.Println("Error sending http request:", reqErr)
		return
	}

	if res.StatusCode != h.StatusCreated {
		fmt.Println("status code: ", res.StatusCode)
		return
	}
	fmt.Println("Status code: ", res.StatusCode)

	var body ResponseBody
	decodeErr := j.NewDecoder(res.Body).Decode(&body)
	if decodeErr != nil {
		fmt.Println("Error decoding response:", decodeErr)
		return
	}

	oldKey := body.Keys[keyIndex]
	fmt.Println("Key revoked from account: ", oldKey.PublicKey)
}

type ReqBody struct {
	Address string `json:"address"`
}

type ResponseBody struct {
	Address   string `json:"address"`
	Keys      []Key  `json:"keys"`
	Type      string `json:"type"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type Key struct {
	Index     int    `json:"index"`
	Type      string `json:"type"`
	PublicKey string `json:"publicKey"`
	SignAlgo  string `json:"signAlgo"`
	HashAlgo  string `json:"hashAlgo"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}
