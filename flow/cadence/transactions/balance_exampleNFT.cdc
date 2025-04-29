import NonFungibleToken from "../contracts/NonFungibleToken.cdc"
import ExampleNFT from "../contracts/ExampleNFT.cdc"

access(all) fun main(account: Address): [UInt64] {
    let receiver = getAccount(account)
        .capabilities.get(ExampleNFT.CollectionPublicPath)!
        .borrow<&{NonFungibleToken.CollectionPublic}>()!

    return receiver.getIDs()
}
