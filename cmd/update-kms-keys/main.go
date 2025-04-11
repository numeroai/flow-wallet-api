package main

import (
	// "bytes"
	"context"
	// j "encoding/json"
	// "errors"
	"fmt"
	// h "net/http"
	// "os"
	// "strings"

	// "github.com/google/uuid"
	"github.com/onflow/flow-go-sdk"
	// "github.com/onflow/flow-go-sdk/access"
	"github.com/onflow/flow-go-sdk/access/http"
	"github.com/onflow/flow-go-sdk/crypto"
	"github.com/onflow/flow-go-sdk/templates"

	"github.com/onflow/flow-go-sdk/examples"

	"github.com/onflow/flowkit/config"
	"github.com/onflow/flowkit/config/json"
	"github.com/spf13/afero"

	"github.com/flow-hydraulics/flow-wallet-api/transactions"
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

	// instead of Random Account, get an account ... 0xfaaecfd784e1508a
	// todo :
		// - fetch these from the database
		// - remove the 0x at the beginning - do i need to do this?
	
	accountAddress := "0xfaaecfd784e1508a"
	acctAddr := flow.HexToAddress(accountAddress) // this is a flow.Address

	acct, getErr := flowClient.GetAccount(ctx, acctAddr)
	if getErr != nil {
		fmt.Println("Error getting account", getErr)
	}

	currentAcctKey := acct.Keys[0]
	fmt.Println("currentAcctKey: ", currentAcctKey)

	// Create the new key to add to your account
	myPrivateKey := examples.RandomPrivateKey() // todo: probably should be smarter about creating the new private key
	newAcctKey := flow.NewAccountKey().
		FromPrivateKey(myPrivateKey).
		SetHashAlgo(crypto.SHA3_256).
		SetWeight(flow.AccountKeyWeightThreshold)

	fmt.Println("newAcctKey", newAcctKey)

	addKeyTx, err := templates.AddAccountKey(acctAddr, newAcctKey)
	examples.Handle(err)
	fmt.Println("addKeyTx: ", addKeyTx)

	// i thnk that flow wallet api handles the reference block and the service account stuff :fingers-crossed
	// // referenceBlockID := examples.GetReferenceBlockId(flowClient)
	// // serviceAcctAddr,  serviceAcctKey, serviceSigner := ServiceAccount(flowClient)

	// what about the proposal key? this should probably be the account that is changing
	// // addKeyTx.SetProposalKey(acctAddr, acctKey.Index, acctKey.SequenceNumber)

	//the service account should probably just be the payer, which is probably handled by flow-wallet-api
	// // addKeyTx.SetPayer(serviceAcctAddr)
	// // addKeyTx.AddAuthorizer(acctAddr)


	// actually, i don't think i need to do this, because that is how flow-wallet-api is already setup 
	// // we just need the account to be the proposer?

	// keyAsKeyListEntry, kErr := templates.AccountKeyToCadenceCryptoKey(currentAcctKey)
	// if kErr != nil {
	// 	fmt.Println("Error converting account key to cadence crypto key", kErr)
	// }


	// txBody := transactions.JSONRequest{
	// 	Code: string(addKeyTx.Script),
	// 	Arguments: []transactions.Argument{keyAsKeyListEntry},
	// }

	// fmt.Println("txBody: ", txBody)

	// jobRes, jobErr := signTx(txBody, acctAddr.Hex())

	// if jobErr != nil {
	// 	fmt.Println("Error signing tx", jobErr)
	// }

	// fmt.Println("Job response: ", jobRes)

	// 	// instead of this... i think i need to send the tx the flow-wallet-api to have it signed by the account
	// 	// err = addKeyTx.SignPayload(acctAddr, acctKey.Index, accountASigner)
	// 	// if err != nil {
	// 	// 	panic(fmt.Sprintf("Failed to sign as Account A: %v", err))
	// 	// }


	// // // Send the transaction to the network.
	// // err = flowClient.SendTransaction(ctx, *addKeyTx)
	// // examples.Handle(err)

	// // examples.WaitForSeal(ctx, flowClient, addKeyTx.ID())

	// // fmt.Println("Public key added to account!")

}


// todo: rework the main package name in go.mod so that i can import transactions from the local version of transactions/transacionts.go
// it would probably work to impoort it from flow-wallet-api's main repo, because i think the type is the same... 
// but i dont' want to rely on thatA long term