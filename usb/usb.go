package usb

import (
	"log"
	"math"

	"github.com/drichelson/libusb"
	"github.com/google/gousb"

	"github.com/lucasb-eyer/go-colorful"
)

//Teensy:
// descriptor: &{Length:18 DescriptorType:Device descriptor. USBSpecification:0x0200 (2.00) DeviceClass:Communications class. DeviceSubClass:0 DeviceProtocol:0 MaxPacketSize0:64 VendorID:5824 ProductID:1155 DeviceReleaseNumber:0x0100 (1.00) ManufacturerIndex:1 ProductIndex:2 SerialNumberIndex:3 NumConfigurations:1}

const (
	teensyVendorID  = 5824
	teensyProductID = 1155
	minimumVisible  = .007843137
)

var (
	ctx          *libusb.Context
	deviceHandle *libusb.DeviceHandle
	device       *gousb.Device
	endpoint     *gousb.OutEndpoint
)

type msgID uint8

func Initialize() {
	//ShowVersion()
	var err error
	ctx := gousb.NewContext()
	//ctx, err = gousb.InInit()
	//if err != nil {
	//	log.Fatalf("error initializing libusb: %v", err)
	//}
	device, err = ctx.OpenDeviceWithVIDPID(teensyVendorID, teensyProductID)

	//_, deviceHandle, err = ctx.OpenDeviceWithVendorProduct(teensyVendorID, teensyProductID)
	if err != nil || device == nil {
		log.Fatalf("Error opening device: %v", err)
	}
	log.Printf("Opened device: %s with description: %s", device.String(), device.Desc.String())
	config, err := device.Config(1)
	if err != nil || device == nil {
		log.Fatalf("Error getting config from device: %v", err)
	}
	log.Printf("Config description: %s", config.Desc.String())

	intf, err := config.Interface(1, 0)
	if err != nil || device == nil {
		log.Fatalf("Error getting default interface from device: %v", err)
	}
	endpoint, err = intf.OutEndpoint(3)
	if err != nil || device == nil {
		log.Fatalf("Error getting out endpoint from interface: %v", err)
	}
	log.Printf("Endpoint: %s", endpoint.Desc.String())
	//showInfo(ctx, "Teensy", teensyVendorID, teensyProductID)
	//err = deviceHandle.ClaimInterface(1)
	//if err != nil {
	//	log.Fatalf("Error claiming bulk transfer interface: %v", err)
	//}
}

func normalizeBrightness(color colorful.Color) (r, g, b uint8) {
	return normalize(color.R), normalize(color.G), normalize(color.B)
}

func normalize(in float64) uint8 {
	//TODO: use a lookup table instead? check performance on arm before
	return uint8(255.0 * math.Pow(in, 1.08))
}

func Render(pixels []colorful.Color, brightness float64) {
	//fmt.Printf("color count: %d\n", len(pixels))
	data := make([]byte, len(pixels)*3+3)
	data[0] = '*'
	data[1] = 238
	data[2] = 2

	for i, c := range pixels {
		//if c == nil {
		//	c = &colorful.Color{} //is this black?
		//} else {
		//	fmt.Printf("%+v", *c)
		//}
		//c.R = c.R * brightness
		//c.G = c.G * brightness
		//c.B = c.B * brightness
		//c.RGB255()
		r, g, b := normalizeBrightness(c)
		data[3*i+3] = byte(r)   //Red
		data[3*i+3+1] = byte(g) //Green
		data[3*i+3+2] = byte(b) //Blue
	}

	//addr := libusb.EndpointAddress(byte(3))
	//start := time.Now()
	_, err := endpoint.Write(data)
	//_, err := deviceHandle.BulkTransfer(addr, data, len(data), 20)
	if err != nil {
		log.Fatalf("Error bulk transferring: %v", err)
	}
	//log.Printf("Usb transfer took: %v\n", time.Since(start))

}

func showDevices() {
	devices, err := ctx.GetDeviceList()
	if err != nil {
		log.Fatalf("Error getting device list: %v", err)
	}

	for _, usbDevice := range devices {
		deviceAddress, _ := usbDevice.GetDeviceAddress()
		deviceSpeed, _ := usbDevice.GetDeviceSpeed()
		busNumber, _ := usbDevice.GetBusNumber()
		usbDeviceDescriptor, _ := usbDevice.GetDeviceDescriptor()
		log.Printf("Device address %v is on bus number %v\n=> %v\n",
			deviceAddress,
			busNumber,
			deviceSpeed,
		)
		log.Printf("=> Vendor: %v \tProduct: %v\n=> Class: %v\n",
			usbDeviceDescriptor.VendorID,
			usbDeviceDescriptor.ProductID,
			usbDeviceDescriptor.DeviceClass,
		)
		log.Printf("=> USB: %v\tMax Packet 0: %v\tSN Index: %v\n",
			usbDeviceDescriptor.USBSpecification,
			usbDeviceDescriptor.MaxPacketSize0,
			usbDeviceDescriptor.SerialNumberIndex,
		)
	}
}

func ShowVersion() {
	version := libusb.GetVersion()
	log.Printf(
		"Using libusb version %d.%d.%d (%d)\n",
		version.Major,
		version.Minor,
		version.Micro,
		version.Nano,
	)
}

func showInfo(ctx *libusb.Context, name string, vendorID, productID uint16) {
	log.Printf("Let's open the %s using the Vendor and Product IDs\n", name)
	usbDevice, usbDeviceHandle, err := ctx.OpenDeviceWithVendorProduct(vendorID, productID)
	if err != nil {
		log.Fatalf("Could not open device with error: %v\n", err)
	}
	usbDeviceDescriptor, err := usbDevice.GetDeviceDescriptor()
	if err != nil {
		log.Fatalf("=> Failed opening the %s: %v\n", name, err)
		return
	}
	defer usbDeviceHandle.Close()
	serialnum, _ := usbDeviceHandle.GetStringDescriptorASCII(
		usbDeviceDescriptor.SerialNumberIndex,
	)
	manufacturer, _ := usbDeviceHandle.GetStringDescriptorASCII(
		usbDeviceDescriptor.ManufacturerIndex)
	product, _ := usbDeviceHandle.GetStringDescriptorASCII(
		usbDeviceDescriptor.ProductIndex)
	log.Printf("Found %v %v S/N %s using Vendor ID %v and Product ID %v\n",
		manufacturer,
		product,
		serialnum,
		vendorID,
		productID,
	)
	configDescriptor, err := usbDevice.GetActiveConfigDescriptor()
	if err != nil {
		log.Fatalf("Failed getting the active config: %v", err)
	}
	log.Printf("=> Max Power = %d mA\n",
		configDescriptor.MaxPowerMilliAmperes)
	var singularPlural string
	if configDescriptor.NumInterfaces == 1 {
		singularPlural = "interface"
	} else {
		singularPlural = "interfaces"
	}
	log.Printf("=> Found %d %s\n",
		configDescriptor.NumInterfaces, singularPlural)

	for i, supportedInterface := range configDescriptor.SupportedInterfaces {

		log.Printf("=> %d interface has %d alternate settings.\n", i,
			supportedInterface.NumAltSettings)
		descriptor := supportedInterface.InterfaceDescriptors[0]
		log.Printf("=> %d interface descriptor has a length of %d.\n", i, descriptor.Length)
		log.Printf("=> %d interface descriptor is interface number %d.\n", i, descriptor.InterfaceNumber)
		log.Printf("=> %d interface descriptor has %d endpoint(s).\n", i, descriptor.NumEndpoints)
		log.Printf(
			"   => USB-IF class %d, subclass %d, protocol %d.\n",
			descriptor.InterfaceClass, descriptor.InterfaceSubClass, descriptor.InterfaceProtocol,
		)
		for j, endpoint := range descriptor.EndpointDescriptors {
			log.Printf(
				"   => Endpoint index %d on Interface %d has the following properties:\n",
				j, descriptor.InterfaceNumber)
			log.Printf("     => Address: %d (b%08b)\n", endpoint.EndpointAddress, endpoint.EndpointAddress)
			log.Printf("       => Endpoint #: %d\n", endpoint.Number())
			log.Printf("       => Direction: %s (%d)\n", endpoint.Direction(), endpoint.Direction())
			log.Printf("     => Attributes: %d (b%08b) \n", endpoint.Attributes, endpoint.Attributes)
			log.Printf("       => Transfer Type: %s (%d) \n", endpoint.TransferType(), endpoint.TransferType())
			log.Printf("     => Max packet size: %d\n", endpoint.MaxPacketSize)
		}
		log.Println()
	}
}
