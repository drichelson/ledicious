package animation

import (
	"encoding/json"
)

type Control struct {
	Vars   *map[string]int
	Colors *map[string]string
}

func (c *Control) State() string {
	jsonBytes, _ := json.Marshal(c)
	return string(jsonBytes)
}

func (c *Control) Load(jsonString string) {
	json.Unmarshal([]byte(jsonString), c)
}
