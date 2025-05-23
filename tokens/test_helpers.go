package tokens

import (
	"context"

	"github.com/numeroai/flow-wallet-api/accounts"
	"github.com/numeroai/flow-wallet-api/flow_helpers"
	"github.com/numeroai/flow-wallet-api/templates"
	"github.com/numeroai/flow-wallet-api/templates/template_strings"
	flow_templates "github.com/onflow/flow-go-sdk/templates"
)

// DeployTokenContractForAccount is used for testing purposes.
func (s *ServiceImpl) DeployTokenContractForAccount(ctx context.Context, runSync bool, tokenName, address string) error {
	// Check if the input is a valid address
	address, err := flow_helpers.ValidateAddress(address, s.cfg.ChainID)
	if err != nil {
		return err
	}

	token, err := s.templates.GetTokenByName(tokenName)
	if err != nil {
		return err
	}

	n := token.Name

	tmplStr, err := template_strings.GetByName(n)
	if err != nil {
		return err
	}

	src := templates.TokenCode(s.cfg.ChainID, token, tmplStr)

	c := flow_templates.Contract{Name: n, Source: src}

	err = accounts.AddContract(ctx, s.fc, s.km, address, c, s.cfg.TransactionTimeout)
	if err != nil {
		return err
	}

	return nil
}
