package animation

import (
	"encoding/json"
	"github.com/lucasb-eyer/go-colorful"
	"log"
	"strings"
	"sync"
)

type Control struct {
	Vars   map[string]float64
	Colors map[string]string
	mu     *sync.Mutex
}

func NewControl() Control {
	return Control{
		Vars:   make(map[string]float64),
		Colors: make(map[string]string),
		mu:     &sync.Mutex{}}
}

func (c *Control) State() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	jsonBytes, _ := json.Marshal(c)
	return string(jsonBytes)
}

// Load values from json
func (c *Control) Load(jsonString string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	json.Unmarshal([]byte(jsonString), c)
}

func (c *Control) GetVar(key string) float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Vars[key]
}

func (c *Control) SetVar(key string, val float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Vars[key] = val
}

func (c *Control) GetColor(colorVar string) colorful.Color {
	c.mu.Lock()
	defer c.mu.Unlock()
	hex := "#" + c.Colors[colorVar]
	color, err := colorful.Hex(hex)
	if err != nil {
		log.Printf("Got error when parsing color: %s %v", hex, err)
	}
	return color
}

func (c *Control) SetColor(colorVar string, color colorful.Color) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Colors[colorVar] = strings.TrimLeft(color.Hex(), "#")
}

//Expects a 6 digit hex color without the leading #
func (c *Control) SetColorHex(colorVar string, color string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Colors[colorVar] = color
}

//Returns 6 digit hex color without the leading #
func (c *Control) GetColorHex(colorVar string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Colors[colorVar]
}
