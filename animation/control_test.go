package animation

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestJsonRoundTrip(t *testing.T) {
	jsonString := `
	{"Vars":{"A":500,"B":500,"C":500,"D":500},"Colors":{"A":"5c0d5c","B":"00ff00","C":"0000ff","D":"000000"}}
	`

	var c Control
	c.Load(jsonString)
	actualJson := c.State()
	assert.JSONEq(t, jsonString, actualJson)
}
