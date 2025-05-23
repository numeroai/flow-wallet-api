package tokens

import (
	"encoding/json"

	"github.com/onflow/cadence"
)

type Balance struct {
	CadenceValue cadence.Value
}

func (b *Balance) MarshalJSON() ([]byte, error) {
	if b.CadenceValue == nil {
		// Not able to omit the balance field as it is in a "parent" struct
		// So using JSON null here
		return json.Marshal(nil)
	}

	// Only handle fixed point numbers differently, rest can use the default
	// _, isUfix64 := b.CadenceValue.Type().(cadence.UFix64Type)
	// _, isFix64 := b.CadenceValue.Type().(cadence.Fix64Type)

	// return json.Marshal(b.CadenceValue.ToGoValue())

	// todo: Is it okay to always return balance's as a string?
	return json.Marshal(b.CadenceValue.String())

}
