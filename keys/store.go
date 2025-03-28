package keys

// Store is the interface required by key manager for data storage.
type Store interface {
	AccountKey(address string) (Storable, error)
	ProposalKeyIndex(limitKeyCount int) (uint32, error)
	ProposalKeyCount() (int64, error)
	InsertProposalKey(proposalKey ProposalKey) error
	DeleteAllProposalKeys() error
}
