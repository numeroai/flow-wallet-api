package main

import (
	"bytes"
	j "encoding/json"
	"fmt"
	h "net/http"

	"github.com/google/uuid"
	"github.com/onflow/flow-go-sdk"
)

func main() {
	addNewKey()
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

func addNewKey() {
	fmt.Println("Running Sync Keys to add new keys to the account")

	accountAddress := "0xfaaecfd784e1508a"
	acctAddr := flow.HexToAddress(accountAddress) // this is a flow.Address

	b := ReqBody{Address: acctAddr.Hex()}

	idempotencyKey := uuid.New().String()

	bodyAsJson, jsonErr := j.Marshal(b)
	if jsonErr != nil {
		fmt.Println("Error marshaling JSON:", jsonErr)
	}

	bodyReader := bytes.NewReader(bodyAsJson)

	req, reqErr := h.NewRequest("POST", "http://localhost:3005/v1/accounts/"+accountAddress+"/add-new-key", bodyReader)
	if reqErr != nil {
		fmt.Println("Error:", reqErr)
		return
	}

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

	fmt.Println("Response Body: ", body)
}
