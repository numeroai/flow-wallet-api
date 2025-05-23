package tokens

import (
	"github.com/numeroai/flow-wallet-api/accounts"
	"github.com/numeroai/flow-wallet-api/flow_helpers"
	"github.com/numeroai/flow-wallet-api/templates"
	log "github.com/sirupsen/logrus"
)

type AccountAddedHandler struct {
	TemplateService templates.Service
	TokenService    Service
}

func (h *AccountAddedHandler) Handle(payload accounts.AccountAddedPayload) {
	address := flow_helpers.FormatAddress(payload.Address)
	h.addFlowToken(address)
}

func (h *AccountAddedHandler) addFlowToken(address string) {
	if err := h.TokenService.AddAccountToken("FlowToken", address); err != nil {
		log.
			WithFields(log.Fields{"error": err}).
			Warn("Error while adding FlowToken to new account")
	}
}
