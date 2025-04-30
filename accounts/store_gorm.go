package accounts

import (
	"fmt"

	"github.com/flow-hydraulics/flow-wallet-api/datastore"
	"github.com/flow-hydraulics/flow-wallet-api/keys"
	"gorm.io/gorm"
)

type GormStore struct {
	db *gorm.DB
}

func NewGormStore(db *gorm.DB) Store {
	return &GormStore{db}
}

func (s *GormStore) Accounts(o datastore.ListOptions) (aa []Account, err error) {
	err = s.db.
		Order("created_at desc").
		Limit(o.Limit).
		Offset(o.Offset).
		Find(&aa).Error
	return
}

func (s *GormStore) Account(address string) (a Account, err error) {
	err = s.db.Preload("Keys").First(&a, "address = ?", address).Error
	return
}

func (s *GormStore) InsertAccount(a *Account) error {
	return s.db.Create(a).Error
}

func (s *GormStore) SaveAccount(a *Account) error {
	return s.db.Save(&a).Error
}

func (s *GormStore) HardDeleteAccount(a *Account) error {
	return s.db.Unscoped().Delete(a).Error
}

func (s *GormStore) DeleteKeyForAccount(a *Account, key *keys.Storable) error {
	if key.AccountAddress != a.Address {
		return fmt.Errorf("key does not belong to the given account")
	}

	return s.db.
		Where("account_address = ? AND id = ?", a.Address, key.ID).
		Delete(&keys.Storable{}).
		Error
}
