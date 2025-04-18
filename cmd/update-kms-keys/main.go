package main

import (
	"bytes"
	"context"
	j "encoding/json"
	"errors"
	"fmt"
	h "net/http"

	jsoncdc "github.com/onflow/cadence/encoding/json"

	// "os"
	// "strings"

	"github.com/google/uuid"
	"github.com/onflow/flow-go-sdk"

	// "github.com/onflow/flow-go-sdk/access"
	"github.com/onflow/flow-go-sdk/access/http"
	"github.com/onflow/flow-go-sdk/crypto"
	"github.com/onflow/flow-go-sdk/templates"
	t "github.com/onflow/sdks"

	"github.com/onflow/flow-go-sdk/examples"

	"github.com/onflow/flowkit/config"
	"github.com/onflow/flowkit/config/json"
	"github.com/spf13/afero"
)

const configPath = "../../flow/flow.json"

var Config *config.Config

// todo: refactor / take another look at this
func initConfig() {
	var mockFS = afero.NewOsFs()

	var af = afero.Afero{Fs: mockFS}

	l := config.NewLoader(af)

	l.AddConfigParser(json.NewParser())

	var err error
	Config, err = l.Load([]string{configPath})
	if err != nil {
		fmt.Println("Error loading config", err)
		return
	}
}

func main() {
	initConfig()
	runKmsUpdate()
}

func runKmsUpdate() {
	fmt.Println("Running KMS update")

	ctx := context.Background()

	flowClient, err := http.NewClient(http.EmulatorHost)
	examples.Handle(err)

	// todo :
	// - fetch accounts from the database
	// - remove the 0x at the beginning - do i need to do this?

	accountAddress := "0xfaaecfd784e1508a"
	acctAddr := flow.HexToAddress(accountAddress) // this is a flow.Address

	acct, getErr := flowClient.GetAccount(ctx, acctAddr)
	if getErr != nil {
		fmt.Println("Error getting account", getErr)
	}

	currentAcctKey := acct.Keys[0] // may not even need this, flow wallet api handles this stuff
	fmt.Println("Current account key: ", currentAcctKey)

	// Create the new key to add to your account
	myPrivateKey := examples.RandomPrivateKey() // todo: probably should be smarter about creating the new private key
	newAcctKey := flow.NewAccountKey().
		FromPrivateKey(myPrivateKey).
		SetHashAlgo(crypto.SHA3_256).
		SetWeight(flow.AccountKeyWeightThreshold)

	//AddAccountKey handles adding the authorizer, script and raw arg
	addKeyTxScript := t.AddAccountKey

	// i thnk that flow wallet api handles the reference block and the service account stuff :fingers-crossed
	// // referenceBlockID := examples.GetReferenceBlockId(flowClient)
	// // serviceAcctAddr,  serviceAcctKey, serviceSigner := ServiceAccount(flowClient)

	// what about the proposal key? this should probably be the account that is changing
	// // addKeyTx.SetProposalKey(acctAddr, acctKey.Index, acctKey.SequenceNumber)

	//the service account should probably just be the payer, which is probably handled by flow-wallet-api
	// // addKeyTx.SetPayer(serviceAcctAddr)
	// // addKeyTx.AddAuthorizer(acctAddr)

	keyAsKeyListEntry, kErr := templates.AccountKeyToCadenceCryptoKey(newAcctKey)
	if kErr != nil {
		fmt.Println("Error converting account key to cadence crypto key", kErr)
	}

	// INSTEAD OF MESSING WITH THIS< MAYBE IT IS EASIER TO CREATE A NEW END POINT, THEN I CAN AVOID THE HTTP STUFF
	encoded, jsonErr := jsoncdc.Encode(keyAsKeyListEntry)
	if jsonErr != nil {
		fmt.Println("Error encoding args with cadence encoder", jsonErr)
	}

	// encodedArgs, err := jsoncdc.Encode(keyAsKeyListEntry)
	// if err != nil {
	// 	fmt.Println("Error encoding args with cadence encoder", err)
	// }

	// marshaled, err:= jsoncdc.Encode(arg)

	encodedArguments := []Argument{string(encoded)}
	txBody := TxRequestBody{
		Code:      addKeyTxScript,
		Arguments: encodedArguments,
	}

	res, err := signTx(txBody, acctAddr.Hex())
	if err != nil {
		fmt.Println("Error signing tx", err)
		return
	}

	fmt.Println("signed tx: ", res.Code)

	sendRes, err := sendTx(TxRequestBody{Code: res.Code, Arguments: encodedArguments}, acctAddr.Hex())
	if err != nil {
		fmt.Println("Error sending tx", err)
	}
	fmt.Println("sendRes: ", sendRes)

	// // // Send the transaction to the network.
	// // err = flowClient.SendTransaction(ctx, *addKeyTx)
	// // examples.Handle(err)

	// // examples.WaitForSeal(ctx, flowClient, addKeyTx.ID())

	// // fmt.Println("Public key added to account!")

}

// todo: rework the main package name in go.mod so that i can import transactions from the local version of transactions/transacionts.go
// it would probably work to impoort it from flow-wallet-api's main repo, because i think the type is the same...
// but i dont' want to rely on thatA long term

// this is copy-pasted from flow-wallet-api/transactions/transactions.go

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
	Arguments          []Argument                   `json:"arguments"`
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

func signTx(reqBody TxRequestBody, acctAddr string) (SignedTransactionResponse, error) {

	idempotencyKey := uuid.New().String()

	b := map[string]interface{}{
		"code":      reqBody.Code,
		"arguments": reqBody.Arguments,
	}

	jsonBytes, err := j.Marshal(b)
	if err != nil {
			fmt.Println("Error marshaling JSON:", err)
	}

	bodyReader := bytes.NewReader(jsonBytes)
	req, reqErr := h.NewRequest("POST", "http://localhost:3005/v1/accounts/"+acctAddr+"/sign", bodyReader)
	if reqErr != nil {
		return SignedTransactionResponse{}, reqErr
	}

	req.Header.Add("Idempotency-Key", idempotencyKey)
	req.Header.Add("Content-Type", "application/json")

	httpClient := &h.Client{}

	res, resErr := httpClient.Do(req)
	if resErr != nil {
		return SignedTransactionResponse{}, resErr
	}

	if res.StatusCode != h.StatusCreated {
		fmt.Println("status code: ", res.StatusCode)
		return SignedTransactionResponse{}, errors.New("failed to sign transaction")
	}

	var body SignedTransactionResponse
	decodeErr := j.NewDecoder(res.Body).Decode(&body)
	if decodeErr != nil {
		return SignedTransactionResponse{}, decodeErr
	}

	return body, nil
}

func sendTx(reqBody TxRequestBody, acctAddr string) (JobResponse, error) {
	idempotencyKey := uuid.New().String()

	bodyAsJson, jsonErr := j.Marshal(reqBody)
	if jsonErr != nil {
		return JobResponse{}, jsonErr
	}
	bodyReader := bytes.NewReader(bodyAsJson)

	req, reqErr := h.NewRequest("POST", "http://localhost:3005/v1/accounts/"+acctAddr+"/transactions", bodyReader)
	if reqErr != nil {
		return JobResponse{}, reqErr
	}

	req.Header.Add("Idempotency-Key", idempotencyKey)
	req.Header.Add("Content-Type", "application/json")

	httpClient := &h.Client{}

	res, resErr := httpClient.Do(req)
	if resErr != nil {
		return JobResponse{}, resErr
	}

	if res.StatusCode != h.StatusCreated {
		fmt.Println("status code: ", res.StatusCode)
		return JobResponse{}, errors.New("failed to send transaction")
	}

	var body JobResponse
	decodeErr := j.NewDecoder(res.Body).Decode(&body)
	if decodeErr != nil {
		return JobResponse{}, decodeErr
	}

	return body, nil
}
