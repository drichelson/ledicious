package animation

import (
	"encoding/json"
	"github.com/lucasb-eyer/go-colorful"
	"log"
	"strings"
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

func (c *Control) GetColor(colorVar string) colorful.Color {
	hex := "#" + (*c.Colors)[colorVar]
	color, err := colorful.Hex(hex)
	if err != nil {
		log.Printf("Got error when parsing color: %s %v", hex, err)
	}
	return color
}

func (c *Control) SetColor(colorVar string, color colorful.Color) {
	(*c.Colors)[colorVar] = strings.TrimLeft(color.Hex(), "#")
}
