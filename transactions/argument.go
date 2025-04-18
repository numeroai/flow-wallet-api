package transactions

import (
	"encoding/json"
	"fmt"

	"github.com/onflow/cadence"
	c_json "github.com/onflow/cadence/encoding/json"
)

type Argument interface{}

func ArgAsCadence(a Argument) (cadence.Value, error) {
	c, ok := a.(cadence.Value)
	if ok {
		return c, nil
	}

	s, ok := a.(string)
	if ok {
		c, err := c_json.Decode(nil, []byte(s))
		if err != nil {
			fmt.Println("THE CJSON ERROR: ", err)
			return cadence.Void{}, err
		}

		fmt.Println("this is the cjson decoded value: ", c)

		return c, nil
	}

	// // Convert to json bytes so we can use cadence's own encoding library
	j, err := json.Marshal(a)
	if err != nil {
		return cadence.Void{}, err
	}

	// Use cadence's own encoding library
	// is this memory guage ok?
	c, err = c_json.Decode(nil, j)
	if err != nil {
		return cadence.Void{}, err
	}

	return c, nil
}

func MustDecodeArgs(aa []Argument) []cadence.Value {
	var cc []cadence.Value

	for _, a := range aa {
		c, err := ArgAsCadence(a)
		if err != nil {
			panic("unable to decode arguments")
		}
		cc = append(cc, c)
	}

	return cc
}
