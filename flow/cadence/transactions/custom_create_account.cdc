import Crypto

transaction(publicKeys: [Crypto.KeyListEntry], contracts: {String: String}) {
	prepare(signer: auth(Storage, Capabilities) &Account) {
		panic("Account initialized with custom script")
	}
}
