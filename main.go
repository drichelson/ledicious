package main

import (
	"fmt"
	"github.com/drichelson/ledicious/animation"
	"gopkg.in/macaron.v1"
	"log"
	"net/http"
	"strconv"
)

var (
	//vars    = make(map[string]float64)
	//colors  = make(map[string]string)
	control = animation.NewControl()
)

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Lshortfile)
	control.SetVar("A", 0.5)
	control.SetVar("B", 0.5)
	control.SetVar("C", 0.5)
	control.SetVar("D", 0.5)

	control.SetColorHex("A", "ff00FF")
	control.SetColorHex("B", "ff00FF")
	control.SetColorHex("C", "ff00FF")
	control.SetColorHex("D", "ff00FF")

	m := macaron.Classic()
	m.Use(macaron.Static("assets",
		macaron.StaticOptions{
			// Prefix is the optional prefix used to serve the static directory content. Default is empty string.
			Prefix: "",
			// SkipLogging will disable [Static] log messages when a static file is served. Default is false.
			SkipLogging: true,
			// IndexFile defines which file to serve as index if it exists. Default is "index.html".
			IndexFile: "index.html",
		}))

	m.Get("/wow", func(ctx *macaron.Context) string {
		log.Println("wow: " + control.State())
		return ""
	})

	m.Get("/varA", func(ctx *macaron.Context) string {
		return getVar(ctx, "A")
	})
	m.Get("/varB", func(ctx *macaron.Context) string {
		return getVar(ctx, "B")
	})
	m.Get("/varC", func(ctx *macaron.Context) string {
		return getVar(ctx, "C")
	})
	m.Get("/varD", func(ctx *macaron.Context) string {
		return getVar(ctx, "D")
	})

	m.Get("/colorA", func(ctx *macaron.Context) string {
		return getColor(ctx, "A")
	})
	m.Get("/colorB", func(ctx *macaron.Context) string {
		return getColor(ctx, "B")
	})
	m.Get("/colorC", func(ctx *macaron.Context) string {
		return getColor(ctx, "C")
	})
	m.Get("/colorD", func(ctx *macaron.Context) string {
		return getColor(ctx, "D")
	})
	go m.Run()
	animation.Start(control)
}

// Generic handler for getting/setting vars.
// Use with GET to retrieve the var
// Use with PUT with query param state=<newVal> to set var.
func getVar(ctx *macaron.Context, varName string) string {
	ctx.Header().Set("Content-Type", "application/json")
	newValString := ctx.Query("state")
	if newValString == "" {
		return "{\"state\": \"" + strconv.Itoa(int(control.GetVar(varName)*1000.0)) + "\"}"
	}
	newVal, err := strconv.Atoi(newValString)
	if err != nil {
		ctx.Resp.WriteHeader(http.StatusBadRequest)
		return "not a number!"
	}
	control.SetVar(varName, float64(newVal)/1000.0)
	fmt.Printf("new value: %s %d\n", varName, newVal)
	log.Println(control.State())
	return "{\"state\": \"" + newValString + "\"}"
}

func getColor(ctx *macaron.Context, varName string) string {
	ctx.Header().Set("Content-Type", "application/json")
	newVal := ctx.Query("state")
	if newVal == "" {
		return "{\"state\": \"" + control.GetColorHex(varName) + "\"}"
	}
	control.SetColorHex(varName, newVal)
	fmt.Printf("new color: %s %s\n", varName, newVal)
	log.Println(control.State())
	return "{\"state\": \"" + newVal + "\"}"
}
