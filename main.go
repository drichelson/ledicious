package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/drichelson/ledicious/animation"
	"gopkg.in/macaron.v1"
)

var (
	control = animation.NewControl()
	wowLog  log.Logger
)

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Lshortfile)
	wowLog.SetFlags(log.Ltime | log.Ldate)

	//create your file with desired read/write permissions
	f, err := os.OpenFile("wowLog.txt", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	wowLog.SetOutput(f)

	control.SetVar("varA", 0.5)
	control.SetVar("varB", 0.5)
	control.SetVar("varC", 0.5)
	control.SetVar("varD", 0.5)
	control.SetVar("brightness", 1.0)
	control.SetVar("speed", 0.3)

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
		wowLog.Println(ctx.Req.URL.RawQuery)
		return ""
	})

	m.Get("/speed", func(ctx *macaron.Context) string {
		return getVar(ctx, "speed")
	})
	m.Get("/brightness", func(ctx *macaron.Context) string {
		return getVar(ctx, "brightness")
	})
	m.Get("/varA", func(ctx *macaron.Context) string {
		return getVar(ctx, "varA")
	})
	m.Get("/varB", func(ctx *macaron.Context) string {
		return getVar(ctx, "varB")
	})
	m.Get("/varC", func(ctx *macaron.Context) string {
		return getVar(ctx, "varC")
	})
	m.Get("/varD", func(ctx *macaron.Context) string {
		return getVar(ctx, "varD")
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
	//fmt.Printf("new value: %s %d\n", varName, newVal)
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
	//fmt.Printf("new color: %s %s\n", varName, newVal)
	log.Println(control.State())
	return "{\"state\": \"" + newVal + "\"}"
}
