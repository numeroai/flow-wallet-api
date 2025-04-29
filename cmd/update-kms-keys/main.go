package main

import (
	// "bytes"
	j "encoding/json"
	"fmt"
	h "net/http"
	"os"

	"github.com/google/uuid"
	// "github.com/onflow/flow-go-sdk"
)

// 0xfaaecfd784e1508a
func main() {
	args := os.Args
	fmt.Println("args: ", args[1])
	if args == nil || len(args) < 2 {
		fmt.Println("No account address provided provided")
		return
	}
	address := os.Args[1]
	addNewKey(address)
}

func addNewKey(accountAddress string) {
	fmt.Println("Running Sync Keys to add new keys to the account")

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
	fmt.Println("status code: ", res.StatusCode)

	var body ResponseBody
	decodeErr := j.NewDecoder(res.Body).Decode(&body)
	if decodeErr != nil {
		fmt.Println("Error decoding response:", decodeErr)
		return
	}

	fmt.Printf("A new key was added to account %s\n", body.Address)
	newKey := body.Keys[len(body.Keys)-1]
	fmt.Printf("New Public Key: %s\n", newKey.PublicKey)
	fmt.Printf("Key Index: %d\n", newKey.Index)
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
