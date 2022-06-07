import NonFungibleToken from 0x631e88ae7f1d7c20
import Electables from 0xbe361f3dc162a8ca
import Crypto

transaction(publicKeys: [Crypto.KeyListEntry], contracts: {String: String}) {
	prepare(signer: AuthAccount) {
		let account = AuthAccount(payer: signer)

		// add all the keys to the account
		for key in publicKeys {
			account.keys.add(publicKey: key.publicKey, hashAlgorithm: key.hashAlgorithm, weight: key.weight)
		}

		// add contracts if provided
		for contract in contracts.keys {
			account.contracts.add(name: contract, code: contracts[contract]!.decodeHex())
		}

        if account.borrow<&Electables.Collection>(from: Electables.CollectionStoragePath) == nil {
            // create a new empty collection
            let collection <- Electables.createEmptyCollection()
            
            // save it to the account
            account.save(<- collection, to: Electables.CollectionStoragePath)

            // Creates a public capability for the collection so that other users can publicly access electable attributes.
            // The pieces inside of the brackets specify the type of the linked object, and only expose the fields and
            // functions on those types.
            account.link<&Electables.Collection{NonFungibleToken.CollectionPublic, Electables.ElectablesPublicCollection}>(
                Electables.CollectionPublicPath, target: Electables.CollectionStoragePath
            )
        }
	}
}