package main

import (
	"bytes"
	j "encoding/json"
	"fmt"
	"io"
	h "net/http"
	"os"
	"strconv"

	"github.com/google/uuid"
	t "github.com/onflow/sdks"
)

var (
	FLOW_WALLET_API_URL string
)

func main() {
	flow_wallet_api_url := os.Getenv("FLOW_WALLET_API_URL")
	if flow_wallet_api_url == "" {
		fmt.Println("Environment variable FLOW_WALLET_API_URL is not set")
		return
	} else {
		FLOW_WALLET_API_URL = flow_wallet_api_url
		fmt.Println("FLOW_WALLET_API_URL: ", flow_wallet_api_url)
	}

	args := os.Args
	fmt.Println("Args: ", args)
	if len(args) == 1 {
		fmt.Println("Accepted arguments: get-keys, add-with-addresses, revoke-with-addresses, revoke-on-chain-with-address-and-index")
		return
	}

	if args[1] == "get-keys" {
		fmt.Println("Getting keys")
		getAwsKeys()
		return
	} else if args[1] == "add-with-addresses" {
		if len(args) == 2 {
			fmt.Println("Please provide at least one address")
			return
		}

		addresses := os.Args[2:]
		fmt.Println("Processing ", len(addresses), " addresses")

		for _, address := range addresses {
			fmt.Println("==========================")
			fmt.Println("Updating account address: ", address)
			addNewKeyErr := addNewKey(address)
			if addNewKeyErr != nil {
				fmt.Println("Error adding new key: ", addNewKeyErr)
				return
			}
			// revokeOldKey(address, 0)
			fmt.Println("==========================")
		}
		return
	} else if args[1] == "revoke-with-addresses" {
		if len(args) == 2 {
			fmt.Println("Please provide at least one address")
			return
		}

		addresses := os.Args[2:]
		fmt.Println("Processing ", len(addresses), " addresses")

		for _, address := range addresses {
			fmt.Println("==========================")
			fmt.Println("Revoked key 0 for account address: ", address)
			revokeOldKey(address, 0)
			fmt.Println("==========================")
		}
		return
	} else if args[1] == "revoke-on-chain-with-address-and-index" {
		if len(args) < 4 {
			fmt.Println("Please provide an address and an index to revoke")
			return
		}

		address := os.Args[2]
		index, err := strconv.Atoi(os.Args[3])
		if err != nil {
			fmt.Println("Error converting index to int: ", err)
			return
		}
		fmt.Println("Revoking ", index, "for ", address)

		revokeOldKeyOnChain(address, index)
		return
	} else {
		fmt.Println("Invalid argument: ", args[1])
		return
	}
}

func getAwsKeys() []StorableKey {
	fmt.Println("Getting AWS keys")
	req, reqErr := h.NewRequest("GET", FLOW_WALLET_API_URL+"/get-keys/aws_kms", nil)
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

func addNewKey(accountAddress string) error {
	fmt.Println("Adding a new key")

	req, reqErr := h.NewRequest("POST", FLOW_WALLET_API_URL+"/accounts/"+accountAddress+"/add-new-key", nil)
	if reqErr != nil {
		fmt.Println("Error:", reqErr)
		return reqErr
	}

	idempotencyKey := uuid.New().String()
	req.Header.Add("Idempotency-Key", idempotencyKey)
	req.Header.Add("Content-Type", "application/json")

	httpClient := &h.Client{}

	res, resErr := httpClient.Do(req)
	if resErr != nil {
		fmt.Println("Error sending http request:", reqErr)
		return resErr
	}

	if res.StatusCode != h.StatusCreated {
		defer res.Body.Close()

		b, err := io.ReadAll(res.Body)
		if err != nil {
			fmt.Println("Error reading response body:", err)
			return err
		}

		fmt.Println(string(b))

		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}
	fmt.Println("Status code: ", res.StatusCode)

	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return err
	}

	fmt.Println(string(b))

	return nil
}

func revokeOldKey(accountAddress string, keyIndex int) {
	fmt.Println("Revoking key with index: ", keyIndex)

	req, reqErr := h.NewRequest("POST", FLOW_WALLET_API_URL+"/accounts/"+accountAddress+"/revoke-key/"+strconv.Itoa(keyIndex), nil)
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

	fmt.Println("Revert status code: ", res.StatusCode)
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
	}

	fmt.Println(string(b))
}

func revokeOldKeyOnChain(accountAddress string, keyIndex int) {
	args := []map[string]string{{"type": "Int", "value": strconv.Itoa(keyIndex)}}
	// Create transaction
	code := t.RemoveAccountKey

	type requestBody struct {
		Code      string      `json:"code"`
		Arguments interface{} `json:"arguments"`
	}

	reqBody := requestBody{
		Code:      code,
		Arguments: args,
	}

	reqBodyBytes, err := j.Marshal(reqBody)
	if err != nil {
		fmt.Println("Error marshalling request body:", err)
		return
	}

	fmt.Println("Request body: ", string(reqBodyBytes))
	req, reqErr := h.NewRequest("POST", FLOW_WALLET_API_URL+"/accounts/"+accountAddress+"/transactions?", bytes.NewBuffer(reqBodyBytes))
	if reqErr != nil {
		fmt.Println("Error:", reqErr)
		return
	}
	req.Header.Add("Idempotency-Key", uuid.New().String())
	req.Header.Add("Content-Type", "application/json")

	httpClient := &h.Client{}
	res, resErr := httpClient.Do(req)
	if resErr != nil {
		fmt.Println("Error sending http request:", reqErr)
		return
	}
	if res.StatusCode != h.StatusOK {
		defer res.Body.Close()

		b, err := io.ReadAll(res.Body)
		if err != nil {
			fmt.Println("Error reading response body:", err)
			return
		}

		fmt.Println(string(b))
	}

	fmt.Println("Status code: ", res.StatusCode)
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}
	fmt.Println(string(b))
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
