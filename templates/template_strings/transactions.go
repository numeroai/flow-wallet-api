package template_strings

const AddAccountContractWithAdmin = `
transaction(name: String, code: String) {
	prepare(signer: auth(AddContract) &Account) {
		signer.contracts.add(name: name, code: code.decodeHex(), adminAccount: signer)
	}
}
`

const CreateAccount = `
transaction(publicKeys: [String]) {
	prepare(signer: auth(BorrowValue) &Account) {
    let acct = Account(payer: signer)


		for key in publicKeys {
			acct.addPublicKey(key.decodeHex())
		}
	}
}
`

const GenericFungibleTransfer = `
import FungibleToken from "./FungibleToken.cdc"
import TOKEN_DECLARATION_NAME from TOKEN_ADDRESS

transaction(amount: UFix64, recipient: Address) {
  let sentVault: @{FungibleToken.Vault}

  prepare(signer: auth(BorrowValue) &Account) {
    let vaultRef = signer.storage.borrow<auth(FungibleToken.Withdraw) &TOKEN_DECLARATION_NAME.Vault>(from: /storage/TOKEN_VAULT)
      ?? panic("failed to borrow reference to sender vault")

    self.sentVault <- vaultRef.withdraw(amount: amount)
  }

  execute {
    let receiverRef = getAccount(recipient)
      .capabilities
      .borrow<&{FungibleToken.Receiver}>(/public/TOKEN_RECEIVER)
      ?? panic("failed to borrow reference to recipient vault")

    receiverRef.deposit(from: <-self.sentVault)
  }
}
`

const GenericFungibleSetup = `
import FungibleToken from "./FungibleToken.cdc"
import TOKEN_DECLARATION_NAME from TOKEN_ADDRESS

transaction {
  prepare(signer: auth(BorrowValue) &Account) {

    let existingVault = signer.borrow<&TOKEN_DECLARATION_NAME.Vault>(from: /storage/TOKEN_VAULT)

    if (existingVault != nil) {
        panic("vault exists")
    }

    signer.save(<-TOKEN_DECLARATION_NAME.createEmptyVault(), to: /storage/TOKEN_VAULT)

    signer.link<&TOKEN_DECLARATION_NAME.Vault>(
      /public/TOKEN_RECEIVER,
      target: /storage/TOKEN_VAULT
    )

    signer.link<&TOKEN_DECLARATION_NAME.Vault>(
      /public/TOKEN_BALANCE,
      target: /storage/TOKEN_VAULT
    )
  }
}
`

const AddProposalKeyTransaction = `
transaction(adminKeyIndex: Int, numProposalKeys: UInt16) {
  prepare(account: auth(Keys) &Account) {
    let key = account.keys.get(keyIndex: adminKeyIndex)!
    var count: UInt16 = 0
    while count < numProposalKeys {
      account.keys.add(
            publicKey: key.publicKey,
            hashAlgorithm: key.hashAlgorithm,
            weight: 0.0
        )
        count = count + 1
    }
  }
}
`

// TODO: sigAlgo & hashAlgo as params, add pre-&post-conditions
const AddAccountKeysTransaction = `
transaction(publicKeys: [String]) {
 	prepare(signer: auth(AddKey) &Account) {
    for pbk in publicKeys {
      let key = PublicKey(
        publicKey: pbk.decodeHex(),
        signatureAlgorithm: SignatureAlgorithm.ECDSA_P256
      )

      signer.keys.add(
        publicKey: key,
        hashAlgorithm: HashAlgorithm.SHA3_256,
        weight: 1000.0
      )
    }
  }
}
`
