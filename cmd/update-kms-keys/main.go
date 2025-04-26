package main

import (
	"bytes"
	"context"
	j "encoding/json"
	"errors"
	"fmt"
	h "net/http"

	"github.com/google/uuid"
	"github.com/onflow/flow-go-sdk"

	"github.com/onflow/flow-go-sdk/access/http"

	"github.com/onflow/flow-go-sdk/examples"
)

func main() {
	addNewKey()
}

func addNewKey() {
	fmt.Println("Running Sync Keys to add new keys to the account")

	ctx := context.Background()

	flowClient, err := http.NewClient(http.EmulatorHost)
	examples.Handle(err)

	accountAddress := "0xfaaecfd784e1508a"
	acctAddr := flow.HexToAddress(accountAddress) // this is a flow.Address

	acct, getErr := flowClient.GetAccount(ctx, acctAddr)
	if getErr != nil {
		fmt.Println("Error getting account", getErr)
	}

	currentAcctKey := acct.Keys[0] // may not even need this, flow wallet api
	fmt.Println("Current account key: ", currentAcctKey)

	type ReqBody struct {
		Address string `json:"address"`
	}

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
	}
}

// Transaction JSON HTTP request
type TxRequestBody struct {
	Code      string     `json:"code"`
	Arguments []Argument `json:"arguments"`
}

type Argument interface{}

type JobResponse struct {
	JobId         string   `json:"jobId"`
	Type          string   `json:"type"`
	State         string   `json:"state"`
	Error         string   `json:"error"`
	Errors        []string `json:"errors"`
	Result        string   `json:"result"`
	TransactionId string   `json:"transactionId"`
	CreatedAt     string   `json:"createdAt"`
	UpdatedAt     string   `json:"updatedAt"`
}

type SignedTransactionResponse struct {
	Code               string                     `json:"code"`
	Arguments          []Argument                 `json:"arguments"`
	ReferenceBlockID   string                     `json:"referenceBlockId"`
	GasLimit           uint64                     `json:"gasLimit"`
	ProposalKey        ProposalKeyJSON            `json:"proposalKey"`
	Payer              string                     `json:"payer"`
	Authorizers        []string                   `json:"authorizers"`
	PayloadSignatures  []TransactionSignatureJSON `json:"payloadSignatures"`
	EnvelopeSignatures []TransactionSignatureJSON `json:"envelopeSignatures"`
}

// are these to json types necessary?
type ProposalKeyJSON struct {
	Address        string `json:"address"`
	KeyIndex       uint32 `json:"keyIndex"`
	SequenceNumber uint64 `json:"sequenceNumber"`
}

type TransactionSignatureJSON struct {
	Address   string `json:"address"`
	KeyIndex  uint32 `json:"keyIndex"`
	Signature string `json:"signature"`
}