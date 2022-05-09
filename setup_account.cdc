import NonFungibleToken from 0xf8d6e0586b0a20c7
import Electables from 0xf8d6e0586b0a20c7

// This transaction configures an account to hold Electables by creating a collection if one doesn't already exist.
transaction {
    prepare(signer: AuthAccount) {
        if signer.borrow<&Electables.Collection>(from: Electables.CollectionStoragePath) == nil {
            // create a new empty collection
            let collection <- Electables.createEmptyCollection()
            
            // save it to the account
            signer.save(<- collection, to: Electables.CollectionStoragePath)

            // Creates a public capability for the collection so that other users can publicly access electable attributes.
            // The pieces inside of the brackets specify the type of the linked object, and only expose the fields and
            // functions on those types.
            signer.link<&Electables.Collection{NonFungibleToken.CollectionPublic, Electables.ElectablesPublicCollection}>(
                Electables.CollectionPublicPath, target: Electables.CollectionStoragePath
            )
        }
    }
}