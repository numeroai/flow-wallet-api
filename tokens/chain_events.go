package tokens

import (
	"context"
	"strings"

	"github.com/flow-hydraulics/flow-wallet-api/accounts"
	"github.com/flow-hydraulics/flow-wallet-api/chain_events"
	"github.com/flow-hydraulics/flow-wallet-api/flow_helpers"
	"github.com/flow-hydraulics/flow-wallet-api/templates"
	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk"
	log "github.com/sirupsen/logrus"
)

type ChainEventHandler struct {
	AccountService  accounts.Service
	ChainListener   chain_events.Listener
	TemplateService templates.Service
	TokenService    Service
}

func (h *ChainEventHandler) Handle(ctx context.Context, event flow.Event) {
	isDeposit := strings.Contains(event.Type, "Deposit")
	if isDeposit {
		h.handleDeposit(ctx, event)
	}
}

func (h *ChainEventHandler) handleDeposit(ctx context.Context, event flow.Event) {
	// We don't have to care about tokens that are not in the database
	// as we could not even listen to events for them
	token, err := h.TemplateService.TokenFromEvent(event)
	if err != nil {
		return
	}

	// TODO: double check that these field names are correct
	// they are based on https://github.com/onflow/flow-ft/blob/master/contracts/FungibleToken.cdc
	// and https://github.com/onflow/flow-nft/blob/master/contracts/NonFungibleToken.cdc
	fields := event.Value.FieldsMappedByName()
	amount := fields["amount"]
	nftId := fields["id"]
	accountAddress := fields["to"]
	var amountOrNftID cadence.Value
	if amount != nil { amountOrNftID = amount } else { amountOrNftID = nftId }

	// Get the target account from database
	account, err := h.AccountService.Details(flow_helpers.HexString(accountAddress.String()))
	if err != nil {
		return
	}

	if err = h.TokenService.RegisterDeposit(ctx, token, event.TransactionID, account, amountOrNftID.String()); err != nil {
		log.
			WithFields(log.Fields{"error": err}).
			Warn("Error while registering a deposit")
		return
	}
}
