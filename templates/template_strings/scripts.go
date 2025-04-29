package template_strings

const GenericFungibleBalance = `
import FungibleToken from "./FungibleToken.cdc"
import TOKEN_DECLARATION_NAME from TOKEN_ADDRESS

access(all) fun main(account: Address): UFix64 {

    let vaultRef = getAccount(account)
        .getCapability(/public/TOKEN_BALANCE)
        .borrow<&TOKEN_DECLARATION_NAME.Vault{FungibleToken.Balance}>()
        ?? panic("failed to borrow reference to vault")

    return vaultRef.balance
}
`
