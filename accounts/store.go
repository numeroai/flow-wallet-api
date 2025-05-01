package accounts

import (
	"github.com/numeroai/flow-wallet-api/datastore"
	"github.com/numeroai/flow-wallet-api/keys"
)

// Store manages data regarding accounts.
type Store interface {
	// List all accounts.
	Accounts(datastore.ListOptions) ([]Account, error)

	// Get account details.
	Account(address string) (Account, error)

	// Insert a new account.
	InsertAccount(a *Account) error

	// Update an existing account.
	SaveAccount(a *Account) error

	// Permanently delete an account, despite of `DeletedAt` field.
	HardDeleteAccount(a *Account) error

	// Delete a storable key for an account
	DeleteKeyForAccount(a *Account, key *keys.Storable) error

	// Get keys by key type
	GetKeysByType(keyType string) ([]keys.Storable, error)
}
