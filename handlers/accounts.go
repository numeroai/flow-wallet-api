package handlers

import (
	"net/http"

	"github.com/numeroai/flow-wallet-api/accounts"
	"github.com/onflow/flow-go-sdk"
)

// Accounts is a HTTP server for account management.
// It provides list, create and details APIs.
// It uses an account service to interface with data.
type Accounts struct {
	service accounts.Service
}

// SyncKeyCountRequest represents a JSON payload for a HTTP request
type SyncKeyCountRequest struct {
	Address flow.Address `json:"address"`
}

// NewAccounts initiates a new accounts server.
func NewAccounts(service accounts.Service) *Accounts {
	return &Accounts{service}
}

func (s *Accounts) List() http.Handler {
	return http.HandlerFunc(s.ListFunc)
}

func (s *Accounts) Create() http.Handler {
	return http.HandlerFunc(s.CreateFunc)
}

func (s *Accounts) AddNonCustodialAccount() http.Handler {
	return http.HandlerFunc(s.AddNonCustodialAccountFunc)
}

func (s *Accounts) DeleteNonCustodialAccount() http.Handler {
	return http.HandlerFunc(s.DeleteNonCustodialAccountFunc)
}

func (s *Accounts) SyncAccountKeyCount() http.Handler {
	return http.HandlerFunc(s.SyncAccountKeyCountFunc)
}

func (s *Accounts) Details() http.Handler {
	return http.HandlerFunc(s.DetailsFunc)
}

func (s *Accounts) AddNewKey() http.Handler {
	return http.HandlerFunc(s.AddNewKeyFunc)
}

func (s *Accounts) RevokeKey() http.Handler {
	return http.HandlerFunc(s.RevokeKeyFunc)
}

func (s *Accounts) GetKeysByType() http.Handler {
	return http.HandlerFunc(s.GetKeysByTypeFunc)
}