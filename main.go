package main

import (
	"fmt"
	"github.com/drichelson/usb-test/animation"
	"gopkg.in/macaron.v1"
	"log"
	"net/http"
	"strconv"
	"sync"
)

var (
	webVar1 = 0
	webVar2 = 0
	webLock = &sync.RWMutex{}
)

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Lshortfile)

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

	m.Get("/var", func(ctx *macaron.Context) string {
		webLock.RLock()
		defer webLock.RUnlock()
		ctx.Header().Set("Content-Type", "application/json")
		newValString := ctx.Query("state")
		if newValString == "" {
			return "{\"state\": \"" + strconv.Itoa(webVar1) + "\"}"
		}
		newVal, err := strconv.Atoi(newValString)
		if err != nil {
			ctx.Resp.WriteHeader(http.StatusBadRequest)
			return "not a number!"
		}
		webVar1 = newVal
		fmt.Printf("new value: %d\n", newVal)
		return "{\"state\": \"" + newValString + "\"}"

	})

	m.Put("/var", func(ctx *macaron.Context) string {
		webLock.Lock()
		defer webLock.Unlock()
		ctx.Header().Set("Content-Type", "application/json")

		newValString := ctx.Query("state")
		newVal, err := strconv.Atoi(newValString)
		if err != nil {
			ctx.Resp.WriteHeader(http.StatusBadRequest)
			return "not a number!"
		}
		webVar1 = newVal
		fmt.Printf("new value: %d", newVal)
		return "{\"state\": \"" + newValString + "\"}"

	})

	go m.Run()
	animation.Start()
}

//Teensy:
// descriptor: &{Length:18 DescriptorType:Device descriptor. USBSpecification:0x0200 (2.00) DeviceClass:Communications class. DeviceSubClass:0 DeviceProtocol:0 MaxPacketSize0:64 VendorID:5824 ProductID:1155 DeviceReleaseNumber:0x0100 (1.00) ManufacturerIndex:1 ProductIndex:2 SerialNumberIndex:3 NumConfigurations:1}
