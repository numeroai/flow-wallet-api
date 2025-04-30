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
	fmt.Println("Args: ", args)
	if len(args) == 1 {
		fmt.Println("Accepted arguments: all, get-keys, with-addresses")
		return
	}

	if args[1] == "get-keys" {
		fmt.Println("Getting keys")
		getAwsKeys()
		return
	} else if args[1] == "all" {
		fmt.Println("Updating all aws keys")
		awsKeys := getAwsKeys()
		if len(awsKeys) == 0 {
			fmt.Println("No keys found")
			return
		}

		for _, key := range awsKeys {
			address := key.AccountAddress
			fmt.Println("==========================")
			fmt.Println("Updating account address: ", address)
			addNewKey(address)
			revokeOldKey(address, 0)
			fmt.Println("==========================")
		}
	} else if args[1] == "with-addresses" {
		if len(args) == 2 {
			fmt.Println("Please provide at least one address")
			return
		}

		addresses := os.Args[2:]
		fmt.Println("Processing ", len(addresses), " addresses")

		for _, address := range addresses {
			fmt.Println("==========================")
			fmt.Println("Updating account address: ", address)
			addNewKey(address)
			revokeOldKey(address, 0)
			fmt.Println("==========================")
		}
		return
	}
}

func getAwsKeys() []StorableKey {
	fmt.Println("Getting AWS keys")
	req, reqErr := h.NewRequest("GET", "http://localhost:3005/v1/get-keys/local", nil)
	if reqErr != nil {
		fmt.Println("Error:", reqErr)
		return []StorableKey{}
	}
	httpClient := &h.Client{}
	res, resErr := httpClient.Do(req)
	if resErr != nil {
		fmt.Println("Error sending http request:", reqErr)
		return []StorableKey{}
	}
	if res.StatusCode != h.StatusOK {
		fmt.Println("Not expected status code: ", res.StatusCode)
		return []StorableKey{}
	}
	fmt.Println("Status code: ", res.StatusCode)

	var body []StorableKey
	decodeErr := j.NewDecoder(res.Body).Decode(&body)
	if decodeErr != nil {
		fmt.Println("Error decoding response:", decodeErr)
		return []StorableKey{}
	}

	fmt.Println("Keys: ", len(body))
	fmt.Println("==========================")
	for _, key := range body {
		fmt.Println("Index: ", key.Index, "Address: ", key.AccountAddress)
	}
	return body
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
		fmt.Println("Not expected status code: ", res.StatusCode)
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

	if res.StatusCode != h.StatusOK {
		fmt.Println("Not expected status code: ", res.StatusCode)
		return
	}
	fmt.Println("Revert status code: ", res.StatusCode)

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

type KeyResponseBody struct {
	Keys []StorableKey `json:"keys"`
}

type StorableKey struct {
	ID             int    `json:"id"`
	AccountAddress string `json:"accountAddress"`
	Index          int    `json:"index"`
	Type           string `json:"type"`
	Value          string `json:"value"`
	SignAlgo       string `json:"signAlgo"`
	HashAlgo       string `json:"hashAlgo"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
	DeletedAt      string `json:"deletedAt"`
	PublicKey      string `json:"publicKey"`
}
