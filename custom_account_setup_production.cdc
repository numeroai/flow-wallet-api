import NonFungibleToken from 0x1d7e57aa55817448
import Electables from 0x6b3fe09edaf89937

transaction(publicKeys: [String], contracts: {String: String}) {
	prepare(signer: AuthAccount) {
		let acct = AuthAccount(payer: signer)

		for key in publicKeys {
			acct.addPublicKey(key.decodeHex())
		}

		for contract in contracts.keys {
			acct.contracts.add(name: contract, code: contracts[contract]!.decodeHex())
		}

        if acct.borrow<&Electables.Collection>(from: Electables.CollectionStoragePath) == nil {
            // create a new empty collection
            let collection <- Electables.createEmptyCollection()
            
            // save it to the account
            acct.save(<- collection, to: Electables.CollectionStoragePath)

            // Creates a public capability for the collection so that other users can publicly access electable attributes.
            // The pieces inside of the brackets specify the type of the linked object, and only expose the fields and
            // functions on those types.
            acct.link<&Electables.Collection{NonFungibleToken.CollectionPublic, Electables.ElectablesPublicCollection}>(
                Electables.CollectionPublicPath, target: Electables.CollectionStoragePath
            )
        }
	}
}