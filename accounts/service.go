package accounts

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/numeroai/flow-wallet-api/configs"
	"github.com/numeroai/flow-wallet-api/datastore"
	"github.com/numeroai/flow-wallet-api/flow_helpers"
	"github.com/numeroai/flow-wallet-api/jobs"
	"github.com/numeroai/flow-wallet-api/keys"
	"github.com/numeroai/flow-wallet-api/templates/template_strings"
	"github.com/numeroai/flow-wallet-api/transactions"
	"github.com/onflow/cadence"
	"github.com/onflow/flow-go-sdk"
	flow_crypto "github.com/onflow/flow-go-sdk/crypto"
	"github.com/onflow/flow-go-sdk/templates"
	flow_templates "github.com/onflow/flow-go-sdk/templates"
	t "github.com/onflow/sdks"
	log "github.com/sirupsen/logrus"
	"go.uber.org/ratelimit"
)

const maxGasLimit = 9999

type Service interface {
	List(limit, offset int) (result []Account, err error)
	Create(ctx context.Context, sync bool) (*jobs.Job, *Account, error)
	AddNonCustodialAccount(address string) (*Account, error)
	DeleteNonCustodialAccount(address string) error
	SyncAccountKeyCount(ctx context.Context, address flow.Address) (*jobs.Job, error)
	Details(address string) (Account, error)
	InitAdminAccount(ctx context.Context) error
	AddNewKey(ctx context.Context, address flow.Address) (*jobs.Job, error)
	RevokeKey(ctx context.Context, address flow.Address, oldKeyIndex uint32) (*jobs.Job, error)
	GetKeysByType(ctx context.Context, keyType string) ([]keys.Storable, error)
}

// ServiceImpl defines the API for account management.
type ServiceImpl struct {
	cfg           *configs.Config
	store         Store
	km            keys.Manager
	fc            flow_helpers.FlowClient
	wp            jobs.WorkerPool
	txs           transactions.Service
	txRateLimiter ratelimit.Limiter
}

// NewService initiates a new account service.
func NewService(
	cfg *configs.Config,
	store Store,
	km keys.Manager,
	fc flow_helpers.FlowClient,
	wp jobs.WorkerPool,
	txs transactions.Service,
	opts ...ServiceOption,
) Service {
	var defaultTxRatelimiter = ratelimit.NewUnlimited()

	// TODO(latenssi): safeguard against nil config?
	svc := &ServiceImpl{cfg, store, km, fc, wp, txs, defaultTxRatelimiter}

	for _, opt := range opts {
		opt(svc)
	}

	if wp == nil {
		panic("workerpool nil")
	}

	// Register asynchronous job executors
	wp.RegisterExecutor(AccountCreateJobType, svc.executeAccountCreateJob)
	wp.RegisterExecutor(SyncAccountKeyCountJobType, svc.executeSyncAccountKeyCountJob)
	wp.RegisterExecutor(AddNewKeyJobType, svc.executeAddNewKeyJob)
	wp.RegisterExecutor(RevokeKeyJobType, svc.executeRevokeKeyJob)

	return svc
}

// List returns all accounts in the datastore.
func (s *ServiceImpl) List(limit, offset int) (result []Account, err error) {
	o := datastore.ParseListOptions(limit, offset)
	return s.store.Accounts(o)
}

// Create calls account.New to generate a new account.
// It receives a new account with a corresponding private key or resource ID
// and stores both in datastore.
// It returns a job, the new account and a possible error.
func (s *ServiceImpl) Create(ctx context.Context, sync bool) (*jobs.Job, *Account, error) {
	log.WithFields(log.Fields{"sync": sync}).Trace("Create account")

	if !sync {
		job, err := s.wp.CreateJob(AccountCreateJobType, "")
		if err != nil {
			return nil, nil, err
		}

		err = s.wp.Schedule(job)
		if err != nil {
			return nil, nil, err
		}

		return job, nil, err
	}

	account, _, err := s.createAccount(ctx)
	if err != nil {
		return nil, nil, err
	}

	return nil, account, nil
}

func (s *ServiceImpl) AddNonCustodialAccount(address string) (*Account, error) {
	log.WithFields(log.Fields{"address": address}).Trace("Add non-custodial account")

	a := &Account{
		Address: flow_helpers.HexString(address),
		Type:    AccountTypeNonCustodial,
	}

	err := s.store.InsertAccount(a)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (s *ServiceImpl) DeleteNonCustodialAccount(address string) error {
	log.WithFields(log.Fields{"address": address}).Trace("Delete non-custodial account")

	a, err := s.store.Account(flow_helpers.HexString(address))
	if err != nil {
		if strings.Contains(err.Error(), "record not found") {
			// Account already gone. All good.
			return nil
		}

		return err
	}

	if a.Type != AccountTypeNonCustodial {
		return fmt.Errorf("only non-custodial accounts supported")
	}

	return s.store.HardDeleteAccount(&a)
}

// Details returns a specific account, does not include private keys
func (s *ServiceImpl) Details(address string) (Account, error) {
	log.WithFields(log.Fields{"address": address}).Trace("Account details")

	// Check if the input is a valid address
	address, err := flow_helpers.ValidateAddress(address, s.cfg.ChainID)
	if err != nil {
		return Account{}, err
	}

	account, err := s.store.Account(address)
	if err != nil {
		return Account{}, err
	}

	// Strip the private keys
	for i := range account.Keys {
		account.Keys[i].Value = make([]byte, 0)
	}

	return account, nil
}

// SyncKeyCount syncs number of keys for given account
func (s *ServiceImpl) SyncAccountKeyCount(ctx context.Context, address flow.Address) (*jobs.Job, error) {
	// Validate address, they might be legit addresses but for the wrong chain
	if !address.IsValid(s.cfg.ChainID) {
		return nil, fmt.Errorf(`not a valid address for %s: "%s"`, s.cfg.ChainID, address)
	}

	// Prepare job attributes required for executing the job
	attrs := syncAccountKeyCountJobAttributes{Address: address, NumKeys: int(s.cfg.DefaultAccountKeyCount)}
	attrBytes, err := json.Marshal(attrs)
	if err != nil {
		return nil, err
	}

	// Create & schedule the "sync key count" job
	job, err := s.wp.CreateJob(SyncAccountKeyCountJobType, "", jobs.WithAttributes(attrBytes))
	if err != nil {
		return nil, err
	}
	err = s.wp.Schedule(job)
	if err != nil {
		return nil, err
	}

	return job, nil
}

// syncAccountKeyCount syncs the number of account keys with the given numKeys and
// returns the number of keys, transaction ID and error.
func (s *ServiceImpl) syncAccountKeyCount(ctx context.Context, address flow.Address, numKeys int) (int, string, error) {
	entry := log.WithFields(log.Fields{"address": address, "numKeys": numKeys, "function": "ServiceImpl.syncAccountKeyCount"})

	if numKeys < 1 {
		return 0, "", fmt.Errorf("invalid number of keys specified: %d, min. 1 expected", numKeys)
	}

	// Check on-chain keys
	flowAccount, err := s.fc.GetAccount(ctx, address)
	if err != nil {
		entry.WithFields(log.Fields{"err": err}).Error("failed to get Flow account")
		return 0, "", err
	}

	// Get stored account
	dbAccount, err := s.store.Account(flow_helpers.FormatAddress(address))
	if err != nil {
		entry.WithFields(log.Fields{"err": err}).Error("failed to get account from database")
		return 0, "", err
	}

	// Pick a source key that will be used to create the new keys & decode public key
	sourceKey := dbAccount.Keys[0] // NOTE: Only valid (not revoked) keys should be stored in the database
	sourceKeyPbkString := strings.TrimPrefix(sourceKey.PublicKey, "0x")
	sourcePbk, err := flow_crypto.DecodePublicKeyHex(flow_crypto.StringToSignatureAlgorithm(sourceKey.SignAlgo), sourceKeyPbkString)
	if err != nil {
		entry.WithFields(log.Fields{"err": err, "sourceKeyPbkString": sourceKeyPbkString}).Error("failed to decode public key for source key")
		return 0, "", err
	}
	entry.WithFields(log.Fields{"sourceKeyId": sourceKey.ID, "sourcePbk": sourcePbk}).Trace("source key selected")

	// Count valid keys, as some keys might be revoked, assuming dbAccount.Keys are clones (all have same public key)
	var validKeys []*flow.AccountKey
	for i := range flowAccount.Keys {
		key := flowAccount.Keys[i]
		if !key.Revoked && key.PublicKey.Equals(sourcePbk) {
			validKeys = append(validKeys, key)
		}
	}

	if len(validKeys) != len(dbAccount.Keys) {
		entry.WithFields(log.Fields{"onChain": len(validKeys), "database": len(dbAccount.Keys)}).Warn("on-chain vs. database key count mismatch")
	}

	entry.WithFields(log.Fields{"validKeys": validKeys}).Trace("filtered valid keys")

	// Add keys by cloning the source key
	if len(validKeys) < numKeys {

		cloneCount := numKeys - len(validKeys)
		code := template_strings.AddAccountKeysTransaction
		pbks := []cadence.Value{}

		entry.WithFields(log.Fields{"validKeys": len(validKeys), "numKeys": numKeys, "cloneCount": cloneCount}).Debug("going to add keys")

		// Sort keys by index
		sort.SliceStable(dbAccount.Keys, func(i, j int) bool {
			return dbAccount.Keys[i].Index < dbAccount.Keys[j].Index
		})

		// Push publickeys to args and prepare db update
		for i := 0; i < cloneCount; i++ {
			pbk, err := cadence.NewString(sourceKey.PublicKey[2:]) // TODO: use a helper function to trim "0x" prefix
			if err != nil {
				return 0, "", err
			}
			pbks = append(pbks, pbk)

			// Create cloned account key & update index
			cloned := keys.Storable{
				ID:             0, // Reset ID to create a new key to DB
				AccountAddress: sourceKey.AccountAddress,
				Index:          dbAccount.Keys[len(dbAccount.Keys)-1].Index + 1,
				Type:           sourceKey.Type,
				Value:          sourceKey.Value,
				PublicKey:      sourceKey.PublicKey,
				SignAlgo:       sourceKey.SignAlgo,
				HashAlgo:       sourceKey.HashAlgo,
			}

			dbAccount.Keys = append(dbAccount.Keys, cloned)
		}

		// Prepare transaction arguments
		x := cadence.NewArray(pbks)
		args := []transactions.Argument{x}

		entry.WithFields(log.Fields{"args": args}).Debug("args prepared")

		// NOTE: sync, so will wait for transaction to be sent & sealed
		_, tx, err := s.txs.Create(ctx, true, dbAccount.Address, code, args, transactions.General)
		if err != nil {
			entry.WithFields(log.Fields{"err": err}).Error("failed to create transaction")
			return 0, tx.TransactionId, err
		}

		// Update account in database
		// TODO: if update fails, should sync keys from chain later
		err = s.store.SaveAccount(&dbAccount)
		if err != nil {
			entry.WithFields(log.Fields{"err": err}).Error("failed to update account in database")
			return 0, tx.TransactionId, err
		}

		return len(dbAccount.Keys), tx.TransactionId, err
	} else if len(validKeys) > numKeys {
		entry.Debug("too many valid keys", len(validKeys), " vs. ", numKeys)
	} else {
		entry.Debug("correct number of keys")
		return numKeys, "", nil
	}

	return 0, "", nil
}

// createAccount creates a new account on the flow blockchain. It generates a
// fresh key pair and constructs a flow transaction to create the account with
// generated key. Admin account is used to pay for the transaction.
//
// Returns created account and the flow transaction ID of the account creation.
func (s *ServiceImpl) createAccount(ctx context.Context) (*Account, string, error) {
	account := &Account{Type: AccountTypeCustodial}

	// Important to ratelimit all the way up here so the keys and reference blocks
	// are "fresh" when the transaction is actually sent
	s.txRateLimiter.Take()

	payer, err := s.km.AdminAuthorizer(ctx)
	if err != nil {
		return nil, "", err
	}

	proposer, err := s.km.AdminProposalKey(ctx)
	if err != nil {
		return nil, "", err
	}

	// Get latest blocks blockID as reference blockID
	referenceBlockID, err := flow_helpers.LatestBlockId(ctx, s.fc)
	if err != nil {
		return nil, "", err
	}

	// Generate a new key pair
	accountKey, newPrivateKey, err := s.km.GenerateDefault(ctx)
	if err != nil {
		return nil, "", err
	}

	// Public keys for creating the account
	publicKeys := []*flow.AccountKey{}

	// Create copies based on the configured key count, changing just the index
	for i := uint32(0); i < uint32(s.cfg.DefaultAccountKeyCount); i++ {
		clonedAccountKey := *accountKey
		clonedAccountKey.Index = i

		publicKeys = append(publicKeys, &clonedAccountKey)
	}

	flowTx, flowTxErr := flow_templates.CreateAccount(
		publicKeys,
		nil,
		payer.Address,
	)

	if err != flowTxErr {
		return nil, "", err
	}

	flowTx.
		SetReferenceBlockID(*referenceBlockID).
		SetProposalKey(proposer.Address, proposer.Key.Index, proposer.Key.SequenceNumber).
		SetPayer(payer.Address).
		SetComputeLimit(maxGasLimit)

	// Check if we want to use a custom account create script
	if s.cfg.ScriptPathCreateAccount != "" {
		bytes, err := os.ReadFile(s.cfg.ScriptPathCreateAccount)
		if err != nil {
			return nil, "", err
		}
		// Overwrite the existing script
		flowTx.SetScript(bytes)
	}

	// Proposer signs the payload (unless proposer == payer).
	if !proposer.Equals(payer) {
		if err := flowTx.SignPayload(proposer.Address, proposer.Key.Index, proposer.Signer); err != nil {
			return nil, "", err
		}
	}

	// Payer signs the envelope
	if err := flowTx.SignEnvelope(payer.Address, payer.Key.Index, payer.Signer); err != nil {
		return nil, "", err
	}

	// Send and wait for the transaction to be sealed
	result, err := flow_helpers.SendAndWait(ctx, s.fc, *flowTx, s.cfg.TransactionTimeout)
	if err != nil {
		return nil, "", err
	}

	// Grab the new address from transaction events
	var newAddress flow.Address
	for _, event := range result.Events {
		if event.Type == flow.EventAccountCreated {
			accountCreatedEvent := flow.AccountCreatedEvent(event)
			newAddress = accountCreatedEvent.Address()
			break
		}
	}

	// Check that we actually got a new address
	if newAddress == flow.EmptyAddress {
		return nil, "", fmt.Errorf("something went wrong when waiting for address")
	}

	account.Address = flow_helpers.FormatAddress(newAddress)

	// Convert the key to storable form (encrypt it)
	encryptedAccountKey, err := s.km.Save(*newPrivateKey)
	if err != nil {
		return nil, "", err
	}
	encryptedAccountKey.PublicKey = accountKey.PublicKey.String()

	// Store account and key(s)
	// Looping through accountKeys to get the correct Index values
	storableKeys := []keys.Storable{}
	for _, pbk := range publicKeys {
		clonedEncryptedAccountKey := encryptedAccountKey
		clonedEncryptedAccountKey.Index = pbk.Index
		storableKeys = append(storableKeys, clonedEncryptedAccountKey)
	}

	account.Keys = storableKeys
	if err := s.store.InsertAccount(account); err != nil {
		return nil, "", err
	}

	AccountAdded.Trigger(AccountAddedPayload{
		Address: flow.HexToAddress(account.Address),
	})

	log.WithFields(log.Fields{"address": account.Address}).Debug("Account created")

	return account, flowTx.ID().String(), nil
}

// AddNewKey adds a new key to the given account
func (s *ServiceImpl) AddNewKey(ctx context.Context, address flow.Address) (*jobs.Job, error) {
	fmt.Println("AddNewKey called")
	// entry := log.WithFields(log.Fields{"address": address, "function": "ServiceImpl.AddNewKey"})

	attrs := addNewKeyJobAttributes{Address: address}
	attrBytes, err := json.Marshal(attrs)
	if err != nil {
		return nil,  err
	}

	// make it always async
	job, err := s.wp.CreateJob(AddNewKeyJobType, "", jobs.WithAttributes(attrBytes))
	if err != nil {
		return nil, err
	}

	err = s.wp.Schedule(job)
	if err != nil {
		return nil, err
	}

	return job, nil
}

func (s *ServiceImpl) addKey(ctx context.Context, logEntry *log.Entry, address flow.Address) (*Account, error) {
	// Get stored account
	dbAccount, err := s.store.Account(flow_helpers.FormatAddress(address))

	if err != nil {
		logEntry.WithFields(log.Fields{"err": err}).Error("failed to get account from database")
		fmt.Println("Error fetching account from database", err)
		return &Account{}, err
	}

	// Get the first existing key to use as the source of the tx
	sort.SliceStable(dbAccount.Keys, func(i, j int) bool {
		return dbAccount.Keys[i].Index < dbAccount.Keys[j].Index
	})
	sourceKey := dbAccount.Keys[0] // NOTE: Only valid (not revoked) keys should be stored in the database
	sourceKeyPbkString := strings.TrimPrefix(sourceKey.PublicKey, "0x")
	_, err = flow_crypto.DecodePublicKeyHex(flow_crypto.StringToSignatureAlgorithm(sourceKey.SignAlgo), sourceKeyPbkString)
	if err != nil {
		logEntry.WithFields(log.Fields{"err": err, "sourceKeyPbkString": sourceKeyPbkString}).Error("failed to decode public key for source key")
		fmt.Println("Error decoding public key for source key", err)
		return &Account{}, err
	}
	logEntry.WithFields(log.Fields{"sourceKeyPbkString": sourceKeyPbkString}).Debug("source key selected")

	// Generate a new key pair
	newAccountKey, newPrivateKey, err := s.km.GenerateDefault(ctx)
	if err != nil {
		logEntry.WithFields(log.Fields{"err": err}).Error("failed to generate new key")
		fmt.Println("Error generating default key", err)
		return &Account{}, err
	}

	// Convert the key to storable form (encrypt it)
	encryptedAccountKey, err := s.km.Save(*newPrivateKey)
	if err != nil {
		return &Account{}, err
	}
	encryptedAccountKey.PublicKey = newAccountKey.PublicKey.String()

	// Get the next index for the new key
	nextIndex, err := s.getNextIndex(ctx, logEntry, address)
	if err != nil {
		logEntry.WithFields(log.Fields{"err": err}).Error("failed to get next index")
		fmt.Println("Error getting next index", err)
		return &Account{}, err
	}

	dbKey := keys.Storable{
		ID:             0, // Reset ID to create a new key to DB
		AccountAddress: dbAccount.Address,
		Index:          nextIndex,
		Type:           "local",
		Value:          encryptedAccountKey.Value,
		PublicKey:      encryptedAccountKey.PublicKey,
		SignAlgo:       encryptedAccountKey.SignAlgo,
		HashAlgo:       encryptedAccountKey.HashAlgo,
	}
	dbAccount.Keys = append(dbAccount.Keys, dbKey)

	addTx, addTxErr := s.createNewKeyTx(ctx, logEntry, dbAccount.Address, newAccountKey)
	if addTxErr != nil {
		logEntry.WithFields(log.Fields{"err": addTxErr, "address": dbAccount.Address}).Error("failed to create transaction")
		fmt.Println("Error creating transaction", addTxErr)
		return &Account{}, addTxErr
	}
	logEntry.WithFields(log.Fields{"txID": addTx.TransactionId}).Info("transaction created")

	// Update account in database
	err = s.store.SaveAccount(&dbAccount)
	if err != nil {
		logEntry.WithFields(log.Fields{"err": err}).Error("failed to update account in database")
		fmt.Println("Error updating account in database", err)
		return &Account{}, err
	}

	return &dbAccount, nil
}

func (s *ServiceImpl) RevokeKey(ctx context.Context, address flow.Address, oldKeyIndex uint32) (*jobs.Job, error) {
	fmt.Println("RevokeKey called")
	// entry := log.WithFields(log.Fields{"address": address, "function": "ServiceImpl.RevokeKey"})
	attrs := revokeKeyJobAttributes{Address: address, OldKeyIndex: oldKeyIndex}
	attrBytes, err := json.Marshal(attrs)
	if err != nil {
		return nil, err
	}
	// make it always async
	job, err := s.wp.CreateJob(RevokeKeyJobType, "", jobs.WithAttributes(attrBytes))
	if err != nil {
		return nil, err
	}

	err = s.wp.Schedule(job)
	if err != nil {
		return nil, err
	}

	return job, nil
}

func (s *ServiceImpl) revokeKey(ctx context.Context, logEntry *log.Entry, address flow.Address, oldKeyIndex uint32) (*Account, error) {
	// Get stored account
	dbAccount, err := s.store.Account(flow_helpers.FormatAddress(address))
	if err != nil {
		logEntry.WithFields(log.Fields{"err": err}).Error("failed to get account from database")
		fmt.Println("Error fetching account from database", err)
		return &Account{}, err
	}

	if len(dbAccount.Keys) == 1 {
		return nil, fmt.Errorf("account %s only has one key, cannot revoke", dbAccount.Address)
	}

	var indexFound bool
	var keyToDelete *keys.Storable
	for _, key := range dbAccount.Keys {
		if key.Index == oldKeyIndex {
			indexFound = true
			keyToDelete = &key
		}
	}

	if !indexFound {
		logEntry.WithFields(log.Fields{"err": err}).Error("failed to find key index in database")
		return &Account{}, fmt.Errorf("failed to find key %d index in database for account %s ", oldKeyIndex, dbAccount.Address)
	}

	revokeTx, revokeTxErr := s.createRevokeKeyTx(ctx, logEntry, dbAccount.Address, oldKeyIndex)
	if revokeTxErr != nil {
		logEntry.WithFields(log.Fields{"err": revokeTxErr}).Error("failed to create transaction")
		fmt.Println("Error creating transaction", revokeTxErr)
		return &Account{}, revokeTxErr
	}
	logEntry.WithFields(log.Fields{"txID": revokeTx.TransactionId}).Info("transaction created")

	// Remove the old key from the db
	err = s.store.DeleteKeyForAccount(&dbAccount, keyToDelete)
	if err != nil {
		logEntry.WithFields(log.Fields{"err": err}).Error("failed to delete key from database")
		fmt.Println("Error deleting key from database", err)
		return &Account{}, err
	}

	return &dbAccount, nil
}

// getNextIndex calculates the next key index for the given account based on the number of keys for that account onchain. Though this will probably not be an issue in production, in test data there are keys missing from the db that are on the chain and this causes some mismatches in indexes. So this approach should be more accurate.
func (s *ServiceImpl) getNextIndex(ctx context.Context, logEntry *log.Entry, address flow.Address) (uint32, error) {
	// Get flow account from client
	flowAccount, err := s.fc.GetAccount(ctx, address)
	if err != nil {
		logEntry.WithFields(log.Fields{"err": err}).Error("failed to get Flow account")
		return 0, err
	}

	flowAccountKeys := flowAccount.Keys
	sort.SliceStable(flowAccountKeys, func(i, j int) bool {
		return flowAccountKeys[i].Index < flowAccountKeys[j].Index
	})

	nextIndex := flowAccount.Keys[len(flowAccount.Keys)-1].Index + 1
	return nextIndex, nil
}

func (s *ServiceImpl) createNewKeyTx(ctx context.Context, logEntry *log.Entry, accountAddress string, newAccountKey *flow.AccountKey) (*transactions.Transaction, error) {

	// Prepare transaction arguments
	keyAsKeyListEntry, kErr := templates.AccountKeyToCadenceCryptoKey(newAccountKey)
	if kErr != nil {
		fmt.Println("Error converting account key to cadence crypto key", kErr)
		return nil, kErr
	}

	args := []transactions.Argument{keyAsKeyListEntry}
	logEntry.WithFields(log.Fields{"args": args}).Info("args prepared")

	// Create & send add key transaction
	code := t.AddAccountKey
	sync := true
	_, tx, err := s.txs.Create(ctx, sync, accountAddress, code, args, transactions.General)

	if err != nil {
		logEntry.WithFields(log.Fields{"err": err}).Error("failed to create transaction")
		return &transactions.Transaction{}, err
	}

	return tx, nil
}

func (s *ServiceImpl) createRevokeKeyTx(ctx context.Context, logEntry *log.Entry, accountAddress string, oldKeyIndex uint32) (*transactions.Transaction, error) {
	indexAsCadenceValue := cadence.NewInt(int(oldKeyIndex))
	args := []transactions.Argument{indexAsCadenceValue}
	logEntry.WithFields(log.Fields{"args": args}).Info("args prepared")

	// Create transaction
	code := t.RemoveAccountKey
	sync := true
	// this is by default using the 'least recently used key' for the transaction
	// since we are just creating a new key, that is the one that should be used
	// flow-wallet-api/keys/basic/keys.go#L165
	// FIXME: make this more explicit about which key is signing the revoke tx
	// it is possible that it will use the key that is being revoked, which could mean that the tx fails
	// if the tx is tried again, it will work, since that key is no longer the 'least recently used' key
	// but this is confusing and not ideal
	_, tx, err := s.txs.Create(ctx, sync, accountAddress, code, args, transactions.General)

	if err != nil {
		logEntry.WithFields(log.Fields{"err": err}).Error("failed to create transaction")
		return &transactions.Transaction{}, err
	}

	return tx, nil
}

func (s *ServiceImpl) GetKeysByType(ctx context.Context, keyType string) ([]keys.Storable, error) {
	return s.store.GetKeysByType(keyType)
}
