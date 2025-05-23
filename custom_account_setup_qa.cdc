import NonFungibleToken from 0x631e88ae7f1d7c20
import Electables from 0x4c05c3d3499ca274

transaction(publicKeys: [String], contracts: {String: String}) {
	prepare(signer: auth(Storage, Capabilities) &Account) {
		let acct = Account(payer: signer)

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