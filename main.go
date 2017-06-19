package main

import (
	"fmt"
	"github.com/drichelson/usb-test/usb"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/ojrac/opensimplex-go"
	"github.com/yvasiyarov/go-metrics"
	"log"
	"math"
	"time"
)

const (
	ColumnCount        = 64
	RowCount           = 20
	expectedPixelCount = 1200
)

var (
	pixels   []*BallPixel
	rows     = make([][]*BallPixel, RowCount)
	cols     = make([][]*BallPixel, ColumnCount)
	renderCh = make(chan []colorful.Color, 1)
)

type Animation interface {
	frame(time float64, frameCount int)
}

type BallPixel struct {
	col      int
	row      int
	x        float64
	y        float64
	z        float64
	lat      float64
	lon      float64
	color    *colorful.Color
	disabled bool
}

func init() {
	loadMapping()
	//populate colors:
	for i, p := range pixels {
		if p == nil {
			pixels[i] = &BallPixel{disabled: true}
		}
		pixels[i].color = &colorful.Color{}
	}

	//for pixelCount := len(colors); pixelCount < expectedPixelCount; pixelCount++ {
	//	colors = append(colors, &colorful.Color{})
	//}

	//populate rows and columns
	for _, p := range pixels {
		if !p.disabled {
			if rows[p.row] == nil {
				rows[p.row] = make([]*BallPixel, 0)
			}
			rows[p.row] = append(rows[p.row], p)

			if cols[p.col] == nil {
				cols[p.col] = make([]*BallPixel, 0)
			}
			cols[p.col] = append(cols[p.col], p)
		}
	}

	fmt.Printf("pixel count: %d\n", len(pixels))
	fmt.Printf("row count: %d\n", len(rows))
	fmt.Printf("col count: %d\n", len(cols))
}

//func (p *BallPixel) setColor(color *colorful.Color) {
//	p.color = color
//}

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Lshortfile)
	usb.Initialize()

	go func() {
		for {
			usb.Render(<-renderCh, 0.3)
		}
	}()

	//test()
	var a Animation
	colors, err := colorful.WarmPalette(3)
	if err != nil {
		fmt.Printf("%v", err)
	}
	a = &OpenSimplexAnimation{
		noise:  opensimplex.NewWithSeed(time.Now().UnixNano()),
		colors: colors,
		histo:  metrics.GetOrRegisterHistogram("histo", metrics.DefaultRegistry, metrics.NewExpDecaySample(expectedPixelCount*10000, 1.0)),
	}
	startTime := time.Now()
	checkPointTime := startTime
	frameCount := 0

	for {
		timeSinceStartSeconds := time.Since(startTime).Seconds()
		a.frame(timeSinceStartSeconds, frameCount)
		//w := float64(time.Since(startTime).Nanoseconds())
		//fmt.Printf("%f\n", w)
		//w += 0.005

		render()
		reset()
		//time.Sleep(100 * time.Millisecond)
		frameCount++
		if frameCount%1000 == 0 {
			newCheckPointTime := time.Now()
			fmt.Printf("Avg FPS for past 1000 frames: %v\n", 1000.0/time.Since(checkPointTime).Seconds())
			checkPointTime = newCheckPointTime
		}
		//fmt.Printf("histo: min: %v median: %v, max: %v\n", histo.Min(), histo.Percentile(0.5), histo.Max())
	}
}

type OpenSimplexAnimation struct {
	noise  *opensimplex.Noise
	colors []colorful.Color
	histo  metrics.Histogram
	min    float64
	max    float64
}

func (a *OpenSimplexAnimation) frame(time float64, frameCount int) {
	for _, p := range pixels {
		if !p.disabled {
			noiseVal := a.noise.Eval4(p.x, p.y, p.z, time/2.0)
			a.min = math.Min(a.min, noiseVal)
			a.max = math.Max(a.max, noiseVal)

			noiseValNormalized := a.normalizeNoiseValue(noiseVal)
			a.histo.Update(int64(noiseValNormalized * 1000.0))
			h := 20.0 * noiseValNormalized
			c := colorful.Hsv(h, 1.0, noiseValNormalized)
			p.color = &c
			//fmt.Printf("%v\n", noiseVal)

		}

	}
	if frameCount%1000 == 0 {
		go func() {
			snapshot := a.histo.Snapshot()
			fmt.Printf("Normalized histo: min: %.3f P10: %.3f P20: %.3f P30: %.3f P40: %.3f P50: %.3f P60: %.3f P70: %.3f P80: %.3f P90: %.3f max: %.3f\n",
				float64(snapshot.Min())/1000.0,
				snapshot.Percentile(0.1)/1000.0,
				snapshot.Percentile(0.2)/1000.0,
				snapshot.Percentile(0.3)/1000.0,
				snapshot.Percentile(0.4)/1000.0,
				snapshot.Percentile(0.5)/1000.0,
				snapshot.Percentile(0.6)/1000.0,
				snapshot.Percentile(0.7)/1000.0,
				snapshot.Percentile(0.8)/1000.0,
				snapshot.Percentile(0.9)/1000.0,
				float64(snapshot.Max())/1000.0)
		}()
	}
}

// takes an arbitrary float and normalizes it to a range between 0-1.0
// based on the animation's min and max. This should give us a smooth adjustment based on the past
// ~100 frames' worth of noise values.
func (a *OpenSimplexAnimation) normalizeNoiseValue(noiseVal float64) float64 {
	noiseVal = noiseVal * 10.0 //adjust for the fact that the noise is clustered around the middle
	noiseVal = math.Max(a.min, noiseVal)
	noiseVal = math.Min(a.max, noiseVal)
	histoDiff := a.max - a.min
	noiseValDistFromMin := noiseVal - a.min
	return noiseValDistFromMin / histoDiff
}

//Teensy:
// descriptor: &{Length:18 DescriptorType:Device descriptor. USBSpecification:0x0200 (2.00) DeviceClass:Communications class. DeviceSubClass:0 DeviceProtocol:0 MaxPacketSize0:64 VendorID:5824 ProductID:1155 DeviceReleaseNumber:0x0100 (1.00) ManufacturerIndex:1 ProductIndex:2 SerialNumberIndex:3 NumConfigurations:1}

func reset() {
	for i := range pixels {
		pixels[i].color = &colorful.Color{}
	}
}

func render() {
	colors := make([]colorful.Color, len(pixels))
	for i, p := range pixels {
		colors[i] = *p.color
	}
	renderCh <- colors
}

func test() {
	for _, c := range []colorful.Color{{R: 1.0}, {G: 1.0}, {B: 1.0}} {
		for i, row := range rows {
			fmt.Printf("row: %d\n", i)
			reset()
			for _, pixel := range row {
				pixel.color = &c
			}
			render()
			time.Sleep(50 * time.Millisecond)
		}
	}
	for _, c := range []colorful.Color{{R: 1.0}, {G: 1.0}, {B: 1.0}} {
		for i, col := range cols {
			fmt.Printf("column: %d\n", i)
			reset()
			for _, pixel := range col {
				pixel.color = &c
			}
			render()
			time.Sleep(50 * time.Millisecond)
		}
	}
	reset()
	render()
	time.Sleep(50 * time.Millisecond)
}

func loadMapping() {
	pixels = make([]*BallPixel, expectedPixelCount)
	pixels[0] = &BallPixel{col: 0, row: 19, x: -3.2997538248697946, y: 3.762969970703125, z: 0.0, lat: 48.75, lon: 0.0}
	pixels[1] = &BallPixel{col: 0, row: 18, x: -3.762969970703126, y: 3.2997538248697915, z: 0.0, lat: 41.25, lon: 0.0}
	pixels[2] = &BallPixel{col: 0, row: 17, x: -4.160919189453128, y: 2.779083251953125, z: 0.0, lat: 33.75, lon: 0.0}
	pixels[3] = &BallPixel{col: 0, row: 16, x: -4.487091064453128, y: 2.2100728352864585, z: 0.0, lat: 26.25, lon: 0.0}
	pixels[4] = &BallPixel{col: 0, row: 15, x: -4.736277262369794, y: 1.6031392415364585, z: 0.0, lat: 18.75, lon: 0.0}
	pixels[5] = &BallPixel{col: 0, row: 14, x: -4.904571533203125, y: 0.970001220703125, z: 0.0, lat: 11.25, lon: 0.0}
	pixels[6] = &BallPixel{col: 0, row: 13, x: -4.989369710286461, y: 0.32367960611979163, z: 0.0, lat: 3.75, lon: 0.0}
	pixels[7] = &BallPixel{col: 0, row: 12, x: -4.989369710286458, y: -0.32367960611979163, z: 0.0, lat: -3.75, lon: 0.0}
	pixels[8] = &BallPixel{col: 0, row: 11, x: -4.904571533203126, y: -0.970001220703125, z: 0.0, lat: -11.25, lon: 0.0}
	pixels[9] = &BallPixel{col: 0, row: 10, x: -4.736277262369791, y: -1.6031392415364585, z: 0.0, lat: -18.75, lon: 0.0}
	pixels[10] = &BallPixel{col: 0, row: 9, x: -4.487091064453125, y: -2.2100728352864585, z: 0.0, lat: -26.25, lon: 0.0}
	pixels[11] = &BallPixel{col: 0, row: 8, x: -4.160919189453125, y: -2.779083251953125, z: 0.0, lat: -33.75, lon: 0.0}
	pixels[12] = &BallPixel{col: 0, row: 7, x: -3.7629699707031254, y: -3.2997538248697915, z: 0.0, lat: -41.25, lon: 0.0}
	pixels[13] = &BallPixel{col: 0, row: 6, x: -3.299753824869792, y: -3.762969970703125, z: 0.0, lat: -48.75, lon: 0.0}
	pixels[14] = &BallPixel{col: 0, row: 5, x: -2.779083251953125, y: -4.160919189453125, z: 0.0, lat: -56.25, lon: 0.0}
	pixels[15] = &BallPixel{col: 0, row: 4, x: -2.210072835286458, y: -4.487091064453125, z: 0.0, lat: -63.75, lon: 0.0}
	pixels[16] = &BallPixel{col: 0, row: 3, x: -1.603139241536459, y: -4.736277262369791, z: 0.0, lat: -71.25, lon: 0.0}
	pixels[17] = &BallPixel{col: 0, row: 2, x: -0.970001220703125, y: -4.904571533203126, z: 0.0, lat: -78.75, lon: 0.0}
	pixels[18] = &BallPixel{col: 0, row: 1, x: -0.32367960611979135, y: -4.989369710286458, z: 0.0, lat: -86.25, lon: 0.0}
	pixels[19] = &BallPixel{col: 0, row: 0, x: 0.3236796061197924, y: -4.989369710286458, z: -0.0, lat: -93.75, lon: 0.0}
	pixels[20] = &BallPixel{col: 32, row: 0, x: -0.3236796061197924, y: -4.989369710286458, z: -0.0, lat: -93.75, lon: -180.0}
	pixels[21] = &BallPixel{col: 32, row: 1, x: 0.32367960611979135, y: -4.989369710286458, z: 0.0, lat: -86.25, lon: -180.0}
	pixels[22] = &BallPixel{col: 32, row: 2, x: 0.970001220703125, y: -4.904571533203126, z: 0.0, lat: -78.75, lon: -180.0}
	pixels[23] = &BallPixel{col: 32, row: 3, x: 1.603139241536459, y: -4.736277262369791, z: 0.0, lat: -71.25, lon: -180.0}
	pixels[24] = &BallPixel{col: 32, row: 4, x: 2.210072835286458, y: -4.487091064453125, z: 0.0, lat: -63.75, lon: -180.0}
	pixels[25] = &BallPixel{col: 32, row: 5, x: 2.779083251953125, y: -4.160919189453125, z: 0.0, lat: -56.25, lon: -180.0}
	pixels[26] = &BallPixel{col: 32, row: 6, x: 3.299753824869792, y: -3.762969970703125, z: 0.0, lat: -48.75, lon: -180.0}
	pixels[27] = &BallPixel{col: 32, row: 7, x: 3.7629699707031254, y: -3.2997538248697915, z: 0.0, lat: -41.25, lon: -180.0}
	pixels[28] = &BallPixel{col: 32, row: 8, x: 4.160919189453125, y: -2.779083251953125, z: 0.0, lat: -33.75, lon: -180.0}
	pixels[29] = &BallPixel{col: 32, row: 9, x: 4.487091064453125, y: -2.2100728352864585, z: 0.0, lat: -26.25, lon: -180.0}
	pixels[30] = &BallPixel{col: 32, row: 10, x: 4.736277262369791, y: -1.6031392415364585, z: 0.0, lat: -18.75, lon: -180.0}
	pixels[31] = &BallPixel{col: 32, row: 11, x: 4.904571533203126, y: -0.970001220703125, z: 0.0, lat: -11.25, lon: -180.0}
	pixels[32] = &BallPixel{col: 32, row: 12, x: 4.989369710286458, y: -0.32367960611979163, z: 0.0, lat: -3.75, lon: -180.0}
	pixels[33] = &BallPixel{col: 32, row: 13, x: 4.989369710286461, y: 0.32367960611979163, z: 0.0, lat: 3.75, lon: -180.0}
	pixels[34] = &BallPixel{col: 32, row: 14, x: 4.904571533203125, y: 0.970001220703125, z: 0.0, lat: 11.25, lon: -180.0}
	pixels[35] = &BallPixel{col: 32, row: 15, x: 4.736277262369794, y: 1.6031392415364585, z: 0.0, lat: 18.75, lon: -180.0}
	pixels[36] = &BallPixel{col: 32, row: 16, x: 4.487091064453128, y: 2.2100728352864585, z: 0.0, lat: 26.25, lon: -180.0}
	pixels[37] = &BallPixel{col: 32, row: 17, x: 4.160919189453128, y: 2.779083251953125, z: 0.0, lat: 33.75, lon: -180.0}
	pixels[38] = &BallPixel{col: 32, row: 18, x: 3.762969970703126, y: 3.2997538248697915, z: 0.0, lat: 41.25, lon: -180.0}
	pixels[39] = &BallPixel{col: 32, row: 19, x: 3.2997538248697946, y: 3.762969970703125, z: 0.0, lat: 48.75, lon: -180.0}
	pixels[40] = &BallPixel{col: 16, row: 18, x: -0.0, y: 3.2997538248697915, z: 3.762969970703126, lat: 41.25, lon: 90.0}
	pixels[41] = &BallPixel{col: 16, row: 17, x: -0.0, y: 2.779083251953125, z: 4.160919189453128, lat: 33.75, lon: 90.0}
	pixels[42] = &BallPixel{col: 16, row: 16, x: -0.0, y: 2.2100728352864585, z: 4.487091064453128, lat: 26.25, lon: 90.0}
	pixels[43] = &BallPixel{col: 16, row: 15, x: -0.0, y: 1.6031392415364585, z: 4.736277262369794, lat: 18.75, lon: 90.0}
	pixels[44] = &BallPixel{col: 16, row: 14, x: -0.0, y: 0.970001220703125, z: 4.904571533203125, lat: 11.25, lon: 90.0}
	pixels[45] = &BallPixel{col: 16, row: 13, x: -0.0, y: 0.32367960611979163, z: 4.989369710286461, lat: 3.75, lon: 90.0}
	pixels[46] = &BallPixel{col: 16, row: 12, x: -0.0, y: -0.32367960611979163, z: 4.989369710286458, lat: -3.75, lon: 90.0}
	pixels[47] = &BallPixel{col: 16, row: 11, x: -0.0, y: -0.970001220703125, z: 4.904571533203126, lat: -11.25, lon: 90.0}
	pixels[48] = &BallPixel{col: 16, row: 10, x: -0.0, y: -1.6031392415364585, z: 4.736277262369791, lat: -18.75, lon: 90.0}
	pixels[49] = &BallPixel{col: 16, row: 9, x: -0.0, y: -2.2100728352864585, z: 4.487091064453125, lat: -26.25, lon: 90.0}
	pixels[50] = &BallPixel{col: 16, row: 8, x: -0.0, y: -2.779083251953125, z: 4.160919189453125, lat: -33.75, lon: 90.0}
	pixels[51] = &BallPixel{col: 16, row: 7, x: -0.0, y: -3.2997538248697915, z: 3.7629699707031254, lat: -41.25, lon: 90.0}
	pixels[52] = &BallPixel{col: 16, row: 6, x: -0.0, y: -3.762969970703125, z: 3.299753824869792, lat: -48.75, lon: 90.0}
	pixels[53] = &BallPixel{col: 16, row: 5, x: -0.0, y: -4.160919189453125, z: 2.779083251953125, lat: -56.25, lon: 90.0}
	pixels[54] = &BallPixel{col: 16, row: 4, x: -0.0, y: -4.487091064453125, z: 2.210072835286458, lat: -63.75, lon: 90.0}
	pixels[55] = &BallPixel{col: 16, row: 3, x: -0.0, y: -4.736277262369791, z: 1.603139241536459, lat: -71.25, lon: 90.0}
	pixels[56] = &BallPixel{col: 16, row: 2, x: -0.0, y: -4.904571533203126, z: 0.970001220703125, lat: -78.75, lon: 90.0}
	pixels[57] = &BallPixel{col: 16, row: 1, x: -0.0, y: -4.989369710286458, z: 0.32367960611979135, lat: -86.25, lon: 90.0}
	pixels[58] = &BallPixel{col: 16, row: 0, x: 0.0, y: -4.989369710286458, z: -0.3236796061197924, lat: -93.75, lon: 90.0}
	pixels[59] = &BallPixel{col: 48, row: 0, x: 0.0, y: -4.989369710286458, z: 0.3236796061197924, lat: -93.75, lon: -90.0}
	pixels[60] = &BallPixel{col: 48, row: 1, x: -0.0, y: -4.989369710286458, z: -0.32367960611979135, lat: -86.25, lon: -90.0}
	pixels[61] = &BallPixel{col: 48, row: 2, x: -0.0, y: -4.904571533203126, z: -0.970001220703125, lat: -78.75, lon: -90.0}
	pixels[62] = &BallPixel{col: 48, row: 3, x: -0.0, y: -4.736277262369791, z: -1.603139241536459, lat: -71.25, lon: -90.0}
	pixels[63] = &BallPixel{col: 48, row: 4, x: -0.0, y: -4.487091064453125, z: -2.210072835286458, lat: -63.75, lon: -90.0}
	pixels[64] = &BallPixel{col: 48, row: 5, x: -0.0, y: -4.160919189453125, z: -2.779083251953125, lat: -56.25, lon: -90.0}
	pixels[65] = &BallPixel{col: 48, row: 6, x: -0.0, y: -3.762969970703125, z: -3.299753824869792, lat: -48.75, lon: -90.0}
	pixels[66] = &BallPixel{col: 48, row: 7, x: -0.0, y: -3.2997538248697915, z: -3.7629699707031254, lat: -41.25, lon: -90.0}
	pixels[67] = &BallPixel{col: 48, row: 8, x: -0.0, y: -2.779083251953125, z: -4.160919189453125, lat: -33.75, lon: -90.0}
	pixels[68] = &BallPixel{col: 48, row: 9, x: -0.0, y: -2.2100728352864585, z: -4.487091064453125, lat: -26.25, lon: -90.0}
	pixels[69] = &BallPixel{col: 48, row: 10, x: -0.0, y: -1.6031392415364585, z: -4.736277262369791, lat: -18.75, lon: -90.0}
	pixels[70] = &BallPixel{col: 48, row: 11, x: -0.0, y: -0.970001220703125, z: -4.904571533203126, lat: -11.25, lon: -90.0}
	pixels[71] = &BallPixel{col: 48, row: 12, x: -0.0, y: -0.32367960611979163, z: -4.989369710286458, lat: -3.75, lon: -90.0}
	pixels[72] = &BallPixel{col: 48, row: 13, x: -0.0, y: 0.32367960611979163, z: -4.989369710286461, lat: 3.75, lon: -90.0}
	pixels[73] = &BallPixel{col: 48, row: 14, x: -0.0, y: 0.970001220703125, z: -4.904571533203125, lat: 11.25, lon: -90.0}
	pixels[74] = &BallPixel{col: 48, row: 15, x: -0.0, y: 1.6031392415364585, z: -4.736277262369794, lat: 18.75, lon: -90.0}
	pixels[75] = &BallPixel{col: 48, row: 16, x: -0.0, y: 2.2100728352864585, z: -4.487091064453128, lat: 26.25, lon: -90.0}
	pixels[76] = &BallPixel{col: 48, row: 17, x: -0.0, y: 2.779083251953125, z: -4.160919189453128, lat: 33.75, lon: -90.0}
	pixels[77] = &BallPixel{col: 48, row: 18, x: -0.0, y: 3.2997538248697915, z: -3.762969970703126, lat: 41.25, lon: -90.0}
	pixels[78] = &BallPixel{col: 48, row: 19, x: -0.0, y: 3.762969970703125, z: -3.2997538248697946, lat: 48.75, lon: -90.0}
	pixels[79] = &BallPixel{col: 56, row: 19, x: -2.3356070041656514, y: 3.762969970703125, z: -2.3356070041656514, lat: 48.75, lon: -45.0}
	pixels[80] = &BallPixel{col: 56, row: 18, x: -2.663477182388306, y: 3.2997538248697915, z: -2.663477182388306, lat: 41.25, lon: -45.0}
	pixels[81] = &BallPixel{col: 56, row: 17, x: -2.945150613784792, y: 2.779083251953125, z: -2.945150613784792, lat: 33.75, lon: -45.0}
	pixels[82] = &BallPixel{col: 56, row: 16, x: -3.176019144058229, y: 2.2100728352864585, z: -3.176019144058229, lat: 26.25, lon: -45.0}
	pixels[83] = &BallPixel{col: 56, row: 15, x: -3.3523962497711195, y: 1.6031392415364585, z: -3.3523962497711195, lat: 18.75, lon: -45.0}
	pixels[84] = &BallPixel{col: 56, row: 14, x: -3.4715170383453366, y: 0.970001220703125, z: -3.4715170383453366, lat: 11.25, lon: -45.0}
	pixels[85] = &BallPixel{col: 56, row: 13, x: -3.531538248062135, y: 0.32367960611979163, z: -3.531538248062135, lat: 3.75, lon: -45.0}
	pixels[86] = &BallPixel{col: 56, row: 12, x: -3.5315382480621333, y: -0.32367960611979163, z: -3.5315382480621333, lat: -3.75, lon: -45.0}
	pixels[87] = &BallPixel{col: 56, row: 11, x: -3.4715170383453375, y: -0.970001220703125, z: -3.4715170383453375, lat: -11.25, lon: -45.0}
	pixels[88] = &BallPixel{col: 56, row: 10, x: -3.3523962497711177, y: -1.6031392415364585, z: -3.3523962497711177, lat: -18.75, lon: -45.0}
	pixels[89] = &BallPixel{col: 56, row: 9, x: -3.1760191440582273, y: -2.2100728352864585, z: -3.1760191440582273, lat: -26.25, lon: -45.0}
	pixels[90] = &BallPixel{col: 56, row: 8, x: -2.94515061378479, y: -2.779083251953125, z: -2.94515061378479, lat: -33.75, lon: -45.0}
	pixels[91] = &BallPixel{col: 56, row: 7, x: -2.6634771823883057, y: -3.2997538248697915, z: -2.6634771823883057, lat: -41.25, lon: -45.0}
	pixels[92] = &BallPixel{col: 56, row: 6, x: -2.3356070041656496, y: -3.762969970703125, z: -2.3356070041656496, lat: -48.75, lon: -45.0}
	pixels[93] = &BallPixel{col: 56, row: 5, x: -1.967069864273071, y: -4.160919189453125, z: -1.967069864273071, lat: -56.25, lon: -45.0}
	pixels[94] = &BallPixel{col: 56, row: 4, x: -1.564317178726196, y: -4.487091064453125, z: -1.564317178726196, lat: -63.75, lon: -45.0}
	pixels[95] = &BallPixel{col: 56, row: 3, x: -1.1347219944000249, y: -4.736277262369791, z: -1.1347219944000249, lat: -71.25, lon: -45.0}
	pixels[96] = &BallPixel{col: 56, row: 2, x: -0.6865789890289307, y: -4.904571533203126, z: -0.6865789890289307, lat: -78.75, lon: -45.0}
	pixels[97] = &BallPixel{col: 56, row: 1, x: -0.2291044712066648, y: -4.989369710286458, z: -0.2291044712066648, lat: -86.25, lon: -45.0}
	pixels[98] = &BallPixel{col: 24, row: 1, x: 0.2291044712066648, y: -4.989369710286458, z: 0.2291044712066648, lat: -86.25, lon: 135.0}
	pixels[99] = &BallPixel{col: 24, row: 2, x: 0.6865789890289307, y: -4.904571533203126, z: 0.6865789890289307, lat: -78.75, lon: 135.0}
	pixels[100] = &BallPixel{col: 24, row: 3, x: 1.1347219944000249, y: -4.736277262369791, z: 1.1347219944000249, lat: -71.25, lon: 135.0}
	pixels[101] = &BallPixel{col: 24, row: 4, x: 1.564317178726196, y: -4.487091064453125, z: 1.564317178726196, lat: -63.75, lon: 135.0}
	pixels[102] = &BallPixel{col: 24, row: 5, x: 1.967069864273071, y: -4.160919189453125, z: 1.967069864273071, lat: -56.25, lon: 135.0}
	pixels[103] = &BallPixel{col: 24, row: 6, x: 2.3356070041656496, y: -3.762969970703125, z: 2.3356070041656496, lat: -48.75, lon: 135.0}
	pixels[104] = &BallPixel{col: 24, row: 7, x: 2.6634771823883057, y: -3.2997538248697915, z: 2.6634771823883057, lat: -41.25, lon: 135.0}
	pixels[105] = &BallPixel{col: 24, row: 8, x: 2.94515061378479, y: -2.779083251953125, z: 2.94515061378479, lat: -33.75, lon: 135.0}
	pixels[106] = &BallPixel{col: 24, row: 9, x: 3.1760191440582273, y: -2.2100728352864585, z: 3.1760191440582273, lat: -26.25, lon: 135.0}
	pixels[107] = &BallPixel{col: 24, row: 10, x: 3.3523962497711177, y: -1.6031392415364585, z: 3.3523962497711177, lat: -18.75, lon: 135.0}
	pixels[108] = &BallPixel{col: 24, row: 11, x: 3.4715170383453375, y: -0.970001220703125, z: 3.4715170383453375, lat: -11.25, lon: 135.0}
	pixels[109] = &BallPixel{col: 24, row: 12, x: 3.5315382480621333, y: -0.32367960611979163, z: 3.5315382480621333, lat: -3.75, lon: 135.0}
	pixels[110] = &BallPixel{col: 24, row: 13, x: 3.531538248062135, y: 0.32367960611979163, z: 3.531538248062135, lat: 3.75, lon: 135.0}
	pixels[111] = &BallPixel{col: 24, row: 14, x: 3.4715170383453366, y: 0.970001220703125, z: 3.4715170383453366, lat: 11.25, lon: 135.0}
	pixels[112] = &BallPixel{col: 24, row: 15, x: 3.3523962497711195, y: 1.6031392415364585, z: 3.3523962497711195, lat: 18.75, lon: 135.0}
	pixels[113] = &BallPixel{col: 24, row: 16, x: 3.176019144058229, y: 2.2100728352864585, z: 3.176019144058229, lat: 26.25, lon: 135.0}
	pixels[114] = &BallPixel{col: 24, row: 17, x: 2.945150613784792, y: 2.779083251953125, z: 2.945150613784792, lat: 33.75, lon: 135.0}
	pixels[115] = &BallPixel{col: 24, row: 18, x: 2.663477182388306, y: 3.2997538248697915, z: 2.663477182388306, lat: 41.25, lon: 135.0}
	pixels[116] = &BallPixel{col: 24, row: 19, x: 2.3356070041656514, y: 3.762969970703125, z: 2.3356070041656514, lat: 48.75, lon: 135.0}
	pixels[150] = &BallPixel{col: 47, row: 17, x: 0.4042207662132576, y: 2.779083251953125, z: -4.14102282637032, lat: 33.75, lon: -95.625}
	pixels[151] = &BallPixel{col: 47, row: 16, x: 0.4359073814121083, y: 2.2100728352864585, z: -4.465635037806353, lat: 26.25, lon: -95.625}
	pixels[152] = &BallPixel{col: 47, row: 15, x: 0.4601150699697132, y: 1.6031392415364585, z: -4.7136296963435615, lat: 18.75, lon: -95.625}
	pixels[153] = &BallPixel{col: 47, row: 14, x: 0.47646435146452915, y: 0.970001220703125, z: -4.881119230587501, lat: 11.25, lon: -95.625}
	pixels[154] = &BallPixel{col: 47, row: 13, x: 0.4847022389488613, y: 0.32367960611979163, z: -4.965511926275215, lat: 3.75, lon: -95.625}
	pixels[155] = &BallPixel{col: 47, row: 12, x: 0.48470223894886105, y: -0.32367960611979163, z: -4.965511926275212, lat: -3.75, lon: -95.625}
	pixels[156] = &BallPixel{col: 47, row: 11, x: 0.4764643514645292, y: -0.970001220703125, z: -4.881119230587502, lat: -11.25, lon: -95.625}
	pixels[157] = &BallPixel{col: 47, row: 10, x: 0.4601150699697129, y: -1.6031392415364585, z: -4.713629696343559, lat: -18.75, lon: -95.625}
	pixels[158] = &BallPixel{col: 47, row: 9, x: 0.435907381412108, y: -2.2100728352864585, z: -4.465635037806351, lat: -26.25, lon: -95.625}
	pixels[159] = &BallPixel{col: 47, row: 8, x: 0.4042207662132573, y: -2.779083251953125, z: -4.1410228263703175, lat: -33.75, lon: -95.625}
	pixels[160] = &BallPixel{col: 47, row: 7, x: 0.3655611982685518, y: -3.2997538248697915, z: -3.744976490561385, lat: -41.25, lon: -95.625}
	pixels[161] = &BallPixel{col: 47, row: 6, x: 0.32056114494723, y: -3.762969970703125, z: -3.2839753160369587, lat: -48.75, lon: -95.625}
	pixels[162] = &BallPixel{col: 47, row: 5, x: 0.269979567092378, y: -4.160919189453125, z: -2.765794445585925, lat: -56.25, lon: -95.625}
	pixels[163] = &BallPixel{col: 45, row: 5, x: 0.8041694404673759, y: -4.160919189453125, z: -2.6601709628594117, lat: -56.25, lon: -106.875}
	pixels[164] = &BallPixel{col: 45, row: 6, x: 0.9548332836595361, y: -3.762969970703125, z: -3.1585629193849565, lat: -48.75, lon: -106.875}
	pixels[165] = &BallPixel{col: 45, row: 7, x: 1.088871826243121, y: -3.2997538248697915, z: -3.6019588269409737, lat: -41.25, lon: -106.875}
	pixels[166] = &BallPixel{col: 45, row: 8, x: 1.2040244041127148, y: -2.779083251953125, z: -3.98288046923699, lat: -33.75, lon: -106.875}
	pixels[167] = &BallPixel{col: 45, row: 9, x: 1.2984071305138054, y: -2.2100728352864585, z: -4.2950959993642766, lat: -26.25, lon: -106.875}
	pixels[168] = &BallPixel{col: 45, row: 10, x: 1.3705128960427821, y: -1.6031392415364585, z: -4.533619939795852, lat: -18.75, lon: -106.875}
	pixels[169] = &BallPixel{col: 45, row: 11, x: 1.419211368646938, y: -0.970001220703125, z: -4.6947131823864785, lat: -11.25, lon: -106.875}
	pixels[170] = &BallPixel{col: 45, row: 12, x: 1.4437489936244667, y: -0.32367960611979163, z: -4.775882988372664, lat: -3.75, lon: -106.875}
	pixels[171] = &BallPixel{col: 45, row: 13, x: 1.4437489936244676, y: 0.32367960611979163, z: -4.775882988372667, lat: 3.75, lon: -106.875}
	pixels[172] = &BallPixel{col: 45, row: 14, x: 1.4192113686469379, y: 0.970001220703125, z: -4.694713182386478, lat: 11.25, lon: -106.875}
	pixels[173] = &BallPixel{col: 45, row: 15, x: 1.370512896042783, y: 1.6031392415364585, z: -4.533619939795853, lat: 18.75, lon: -106.875}
	pixels[174] = &BallPixel{col: 45, row: 16, x: 1.2984071305138063, y: 2.2100728352864585, z: -4.295095999364279, lat: 26.25, lon: -106.875}
	pixels[175] = &BallPixel{col: 45, row: 17, x: 1.2040244041127157, y: 2.779083251953125, z: -3.982880469236992, lat: 33.75, lon: -106.875}
	pixels[176] = &BallPixel{col: 46, row: 19, x: 0.6401530476287013, y: 3.762969970703125, z: -3.236775735206905, lat: 48.75, lon: -101.25}
	pixels[177] = &BallPixel{col: 46, row: 18, x: 0.730017093010247, y: 3.2997538248697915, z: -3.6911510797217497, lat: 41.25, lon: -101.25}
	pixels[178] = &BallPixel{col: 46, row: 17, x: 0.8072193386033183, y: 2.779083251953125, z: -4.081505161710086, lat: 33.75, lon: -101.25}
	pixels[179] = &BallPixel{col: 46, row: 16, x: 0.8704967619851237, y: 2.2100728352864585, z: -4.401451820321384, lat: 26.25, lon: -101.25}
	pixels[180] = &BallPixel{col: 46, row: 15, x: 0.918838945217431, y: 1.6031392415364585, z: -4.6458821268752235, lat: 18.75, lon: -101.25}
	pixels[181] = &BallPixel{col: 46, row: 14, x: 0.9514880748465657, y: 0.970001220703125, z: -4.810964384861291, lat: 11.25, lon: -101.25}
	pixels[182] = &BallPixel{col: 46, row: 13, x: 0.9679389419034128, y: 0.32367960611979163, z: -4.894144129939379, lat: 3.75, lon: -101.25}
	pixels[183] = &BallPixel{col: 46, row: 12, x: 0.9679389419034122, y: -0.32367960611979163, z: -4.894144129939376, lat: -3.75, lon: -101.25}
	pixels[184] = &BallPixel{col: 46, row: 11, x: 0.951488074846566, y: -0.970001220703125, z: -4.8109643848612915, lat: -11.25, lon: -101.25}
	pixels[185] = &BallPixel{col: 46, row: 10, x: 0.9188389452174305, y: -1.6031392415364585, z: -4.645882126875221, lat: -18.75, lon: -101.25}
	pixels[186] = &BallPixel{col: 46, row: 9, x: 0.8704967619851232, y: -2.2100728352864585, z: -4.401451820321381, lat: -26.25, lon: -101.25}
	pixels[187] = &BallPixel{col: 46, row: 8, x: 0.8072193386033177, y: -2.779083251953125, z: -4.0815051617100835, lat: -33.75, lon: -101.25}
	pixels[188] = &BallPixel{col: 46, row: 7, x: 0.7300170930102469, y: -3.2997538248697915, z: -3.6911510797217493, lat: -41.25, lon: -101.25}
	pixels[189] = &BallPixel{col: 46, row: 6, x: 0.6401530476287008, y: -3.762969970703125, z: -3.236775735206902, lat: -48.75, lon: -101.25}
	pixels[190] = &BallPixel{col: 46, row: 5, x: 0.5391428293660283, y: -4.160919189453125, z: -2.726042521186173, lat: -56.25, lon: -101.25}
	pixels[191] = &BallPixel{col: 46, row: 4, x: 0.42875466961413616, y: -4.487091064453125, z: -2.1678920628502962, lat: -63.75, lon: -101.25}
	pixels[192] = &BallPixel{col: 46, row: 3, x: 0.3110094042494894, y: -4.736277262369791, z: -1.572542217560113, lat: -71.25, lon: -101.25}
	pixels[193] = &BallPixel{col: 44, row: 1, x: 0.12368733386198667, y: -4.989369710286458, z: -0.2991823703050611, lat: -86.25, lon: -112.5}
	pixels[194] = &BallPixel{col: 44, row: 2, x: 0.37066550552845, y: -4.904571533203126, z: -0.8965880423784258, lat: -78.75, lon: -112.5}
	pixels[195] = &BallPixel{col: 44, row: 3, x: 0.6126058449347817, y: -4.736277262369791, z: -1.4818079024553308, lat: -71.25, lon: -112.5}
	pixels[196] = &BallPixel{col: 44, row: 4, x: 0.8445327152808506, y: -4.487091064453125, z: -2.0428065806627274, lat: -63.75, lon: -112.5}
	pixels[197] = &BallPixel{col: 44, row: 5, x: 1.0619680434465408, y: -4.160919189453125, z: -2.5687522441148762, lat: -56.25, lon: -112.5}
	pixels[198] = &BallPixel{col: 44, row: 6, x: 1.2609313199917478, y: -3.762969970703125, z: -3.0500165969133386, lat: -48.75, lon: -112.5}
	pixels[199] = &BallPixel{col: 44, row: 7, x: 1.43793959915638, y: -3.2997538248697915, z: -3.4781748801469816, lat: -41.25, lon: -112.5}
	pixels[200] = &BallPixel{col: 44, row: 8, x: 1.5900074988603592, y: -2.779083251953125, z: -3.8460058718919763, lat: -33.75, lon: -112.5}
	pixels[201] = &BallPixel{col: 44, row: 9, x: 1.7146472007036209, y: -2.2100728352864585, z: -4.1474918872118005, lat: -26.25, lon: -112.5}
	pixels[202] = &BallPixel{col: 44, row: 10, x: 1.8098684499661126, y: -1.6031392415364585, z: -4.377818778157235, lat: -18.75, lon: -112.5}
	pixels[203] = &BallPixel{col: 44, row: 11, x: 1.8741785556077961, y: -0.970001220703125, z: -4.533375933766367, lat: -11.25, lon: -112.5}
	pixels[204] = &BallPixel{col: 44, row: 12, x: 1.9065823902686436, y: -0.32367960611979163, z: -4.611756280064584, lat: -3.75, lon: -112.5}
	pixels[205] = &BallPixel{col: 44, row: 13, x: 1.9065823902686447, y: 0.32367960611979163, z: -4.611756280064586, lat: 3.75, lon: -112.5}
	pixels[206] = &BallPixel{col: 44, row: 14, x: 1.8741785556077957, y: 0.970001220703125, z: -4.533375933766366, lat: 11.25, lon: -112.5}
	pixels[207] = &BallPixel{col: 44, row: 15, x: 1.8098684499661135, y: 1.6031392415364585, z: -4.377818778157237, lat: 18.75, lon: -112.5}
	pixels[208] = &BallPixel{col: 44, row: 16, x: 1.714647200703622, y: 2.2100728352864585, z: -4.147491887211803, lat: 26.25, lon: -112.5}
	pixels[209] = &BallPixel{col: 44, row: 17, x: 1.5900074988603603, y: 2.779083251953125, z: -3.846005871891979, lat: 33.75, lon: -112.5}
	pixels[210] = &BallPixel{col: 44, row: 18, x: 1.4379395991563801, y: 3.2997538248697915, z: -3.478174880146982, lat: 41.25, lon: -112.5}
	pixels[211] = &BallPixel{col: 44, row: 19, x: 1.2609313199917487, y: 3.762969970703125, z: -3.0500165969133413, lat: 48.75, lon: -112.5}
	pixels[212] = &BallPixel{col: 42, row: 19, x: 1.834058118052782, y: 3.762969970703125, z: -2.746001802074417, lat: 48.75, lon: -123.75}
	pixels[213] = &BallPixel{col: 42, row: 18, x: 2.0915213646367192, y: 3.2997538248697915, z: -3.131482792086902, lat: 41.25, lon: -123.75}
	pixels[214] = &BallPixel{col: 42, row: 17, x: 2.312708166427911, y: 2.779083251953125, z: -3.4626497002318546, lat: 33.75, lon: -123.75}
	pixels[215] = &BallPixel{col: 42, row: 16, x: 2.493999925442041, y: 2.2100728352864585, z: -3.7340846629813362, lat: 26.25, lon: -123.75}
	pixels[216] = &BallPixel{col: 42, row: 15, x: 2.6325017632916574, y: 1.6031392415364585, z: -3.941453389513, lat: 18.75, lon: -123.75}
	pixels[217] = &BallPixel{col: 42, row: 14, x: 2.7260425211861725, y: 0.970001220703125, z: -4.081505161710086, lat: 11.25, lon: -123.75}
	pixels[218] = &BallPixel{col: 42, row: 13, x: 2.7731747599318632, y: 0.32367960611979163, z: -4.152072834161426, lat: 3.75, lon: -123.75}
	pixels[219] = &BallPixel{col: 42, row: 12, x: 2.7731747599318615, y: -0.32367960611979163, z: -4.152072834161423, lat: -3.75, lon: -123.75}
	pixels[220] = &BallPixel{col: 42, row: 11, x: 2.726042521186173, y: -0.970001220703125, z: -4.081505161710087, lat: -11.25, lon: -123.75}
	pixels[221] = &BallPixel{col: 42, row: 10, x: 2.632501763291656, y: -1.6031392415364585, z: -3.941453389512998, lat: -18.75, lon: -123.75}
	pixels[222] = &BallPixel{col: 42, row: 9, x: 2.4939999254420395, y: -2.2100728352864585, z: -3.734084662981334, lat: -26.25, lon: -123.75}
	pixels[223] = &BallPixel{col: 42, row: 8, x: 2.31270816642791, y: -2.779083251953125, z: -3.4626497002318524, lat: -33.75, lon: -123.75}
	pixels[224] = &BallPixel{col: 42, row: 7, x: 2.0915213646367192, y: -3.2997538248697915, z: -3.131482792086902, lat: -41.25, lon: -123.75}
	pixels[225] = &BallPixel{col: 42, row: 6, x: 1.8340581180527804, y: -3.762969970703125, z: -2.7460018020744146, lat: -48.75, lon: -123.75}
	pixels[226] = &BallPixel{col: 42, row: 5, x: 1.544660744257271, y: -4.160919189453125, z: -2.3127081664279117, lat: -56.25, lon: -123.75}
	pixels[227] = &BallPixel{col: 42, row: 4, x: 1.2283952804282303, y: -4.487091064453125, z: -1.839186894086501, lat: -63.75, lon: -123.75}
	pixels[228] = &BallPixel{col: 42, row: 3, x: 0.8910514833405615, y: -4.736277262369791, z: -1.334106566694877, lat: -71.25, lon: -123.75}
	pixels[229] = &BallPixel{col: 43, row: 5, x: 1.3096762707573364, y: -4.160919189453125, z: -2.4525878278654996, lat: -56.25, lon: -118.125}
	pixels[230] = &BallPixel{col: 43, row: 6, x: 1.5550485149142337, y: -3.762969970703125, z: -2.9120883874711576, lat: -48.75, lon: -118.125}
	pixels[231] = &BallPixel{col: 43, row: 7, x: 1.773344672110398, y: -3.2997538248697915, z: -3.320884446438867, lat: -41.25, lon: -118.125}
	pixels[232] = &BallPixel{col: 43, row: 8, x: 1.9608830081415367, y: -2.779083251953125, z: -3.672081341792363, lat: -33.75, lon: -118.125}
	pixels[233] = &BallPixel{col: 43, row: 9, x: 2.11459541117074, y: -2.2100728352864585, z: -3.9599335210514246, lat: -26.25, lon: -118.125}
	pixels[234] = &BallPixel{col: 43, row: 10, x: 2.23202739172848, y: -1.6031392415364585, z: -4.179844542231875, lat: -18.75, lon: -118.125}
	pixels[235] = &BallPixel{col: 43, row: 11, x: 2.31133808271261, y: -0.970001220703125, z: -4.328367073845584, lat: -11.25, lon: -118.125}
	pixels[236] = &BallPixel{col: 43, row: 12, x: 2.3513002393883657, y: -0.32367960611979163, z: -4.403202894900458, lat: -3.75, lon: -118.125}
	pixels[237] = &BallPixel{col: 43, row: 13, x: 2.351300239388367, y: 0.32367960611979163, z: -4.403202894900461, lat: 3.75, lon: -118.125}
	pixels[238] = &BallPixel{col: 43, row: 14, x: 2.3113380827126098, y: 0.970001220703125, z: -4.328367073845583, lat: 11.25, lon: -118.125}
	pixels[239] = &BallPixel{col: 43, row: 15, x: 2.2320273917284807, y: 1.6031392415364585, z: -4.179844542231877, lat: 18.75, lon: -118.125}
	pixels[240] = &BallPixel{col: 43, row: 16, x: 2.1145954111707415, y: 2.2100728352864585, z: -3.959933521051427, lat: 26.25, lon: -118.125}
	pixels[241] = &BallPixel{col: 43, row: 17, x: 1.960883008141538, y: 2.779083251953125, z: -3.672081341792365, lat: 33.75, lon: -118.125}
	pixels[242] = &BallPixel{col: 41, row: 17, x: 2.641883057367524, y: 2.779083251953125, z: -3.2195966176805126, lat: 33.75, lon: -129.375}
	pixels[243] = &BallPixel{col: 41, row: 16, x: 2.8489786319551076, y: 2.2100728352864585, z: -3.471978800010406, lat: 26.25, lon: -129.375}
	pixels[244] = &BallPixel{col: 41, row: 15, x: 3.007193863838136, y: 1.6031392415364585, z: -3.6647917347145307, lat: 18.75, lon: -129.375}
	pixels[245] = &BallPixel{col: 41, row: 14, x: 3.1140485664946027, y: 0.970001220703125, z: -3.7950128806871386, lat: 11.25, lon: -129.375}
	pixels[246] = &BallPixel{col: 41, row: 13, x: 3.1678892822431726, y: 0.32367960611979163, z: -3.8606272105244033, lat: 3.75, lon: -129.375}
	pixels[247] = &BallPixel{col: 41, row: 12, x: 3.167889282243171, y: -0.32367960611979163, z: -3.860627210524401, lat: -3.75, lon: -129.375}
	pixels[248] = &BallPixel{col: 41, row: 11, x: 3.114048566494603, y: -0.970001220703125, z: -3.7950128806871395, lat: -11.25, lon: -129.375}
	pixels[249] = &BallPixel{col: 41, row: 10, x: 3.0071938638381344, y: -1.6031392415364585, z: -3.6647917347145285, lat: -18.75, lon: -129.375}
	pixels[250] = &BallPixel{col: 41, row: 9, x: 2.848978631955106, y: -2.2100728352864585, z: -3.471978800010404, lat: -26.25, lon: -129.375}
	pixels[251] = &BallPixel{col: 41, row: 8, x: 2.6418830573675223, y: -2.779083251953125, z: -3.219596617680511, lat: -33.75, lon: -129.375}
	pixels[252] = &BallPixel{col: 41, row: 7, x: 2.3892140554380608, y: -3.2997538248697915, z: -2.9116752425325125, lat: -41.25, lon: -129.375}
	pixels[253] = &BallPixel{col: 41, row: 6, x: 2.095105270370065, y: -3.762969970703125, z: -2.5532522430759874, lat: -48.75, lon: -129.375}
	pixels[254] = &BallPixel{col: 41, row: 5, x: 1.76451707520755, y: -4.160919189453125, z: -2.15037270152243, lat: -56.25, lon: -129.375}
	pixels[255] = &BallPixel{col: 38, row: 3, x: 1.3341065666948762, y: -4.736277262369791, z: -0.8910514833405617, lat: -71.25, lon: -146.25}
	pixels[256] = &BallPixel{col: 38, row: 4, x: 1.8391868940864997, y: -4.487091064453125, z: -1.2283952804282305, lat: -63.75, lon: -146.25}
	pixels[257] = &BallPixel{col: 38, row: 5, x: 2.3127081664279103, y: -4.160919189453125, z: -1.5446607442572713, lat: -56.25, lon: -146.25}
	pixels[258] = &BallPixel{col: 38, row: 6, x: 2.746001802074413, y: -3.762969970703125, z: -1.8340581180527809, lat: -48.75, lon: -146.25}
	pixels[259] = &BallPixel{col: 38, row: 7, x: 3.1314827920868997, y: -3.2997538248697915, z: -2.0915213646367197, lat: -41.25, lon: -146.25}
	pixels[260] = &BallPixel{col: 38, row: 8, x: 3.46264970023185, y: -2.779083251953125, z: -2.3127081664279103, lat: -33.75, lon: -146.25}
	pixels[261] = &BallPixel{col: 38, row: 9, x: 3.7340846629813313, y: -2.2100728352864585, z: -2.49399992544204, lat: -26.25, lon: -146.25}
	pixels[262] = &BallPixel{col: 38, row: 10, x: 3.9414533895129953, y: -1.6031392415364585, z: -2.6325017632916565, lat: -18.75, lon: -146.25}
	pixels[263] = &BallPixel{col: 38, row: 11, x: 4.081505161710084, y: -0.970001220703125, z: -2.7260425211861734, lat: -11.25, lon: -146.25}
	pixels[264] = &BallPixel{col: 38, row: 12, x: 4.15207283416142, y: -0.32367960611979163, z: -2.7731747599318624, lat: -3.75, lon: -146.25}
	pixels[265] = &BallPixel{col: 38, row: 13, x: 4.152072834161423, y: 0.32367960611979163, z: -2.7731747599318637, lat: 3.75, lon: -146.25}
	pixels[266] = &BallPixel{col: 38, row: 14, x: 4.0815051617100835, y: 0.970001220703125, z: -2.726042521186173, lat: 11.25, lon: -146.25}
	pixels[267] = &BallPixel{col: 38, row: 15, x: 3.9414533895129975, y: 1.6031392415364585, z: -2.6325017632916583, lat: 18.75, lon: -146.25}
	pixels[268] = &BallPixel{col: 38, row: 16, x: 3.7340846629813336, y: 2.2100728352864585, z: -2.4939999254420413, lat: 26.25, lon: -146.25}
	pixels[269] = &BallPixel{col: 38, row: 17, x: 3.4626497002318524, y: 2.779083251953125, z: -2.3127081664279117, lat: 33.75, lon: -146.25}
	pixels[270] = &BallPixel{col: 38, row: 18, x: 3.1314827920869, y: 3.2997538248697915, z: -2.0915213646367197, lat: 41.25, lon: -146.25}
	pixels[271] = &BallPixel{col: 38, row: 19, x: 2.746001802074415, y: 3.762969970703125, z: -1.8340581180527822, lat: 48.75, lon: -146.25}
	pixels[300] = &BallPixel{col: 49, row: 17, x: -0.4042207662132576, y: 2.779083251953125, z: -4.141022826370321, lat: 33.75, lon: -84.375}
	pixels[301] = &BallPixel{col: 49, row: 16, x: -0.4359073814121083, y: 2.2100728352864585, z: -4.465635037806354, lat: 26.25, lon: -84.375}
	pixels[302] = &BallPixel{col: 49, row: 15, x: -0.4601150699697132, y: 1.6031392415364585, z: -4.713629696343563, lat: 18.75, lon: -84.375}
	pixels[303] = &BallPixel{col: 49, row: 14, x: -0.47646435146452915, y: 0.970001220703125, z: -4.881119230587502, lat: 11.25, lon: -84.375}
	pixels[304] = &BallPixel{col: 49, row: 13, x: -0.4847022389488613, y: 0.32367960611979163, z: -4.965511926275216, lat: 3.75, lon: -84.375}
	pixels[305] = &BallPixel{col: 49, row: 12, x: -0.48470223894886105, y: -0.32367960611979163, z: -4.965511926275213, lat: -3.75, lon: -84.375}
	pixels[306] = &BallPixel{col: 49, row: 11, x: -0.4764643514645292, y: -0.970001220703125, z: -4.881119230587503, lat: -11.25, lon: -84.375}
	pixels[307] = &BallPixel{col: 49, row: 10, x: -0.4601150699697129, y: -1.6031392415364585, z: -4.713629696343561, lat: -18.75, lon: -84.375}
	pixels[308] = &BallPixel{col: 49, row: 9, x: -0.435907381412108, y: -2.2100728352864585, z: -4.465635037806352, lat: -26.25, lon: -84.375}
	pixels[309] = &BallPixel{col: 49, row: 8, x: -0.4042207662132573, y: -2.779083251953125, z: -4.141022826370318, lat: -33.75, lon: -84.375}
	pixels[310] = &BallPixel{col: 49, row: 7, x: -0.3655611982685518, y: -3.2997538248697915, z: -3.744976490561386, lat: -41.25, lon: -84.375}
	pixels[311] = &BallPixel{col: 49, row: 6, x: -0.32056114494723, y: -3.762969970703125, z: -3.283975316036959, lat: -48.75, lon: -84.375}
	pixels[312] = &BallPixel{col: 49, row: 5, x: -0.269979567092378, y: -4.160919189453125, z: -2.7657944455859256, lat: -56.25, lon: -84.375}
	pixels[313] = &BallPixel{col: 50, row: 3, x: -0.3110094042494894, y: -4.736277262369791, z: -1.5725422175601134, lat: -71.25, lon: -78.75}
	pixels[314] = &BallPixel{col: 50, row: 4, x: -0.42875466961413616, y: -4.487091064453125, z: -2.1678920628502967, lat: -63.75, lon: -78.75}
	pixels[315] = &BallPixel{col: 50, row: 5, x: -0.5391428293660283, y: -4.160919189453125, z: -2.7260425211861734, lat: -56.25, lon: -78.75}
	pixels[316] = &BallPixel{col: 50, row: 6, x: -0.6401530476287008, y: -3.762969970703125, z: -3.236775735206903, lat: -48.75, lon: -78.75}
	pixels[317] = &BallPixel{col: 50, row: 7, x: -0.7300170930102469, y: -3.2997538248697915, z: -3.6911510797217497, lat: -41.25, lon: -78.75}
	pixels[318] = &BallPixel{col: 50, row: 8, x: -0.8072193386033177, y: -2.779083251953125, z: -4.081505161710084, lat: -33.75, lon: -78.75}
	pixels[319] = &BallPixel{col: 50, row: 9, x: -0.8704967619851232, y: -2.2100728352864585, z: -4.401451820321382, lat: -26.25, lon: -78.75}
	pixels[320] = &BallPixel{col: 50, row: 10, x: -0.9188389452174305, y: -1.6031392415364585, z: -4.645882126875222, lat: -18.75, lon: -78.75}
	pixels[321] = &BallPixel{col: 50, row: 11, x: -0.951488074846566, y: -0.970001220703125, z: -4.810964384861292, lat: -11.25, lon: -78.75}
	pixels[322] = &BallPixel{col: 50, row: 12, x: -0.9679389419034122, y: -0.32367960611979163, z: -4.894144129939378, lat: -3.75, lon: -78.75}
	pixels[323] = &BallPixel{col: 50, row: 13, x: -0.9679389419034128, y: 0.32367960611979163, z: -4.894144129939381, lat: 3.75, lon: -78.75}
	pixels[324] = &BallPixel{col: 50, row: 14, x: -0.9514880748465657, y: 0.970001220703125, z: -4.8109643848612915, lat: 11.25, lon: -78.75}
	pixels[325] = &BallPixel{col: 50, row: 15, x: -0.918838945217431, y: 1.6031392415364585, z: -4.645882126875224, lat: 18.75, lon: -78.75}
	pixels[326] = &BallPixel{col: 50, row: 16, x: -0.8704967619851237, y: 2.2100728352864585, z: -4.401451820321385, lat: 26.25, lon: -78.75}
	pixels[327] = &BallPixel{col: 50, row: 17, x: -0.8072193386033183, y: 2.779083251953125, z: -4.081505161710087, lat: 33.75, lon: -78.75}
	pixels[328] = &BallPixel{col: 50, row: 18, x: -0.730017093010247, y: 3.2997538248697915, z: -3.69115107972175, lat: 41.25, lon: -78.75}
	pixels[329] = &BallPixel{col: 50, row: 19, x: -0.6401530476287013, y: 3.762969970703125, z: -3.2367757352069053, lat: 48.75, lon: -78.75}
	pixels[330] = &BallPixel{col: 51, row: 17, x: -1.2040244041127157, y: 2.779083251953125, z: -3.9828804692369917, lat: 33.75, lon: -73.125}
	pixels[331] = &BallPixel{col: 51, row: 16, x: -1.2984071305138063, y: 2.2100728352864585, z: -4.295095999364279, lat: 26.25, lon: -73.125}
	pixels[332] = &BallPixel{col: 51, row: 15, x: -1.370512896042783, y: 1.6031392415364585, z: -4.533619939795853, lat: 18.75, lon: -73.125}
	pixels[333] = &BallPixel{col: 51, row: 14, x: -1.4192113686469379, y: 0.970001220703125, z: -4.694713182386478, lat: 11.25, lon: -73.125}
	pixels[334] = &BallPixel{col: 51, row: 13, x: -1.4437489936244676, y: 0.32367960611979163, z: -4.775882988372666, lat: 3.75, lon: -73.125}
	pixels[335] = &BallPixel{col: 51, row: 12, x: -1.4437489936244667, y: -0.32367960611979163, z: -4.775882988372663, lat: -3.75, lon: -73.125}
	pixels[336] = &BallPixel{col: 51, row: 11, x: -1.419211368646938, y: -0.970001220703125, z: -4.6947131823864785, lat: -11.25, lon: -73.125}
	pixels[337] = &BallPixel{col: 51, row: 10, x: -1.3705128960427821, y: -1.6031392415364585, z: -4.533619939795851, lat: -18.75, lon: -73.125}
	pixels[338] = &BallPixel{col: 51, row: 9, x: -1.2984071305138054, y: -2.2100728352864585, z: -4.2950959993642766, lat: -26.25, lon: -73.125}
	pixels[339] = &BallPixel{col: 51, row: 8, x: -1.2040244041127148, y: -2.779083251953125, z: -3.9828804692369895, lat: -33.75, lon: -73.125}
	pixels[340] = &BallPixel{col: 51, row: 7, x: -1.088871826243121, y: -3.2997538248697915, z: -3.6019588269409732, lat: -41.25, lon: -73.125}
	pixels[341] = &BallPixel{col: 51, row: 6, x: -0.9548332836595361, y: -3.762969970703125, z: -3.158562919384956, lat: -48.75, lon: -73.125}
	pixels[342] = &BallPixel{col: 51, row: 5, x: -0.8041694404673759, y: -4.160919189453125, z: -2.6601709628594112, lat: -56.25, lon: -73.125}
	pixels[343] = &BallPixel{col: 52, row: 1, x: -0.12368733386198667, y: -4.989369710286458, z: -0.29918237030506106, lat: -86.25, lon: -67.5}
	pixels[344] = &BallPixel{col: 52, row: 2, x: -0.37066550552845, y: -4.904571533203126, z: -0.8965880423784256, lat: -78.75, lon: -67.5}
	pixels[345] = &BallPixel{col: 52, row: 3, x: -0.6126058449347817, y: -4.736277262369791, z: -1.4818079024553303, lat: -71.25, lon: -67.5}
	pixels[346] = &BallPixel{col: 52, row: 4, x: -0.8445327152808506, y: -4.487091064453125, z: -2.042806580662727, lat: -63.75, lon: -67.5}
	pixels[347] = &BallPixel{col: 52, row: 5, x: -1.0619680434465408, y: -4.160919189453125, z: -2.568752244114876, lat: -56.25, lon: -67.5}
	pixels[348] = &BallPixel{col: 52, row: 6, x: -1.2609313199917478, y: -3.762969970703125, z: -3.050016596913338, lat: -48.75, lon: -67.5}
	pixels[349] = &BallPixel{col: 52, row: 7, x: -1.43793959915638, y: -3.2997538248697915, z: -3.4781748801469807, lat: -41.25, lon: -67.5}
	pixels[350] = &BallPixel{col: 52, row: 8, x: -1.5900074988603592, y: -2.779083251953125, z: -3.8460058718919754, lat: -33.75, lon: -67.5}
	pixels[351] = &BallPixel{col: 52, row: 9, x: -1.7146472007036209, y: -2.2100728352864585, z: -4.1474918872118, lat: -26.25, lon: -67.5}
	pixels[352] = &BallPixel{col: 52, row: 10, x: -1.8098684499661126, y: -1.6031392415364585, z: -4.377818778157233, lat: -18.75, lon: -67.5}
	pixels[353] = &BallPixel{col: 52, row: 11, x: -1.8741785556077961, y: -0.970001220703125, z: -4.533375933766366, lat: -11.25, lon: -67.5}
	pixels[354] = &BallPixel{col: 52, row: 12, x: -1.9065823902686436, y: -0.32367960611979163, z: -4.611756280064583, lat: -3.75, lon: -67.5}
	pixels[355] = &BallPixel{col: 52, row: 13, x: -1.9065823902686447, y: 0.32367960611979163, z: -4.611756280064585, lat: 3.75, lon: -67.5}
	pixels[356] = &BallPixel{col: 52, row: 14, x: -1.8741785556077957, y: 0.970001220703125, z: -4.533375933766365, lat: 11.25, lon: -67.5}
	pixels[357] = &BallPixel{col: 52, row: 15, x: -1.8098684499661135, y: 1.6031392415364585, z: -4.377818778157236, lat: 18.75, lon: -67.5}
	pixels[358] = &BallPixel{col: 52, row: 16, x: -1.714647200703622, y: 2.2100728352864585, z: -4.147491887211802, lat: 26.25, lon: -67.5}
	pixels[359] = &BallPixel{col: 52, row: 17, x: -1.5900074988603603, y: 2.779083251953125, z: -3.846005871891978, lat: 33.75, lon: -67.5}
	pixels[360] = &BallPixel{col: 52, row: 18, x: -1.4379395991563801, y: 3.2997538248697915, z: -3.478174880146981, lat: 41.25, lon: -67.5}
	pixels[361] = &BallPixel{col: 52, row: 19, x: -1.2609313199917487, y: 3.762969970703125, z: -3.0500165969133404, lat: 48.75, lon: -67.5}
	pixels[362] = &BallPixel{col: 53, row: 17, x: -1.960883008141538, y: 2.779083251953125, z: -3.6720813417923663, lat: 33.75, lon: -61.875}
	pixels[363] = &BallPixel{col: 53, row: 16, x: -2.1145954111707415, y: 2.2100728352864585, z: -3.959933521051428, lat: 26.25, lon: -61.875}
	pixels[364] = &BallPixel{col: 53, row: 15, x: -2.2320273917284807, y: 1.6031392415364585, z: -4.179844542231879, lat: 18.75, lon: -61.875}
	pixels[365] = &BallPixel{col: 53, row: 14, x: -2.3113380827126098, y: 0.970001220703125, z: -4.328367073845585, lat: 11.25, lon: -61.875}
	pixels[366] = &BallPixel{col: 53, row: 13, x: -2.351300239388367, y: 0.32367960611979163, z: -4.4032028949004625, lat: 3.75, lon: -61.875}
	pixels[367] = &BallPixel{col: 53, row: 12, x: -2.3513002393883657, y: -0.32367960611979163, z: -4.40320289490046, lat: -3.75, lon: -61.875}
	pixels[368] = &BallPixel{col: 53, row: 11, x: -2.31133808271261, y: -0.970001220703125, z: -4.328367073845585, lat: -11.25, lon: -61.875}
	pixels[369] = &BallPixel{col: 53, row: 10, x: -2.23202739172848, y: -1.6031392415364585, z: -4.179844542231876, lat: -18.75, lon: -61.875}
	pixels[370] = &BallPixel{col: 53, row: 9, x: -2.11459541117074, y: -2.2100728352864585, z: -3.959933521051426, lat: -26.25, lon: -61.875}
	pixels[371] = &BallPixel{col: 53, row: 8, x: -1.9608830081415367, y: -2.779083251953125, z: -3.672081341792364, lat: -33.75, lon: -61.875}
	pixels[372] = &BallPixel{col: 53, row: 7, x: -1.773344672110398, y: -3.2997538248697915, z: -3.3208844464388685, lat: -41.25, lon: -61.875}
	pixels[373] = &BallPixel{col: 53, row: 6, x: -1.5550485149142337, y: -3.762969970703125, z: -2.9120883874711585, lat: -48.75, lon: -61.875}
	pixels[374] = &BallPixel{col: 53, row: 5, x: -1.3096762707573364, y: -4.160919189453125, z: -2.4525878278655004, lat: -56.25, lon: -61.875}
	pixels[375] = &BallPixel{col: 54, row: 3, x: -0.8910514833405617, y: -4.736277262369791, z: -1.3341065666948762, lat: -71.25, lon: -56.25}
	pixels[376] = &BallPixel{col: 54, row: 4, x: -1.2283952804282305, y: -4.487091064453125, z: -1.8391868940864997, lat: -63.75, lon: -56.25}
	pixels[377] = &BallPixel{col: 54, row: 5, x: -1.5446607442572713, y: -4.160919189453125, z: -2.3127081664279103, lat: -56.25, lon: -56.25}
	pixels[378] = &BallPixel{col: 54, row: 6, x: -1.8340581180527809, y: -3.762969970703125, z: -2.746001802074413, lat: -48.75, lon: -56.25}
	pixels[379] = &BallPixel{col: 54, row: 7, x: -2.0915213646367197, y: -3.2997538248697915, z: -3.1314827920868997, lat: -41.25, lon: -56.25}
	pixels[380] = &BallPixel{col: 54, row: 8, x: -2.3127081664279103, y: -2.779083251953125, z: -3.46264970023185, lat: -33.75, lon: -56.25}
	pixels[381] = &BallPixel{col: 54, row: 9, x: -2.49399992544204, y: -2.2100728352864585, z: -3.7340846629813313, lat: -26.25, lon: -56.25}
	pixels[382] = &BallPixel{col: 54, row: 10, x: -2.6325017632916565, y: -1.6031392415364585, z: -3.9414533895129953, lat: -18.75, lon: -56.25}
	pixels[383] = &BallPixel{col: 54, row: 11, x: -2.7260425211861734, y: -0.970001220703125, z: -4.081505161710084, lat: -11.25, lon: -56.25}
	pixels[384] = &BallPixel{col: 54, row: 12, x: -2.7731747599318624, y: -0.32367960611979163, z: -4.15207283416142, lat: -3.75, lon: -56.25}
	pixels[385] = &BallPixel{col: 54, row: 13, x: -2.7731747599318637, y: 0.32367960611979163, z: -4.152072834161423, lat: 3.75, lon: -56.25}
	pixels[386] = &BallPixel{col: 54, row: 14, x: -2.726042521186173, y: 0.970001220703125, z: -4.0815051617100835, lat: 11.25, lon: -56.25}
	pixels[387] = &BallPixel{col: 54, row: 15, x: -2.6325017632916583, y: 1.6031392415364585, z: -3.9414533895129975, lat: 18.75, lon: -56.25}
	pixels[388] = &BallPixel{col: 54, row: 16, x: -2.4939999254420413, y: 2.2100728352864585, z: -3.7340846629813336, lat: 26.25, lon: -56.25}
	pixels[389] = &BallPixel{col: 54, row: 17, x: -2.3127081664279117, y: 2.779083251953125, z: -3.4626497002318524, lat: 33.75, lon: -56.25}
	pixels[390] = &BallPixel{col: 54, row: 18, x: -2.0915213646367197, y: 3.2997538248697915, z: -3.1314827920869, lat: 41.25, lon: -56.25}
	pixels[391] = &BallPixel{col: 54, row: 19, x: -1.8340581180527822, y: 3.762969970703125, z: -2.746001802074415, lat: 48.75, lon: -56.25}
	pixels[392] = &BallPixel{col: 55, row: 17, x: -2.641883057367524, y: 2.779083251953125, z: -3.2195966176805104, lat: 33.75, lon: -50.625}
	pixels[393] = &BallPixel{col: 55, row: 16, x: -2.8489786319551076, y: 2.2100728352864585, z: -3.4719788000104037, lat: 26.25, lon: -50.625}
	pixels[394] = &BallPixel{col: 55, row: 15, x: -3.007193863838136, y: 1.6031392415364585, z: -3.664791734714528, lat: 18.75, lon: -50.625}
	pixels[395] = &BallPixel{col: 55, row: 14, x: -3.1140485664946027, y: 0.970001220703125, z: -3.795012880687136, lat: 11.25, lon: -50.625}
	pixels[396] = &BallPixel{col: 55, row: 13, x: -3.1678892822431726, y: 0.32367960611979163, z: -3.8606272105244006, lat: 3.75, lon: -50.625}
	pixels[397] = &BallPixel{col: 55, row: 12, x: -3.167889282243171, y: -0.32367960611979163, z: -3.8606272105243984, lat: -3.75, lon: -50.625}
	pixels[398] = &BallPixel{col: 55, row: 11, x: -3.114048566494603, y: -0.970001220703125, z: -3.795012880687137, lat: -11.25, lon: -50.625}
	pixels[399] = &BallPixel{col: 55, row: 10, x: -3.0071938638381344, y: -1.6031392415364585, z: -3.664791734714526, lat: -18.75, lon: -50.625}
	pixels[400] = &BallPixel{col: 55, row: 9, x: -2.848978631955106, y: -2.2100728352864585, z: -3.4719788000104015, lat: -26.25, lon: -50.625}
	pixels[401] = &BallPixel{col: 55, row: 8, x: -2.6418830573675223, y: -2.779083251953125, z: -3.2195966176805086, lat: -33.75, lon: -50.625}
	pixels[402] = &BallPixel{col: 55, row: 7, x: -2.3892140554380608, y: -3.2997538248697915, z: -2.9116752425325103, lat: -41.25, lon: -50.625}
	pixels[403] = &BallPixel{col: 55, row: 6, x: -2.095105270370065, y: -3.762969970703125, z: -2.5532522430759856, lat: -48.75, lon: -50.625}
	pixels[404] = &BallPixel{col: 55, row: 5, x: -1.76451707520755, y: -4.160919189453125, z: -2.1503727015224285, lat: -56.25, lon: -50.625}
	pixels[405] = &BallPixel{col: 57, row: 5, x: -2.1503727015224285, y: -4.160919189453125, z: -1.76451707520755, lat: -56.25, lon: -39.375}
	pixels[406] = &BallPixel{col: 57, row: 6, x: -2.5532522430759856, y: -3.762969970703125, z: -2.095105270370065, lat: -48.75, lon: -39.375}
	pixels[407] = &BallPixel{col: 57, row: 7, x: -2.9116752425325103, y: -3.2997538248697915, z: -2.3892140554380608, lat: -41.25, lon: -39.375}
	pixels[408] = &BallPixel{col: 57, row: 8, x: -3.2195966176805086, y: -2.779083251953125, z: -2.6418830573675223, lat: -33.75, lon: -39.375}
	pixels[409] = &BallPixel{col: 57, row: 9, x: -3.4719788000104015, y: -2.2100728352864585, z: -2.848978631955106, lat: -26.25, lon: -39.375}
	pixels[410] = &BallPixel{col: 57, row: 10, x: -3.664791734714526, y: -1.6031392415364585, z: -3.0071938638381344, lat: -18.75, lon: -39.375}
	pixels[411] = &BallPixel{col: 57, row: 11, x: -3.795012880687137, y: -0.970001220703125, z: -3.114048566494603, lat: -11.25, lon: -39.375}
	pixels[412] = &BallPixel{col: 57, row: 12, x: -3.8606272105243984, y: -0.32367960611979163, z: -3.167889282243171, lat: -3.75, lon: -39.375}
	pixels[413] = &BallPixel{col: 57, row: 13, x: -3.8606272105244006, y: 0.32367960611979163, z: -3.1678892822431726, lat: 3.75, lon: -39.375}
	pixels[414] = &BallPixel{col: 57, row: 14, x: -3.795012880687136, y: 0.970001220703125, z: -3.1140485664946027, lat: 11.25, lon: -39.375}
	pixels[415] = &BallPixel{col: 57, row: 15, x: -3.664791734714528, y: 1.6031392415364585, z: -3.007193863838136, lat: 18.75, lon: -39.375}
	pixels[416] = &BallPixel{col: 57, row: 16, x: -3.4719788000104037, y: 2.2100728352864585, z: -2.8489786319551076, lat: 26.25, lon: -39.375}
	pixels[417] = &BallPixel{col: 57, row: 17, x: -3.2195966176805104, y: 2.779083251953125, z: -2.641883057367524, lat: 33.75, lon: -39.375}
	pixels[418] = &BallPixel{col: 58, row: 19, x: -2.746001802074415, y: 3.762969970703125, z: -1.8340581180527822, lat: 48.75, lon: -33.75}
	pixels[419] = &BallPixel{col: 58, row: 18, x: -3.1314827920869, y: 3.2997538248697915, z: -2.0915213646367197, lat: 41.25, lon: -33.75}
	pixels[420] = &BallPixel{col: 58, row: 17, x: -3.4626497002318524, y: 2.779083251953125, z: -2.3127081664279117, lat: 33.75, lon: -33.75}
	pixels[421] = &BallPixel{col: 58, row: 16, x: -3.7340846629813336, y: 2.2100728352864585, z: -2.4939999254420413, lat: 26.25, lon: -33.75}
	pixels[422] = &BallPixel{col: 58, row: 15, x: -3.9414533895129975, y: 1.6031392415364585, z: -2.6325017632916583, lat: 18.75, lon: -33.75}
	pixels[423] = &BallPixel{col: 58, row: 14, x: -4.0815051617100835, y: 0.970001220703125, z: -2.726042521186173, lat: 11.25, lon: -33.75}
	pixels[424] = &BallPixel{col: 58, row: 13, x: -4.152072834161423, y: 0.32367960611979163, z: -2.7731747599318637, lat: 3.75, lon: -33.75}
	pixels[425] = &BallPixel{col: 58, row: 12, x: -4.15207283416142, y: -0.32367960611979163, z: -2.7731747599318624, lat: -3.75, lon: -33.75}
	pixels[426] = &BallPixel{col: 58, row: 11, x: -4.081505161710084, y: -0.970001220703125, z: -2.7260425211861734, lat: -11.25, lon: -33.75}
	pixels[427] = &BallPixel{col: 58, row: 10, x: -3.9414533895129953, y: -1.6031392415364585, z: -2.6325017632916565, lat: -18.75, lon: -33.75}
	pixels[428] = &BallPixel{col: 58, row: 9, x: -3.7340846629813313, y: -2.2100728352864585, z: -2.49399992544204, lat: -26.25, lon: -33.75}
	pixels[429] = &BallPixel{col: 58, row: 8, x: -3.46264970023185, y: -2.779083251953125, z: -2.3127081664279103, lat: -33.75, lon: -33.75}
	pixels[430] = &BallPixel{col: 58, row: 7, x: -3.1314827920868997, y: -3.2997538248697915, z: -2.0915213646367197, lat: -41.25, lon: -33.75}
	pixels[431] = &BallPixel{col: 58, row: 6, x: -2.746001802074413, y: -3.762969970703125, z: -1.8340581180527809, lat: -48.75, lon: -33.75}
	pixels[432] = &BallPixel{col: 58, row: 5, x: -2.3127081664279103, y: -4.160919189453125, z: -1.5446607442572713, lat: -56.25, lon: -33.75}
	pixels[433] = &BallPixel{col: 58, row: 4, x: -1.8391868940864997, y: -4.487091064453125, z: -1.2283952804282305, lat: -63.75, lon: -33.75}
	pixels[434] = &BallPixel{col: 58, row: 3, x: -1.3341065666948762, y: -4.736277262369791, z: -0.8910514833405617, lat: -71.25, lon: -33.75}
	pixels[450] = &BallPixel{col: 59, row: 17, x: -3.6720813417923663, y: 2.779083251953125, z: -1.9608830081415376, lat: 33.75, lon: -28.125}
	pixels[451] = &BallPixel{col: 59, row: 16, x: -3.959933521051428, y: 2.2100728352864585, z: -2.114595411170741, lat: 26.25, lon: -28.125}
	pixels[452] = &BallPixel{col: 59, row: 15, x: -4.179844542231879, y: 1.6031392415364585, z: -2.2320273917284803, lat: 18.75, lon: -28.125}
	pixels[453] = &BallPixel{col: 59, row: 14, x: -4.328367073845585, y: 0.970001220703125, z: -2.3113380827126093, lat: 11.25, lon: -28.125}
	pixels[454] = &BallPixel{col: 59, row: 13, x: -4.4032028949004625, y: 0.32367960611979163, z: -2.3513002393883666, lat: 3.75, lon: -28.125}
	pixels[455] = &BallPixel{col: 59, row: 12, x: -4.40320289490046, y: -0.32367960611979163, z: -2.3513002393883653, lat: -3.75, lon: -28.125}
	pixels[456] = &BallPixel{col: 59, row: 11, x: -4.328367073845585, y: -0.970001220703125, z: -2.3113380827126098, lat: -11.25, lon: -28.125}
	pixels[457] = &BallPixel{col: 59, row: 10, x: -4.179844542231876, y: -1.6031392415364585, z: -2.232027391728479, lat: -18.75, lon: -28.125}
	pixels[458] = &BallPixel{col: 59, row: 9, x: -3.959933521051426, y: -2.2100728352864585, z: -2.1145954111707397, lat: -26.25, lon: -28.125}
	pixels[459] = &BallPixel{col: 59, row: 8, x: -3.672081341792364, y: -2.779083251953125, z: -1.9608830081415363, lat: -33.75, lon: -28.125}
	pixels[460] = &BallPixel{col: 59, row: 7, x: -3.3208844464388685, y: -3.2997538248697915, z: -1.7733446721103976, lat: -41.25, lon: -28.125}
	pixels[461] = &BallPixel{col: 59, row: 6, x: -2.9120883874711585, y: -3.762969970703125, z: -1.5550485149142335, lat: -48.75, lon: -28.125}
	pixels[462] = &BallPixel{col: 60, row: 1, x: -0.29918237030506106, y: -4.989369710286458, z: -0.12368733386198667, lat: -86.25, lon: -22.5}
	pixels[463] = &BallPixel{col: 60, row: 2, x: -0.8965880423784256, y: -4.904571533203126, z: -0.37066550552845, lat: -78.75, lon: -22.5}
	pixels[464] = &BallPixel{col: 60, row: 3, x: -1.4818079024553303, y: -4.736277262369791, z: -0.6126058449347817, lat: -71.25, lon: -22.5}
	pixels[465] = &BallPixel{col: 60, row: 4, x: -2.042806580662727, y: -4.487091064453125, z: -0.8445327152808506, lat: -63.75, lon: -22.5}
	pixels[466] = &BallPixel{col: 60, row: 5, x: -2.568752244114876, y: -4.160919189453125, z: -1.0619680434465408, lat: -56.25, lon: -22.5}
	pixels[467] = &BallPixel{col: 60, row: 6, x: -3.050016596913338, y: -3.762969970703125, z: -1.2609313199917478, lat: -48.75, lon: -22.5}
	pixels[468] = &BallPixel{col: 60, row: 7, x: -3.4781748801469807, y: -3.2997538248697915, z: -1.43793959915638, lat: -41.25, lon: -22.5}
	pixels[469] = &BallPixel{col: 60, row: 8, x: -3.8460058718919754, y: -2.779083251953125, z: -1.5900074988603592, lat: -33.75, lon: -22.5}
	pixels[470] = &BallPixel{col: 60, row: 9, x: -4.1474918872118, y: -2.2100728352864585, z: -1.7146472007036209, lat: -26.25, lon: -22.5}
	pixels[471] = &BallPixel{col: 60, row: 10, x: -4.377818778157233, y: -1.6031392415364585, z: -1.8098684499661126, lat: -18.75, lon: -22.5}
	pixels[472] = &BallPixel{col: 60, row: 11, x: -4.533375933766366, y: -0.970001220703125, z: -1.8741785556077961, lat: -11.25, lon: -22.5}
	pixels[473] = &BallPixel{col: 60, row: 12, x: -4.611756280064583, y: -0.32367960611979163, z: -1.9065823902686436, lat: -3.75, lon: -22.5}
	pixels[474] = &BallPixel{col: 60, row: 13, x: -4.611756280064585, y: 0.32367960611979163, z: -1.9065823902686447, lat: 3.75, lon: -22.5}
	pixels[475] = &BallPixel{col: 60, row: 14, x: -4.533375933766365, y: 0.970001220703125, z: -1.8741785556077957, lat: 11.25, lon: -22.5}
	pixels[476] = &BallPixel{col: 60, row: 15, x: -4.377818778157236, y: 1.6031392415364585, z: -1.8098684499661135, lat: 18.75, lon: -22.5}
	pixels[477] = &BallPixel{col: 60, row: 16, x: -4.147491887211802, y: 2.2100728352864585, z: -1.714647200703622, lat: 26.25, lon: -22.5}
	pixels[478] = &BallPixel{col: 60, row: 17, x: -3.846005871891978, y: 2.779083251953125, z: -1.5900074988603603, lat: 33.75, lon: -22.5}
	pixels[479] = &BallPixel{col: 60, row: 18, x: -3.478174880146981, y: 3.2997538248697915, z: -1.4379395991563801, lat: 41.25, lon: -22.5}
	pixels[480] = &BallPixel{col: 60, row: 19, x: -3.0500165969133404, y: 3.762969970703125, z: -1.2609313199917487, lat: 48.75, lon: -22.5}
	pixels[481] = &BallPixel{col: 61, row: 17, x: -3.9828804692369917, y: 2.779083251953125, z: -1.2040244041127162, lat: 33.75, lon: -16.875}
	pixels[482] = &BallPixel{col: 61, row: 16, x: -4.295095999364279, y: 2.2100728352864585, z: -1.2984071305138067, lat: 26.25, lon: -16.875}
	pixels[483] = &BallPixel{col: 61, row: 15, x: -4.533619939795853, y: 1.6031392415364585, z: -1.3705128960427835, lat: 18.75, lon: -16.875}
	pixels[484] = &BallPixel{col: 61, row: 14, x: -4.694713182386478, y: 0.970001220703125, z: -1.4192113686469383, lat: 11.25, lon: -16.875}
	pixels[485] = &BallPixel{col: 61, row: 13, x: -4.775882988372666, y: 0.32367960611979163, z: -1.443748993624468, lat: 3.75, lon: -16.875}
	pixels[486] = &BallPixel{col: 61, row: 12, x: -4.775882988372663, y: -0.32367960611979163, z: -1.4437489936244674, lat: -3.75, lon: -16.875}
	pixels[487] = &BallPixel{col: 61, row: 11, x: -4.6947131823864785, y: -0.970001220703125, z: -1.4192113686469388, lat: -11.25, lon: -16.875}
	pixels[488] = &BallPixel{col: 61, row: 10, x: -4.533619939795851, y: -1.6031392415364585, z: -1.3705128960427826, lat: -18.75, lon: -16.875}
	pixels[489] = &BallPixel{col: 61, row: 9, x: -4.2950959993642766, y: -2.2100728352864585, z: -1.2984071305138059, lat: -26.25, lon: -16.875}
	pixels[490] = &BallPixel{col: 61, row: 8, x: -3.9828804692369895, y: -2.779083251953125, z: -1.2040244041127153, lat: -33.75, lon: -16.875}
	pixels[491] = &BallPixel{col: 61, row: 7, x: -3.6019588269409732, y: -3.2997538248697915, z: -1.0888718262431214, lat: -41.25, lon: -16.875}
	pixels[492] = &BallPixel{col: 61, row: 6, x: -3.158562919384956, y: -3.762969970703125, z: -0.9548332836595366, lat: -48.75, lon: -16.875}
	pixels[493] = &BallPixel{col: 61, row: 5, x: -2.6601709628594112, y: -4.160919189453125, z: -0.8041694404673763, lat: -56.25, lon: -16.875}
	pixels[494] = &BallPixel{col: 62, row: 3, x: -1.5725422175601134, y: -4.736277262369791, z: -0.3110094042494894, lat: -71.25, lon: -11.25}
	pixels[495] = &BallPixel{col: 62, row: 4, x: -2.1678920628502967, y: -4.487091064453125, z: -0.42875466961413616, lat: -63.75, lon: -11.25}
	pixels[496] = &BallPixel{col: 62, row: 5, x: -2.7260425211861734, y: -4.160919189453125, z: -0.5391428293660283, lat: -56.25, lon: -11.25}
	pixels[497] = &BallPixel{col: 62, row: 6, x: -3.236775735206903, y: -3.762969970703125, z: -0.6401530476287008, lat: -48.75, lon: -11.25}
	pixels[498] = &BallPixel{col: 62, row: 7, x: -3.6911510797217497, y: -3.2997538248697915, z: -0.7300170930102469, lat: -41.25, lon: -11.25}
	pixels[499] = &BallPixel{col: 62, row: 8, x: -4.081505161710084, y: -2.779083251953125, z: -0.8072193386033177, lat: -33.75, lon: -11.25}
	pixels[500] = &BallPixel{col: 62, row: 9, x: -4.401451820321382, y: -2.2100728352864585, z: -0.8704967619851232, lat: -26.25, lon: -11.25}
	pixels[501] = &BallPixel{col: 62, row: 10, x: -4.645882126875222, y: -1.6031392415364585, z: -0.9188389452174305, lat: -18.75, lon: -11.25}
	pixels[502] = &BallPixel{col: 62, row: 11, x: -4.810964384861292, y: -0.970001220703125, z: -0.951488074846566, lat: -11.25, lon: -11.25}
	pixels[503] = &BallPixel{col: 62, row: 12, x: -4.894144129939378, y: -0.32367960611979163, z: -0.9679389419034122, lat: -3.75, lon: -11.25}
	pixels[504] = &BallPixel{col: 62, row: 13, x: -4.894144129939381, y: 0.32367960611979163, z: -0.9679389419034128, lat: 3.75, lon: -11.25}
	pixels[505] = &BallPixel{col: 62, row: 14, x: -4.8109643848612915, y: 0.970001220703125, z: -0.9514880748465657, lat: 11.25, lon: -11.25}
	pixels[506] = &BallPixel{col: 62, row: 15, x: -4.645882126875224, y: 1.6031392415364585, z: -0.918838945217431, lat: 18.75, lon: -11.25}
	pixels[507] = &BallPixel{col: 62, row: 16, x: -4.401451820321385, y: 2.2100728352864585, z: -0.8704967619851237, lat: 26.25, lon: -11.25}
	pixels[508] = &BallPixel{col: 62, row: 17, x: -4.081505161710087, y: 2.779083251953125, z: -0.8072193386033183, lat: 33.75, lon: -11.25}
	pixels[509] = &BallPixel{col: 62, row: 18, x: -3.69115107972175, y: 3.2997538248697915, z: -0.730017093010247, lat: 41.25, lon: -11.25}
	pixels[510] = &BallPixel{col: 62, row: 19, x: -3.2367757352069053, y: 3.762969970703125, z: -0.6401530476287013, lat: 48.75, lon: -11.25}
	pixels[511] = &BallPixel{col: 63, row: 17, x: -4.141022826370321, y: 2.779083251953125, z: -0.40422076621325714, lat: 33.75, lon: -5.625}
	pixels[512] = &BallPixel{col: 63, row: 16, x: -4.465635037806354, y: 2.2100728352864585, z: -0.4359073814121078, lat: 26.25, lon: -5.625}
	pixels[513] = &BallPixel{col: 63, row: 15, x: -4.713629696343563, y: 1.6031392415364585, z: -0.46011506996971263, lat: 18.75, lon: -5.625}
	pixels[514] = &BallPixel{col: 63, row: 14, x: -4.881119230587502, y: 0.970001220703125, z: -0.4764643514645286, lat: 11.25, lon: -5.625}
	pixels[515] = &BallPixel{col: 63, row: 13, x: -4.965511926275216, y: 0.32367960611979163, z: -0.4847022389488607, lat: 3.75, lon: -5.625}
	pixels[516] = &BallPixel{col: 63, row: 12, x: -4.965511926275213, y: -0.32367960611979163, z: -0.4847022389488605, lat: -3.75, lon: -5.625}
	pixels[517] = &BallPixel{col: 63, row: 11, x: -4.881119230587503, y: -0.970001220703125, z: -0.4764643514645287, lat: -11.25, lon: -5.625}
	pixels[518] = &BallPixel{col: 63, row: 10, x: -4.713629696343561, y: -1.6031392415364585, z: -0.4601150699697124, lat: -18.75, lon: -5.625}
	pixels[519] = &BallPixel{col: 63, row: 9, x: -4.465635037806352, y: -2.2100728352864585, z: -0.4359073814121075, lat: -26.25, lon: -5.625}
	pixels[520] = &BallPixel{col: 63, row: 8, x: -4.141022826370318, y: -2.779083251953125, z: -0.40422076621325687, lat: -33.75, lon: -5.625}
	pixels[521] = &BallPixel{col: 63, row: 7, x: -3.744976490561386, y: -3.2997538248697915, z: -0.36556119826855143, lat: -41.25, lon: -5.625}
	pixels[522] = &BallPixel{col: 63, row: 6, x: -3.283975316036959, y: -3.762969970703125, z: -0.3205611449472296, lat: -48.75, lon: -5.625}
	pixels[523] = &BallPixel{col: 63, row: 5, x: -2.7657944455859256, y: -4.160919189453125, z: -0.2699795670923777, lat: -56.25, lon: -5.625}
	pixels[524] = &BallPixel{col: 1, row: 5, x: -2.765794445585925, y: -4.160919189453125, z: 0.2699795670923777, lat: -56.25, lon: 5.625}
	pixels[525] = &BallPixel{col: 1, row: 6, x: -3.2839753160369587, y: -3.762969970703125, z: 0.3205611449472296, lat: -48.75, lon: 5.625}
	pixels[526] = &BallPixel{col: 1, row: 7, x: -3.744976490561385, y: -3.2997538248697915, z: 0.36556119826855143, lat: -41.25, lon: 5.625}
	pixels[527] = &BallPixel{col: 1, row: 8, x: -4.1410228263703175, y: -2.779083251953125, z: 0.40422076621325687, lat: -33.75, lon: 5.625}
	pixels[528] = &BallPixel{col: 1, row: 9, x: -4.465635037806351, y: -2.2100728352864585, z: 0.4359073814121075, lat: -26.25, lon: 5.625}
	pixels[529] = &BallPixel{col: 1, row: 10, x: -4.713629696343559, y: -1.6031392415364585, z: 0.4601150699697124, lat: -18.75, lon: 5.625}
	pixels[530] = &BallPixel{col: 1, row: 11, x: -4.881119230587502, y: -0.970001220703125, z: 0.4764643514645287, lat: -11.25, lon: 5.625}
	pixels[531] = &BallPixel{col: 1, row: 12, x: -4.965511926275212, y: -0.32367960611979163, z: 0.4847022389488605, lat: -3.75, lon: 5.625}
	pixels[532] = &BallPixel{col: 1, row: 13, x: -4.965511926275215, y: 0.32367960611979163, z: 0.4847022389488607, lat: 3.75, lon: 5.625}
	pixels[533] = &BallPixel{col: 1, row: 14, x: -4.881119230587501, y: 0.970001220703125, z: 0.4764643514645286, lat: 11.25, lon: 5.625}
	pixels[534] = &BallPixel{col: 1, row: 15, x: -4.7136296963435615, y: 1.6031392415364585, z: 0.46011506996971263, lat: 18.75, lon: 5.625}
	pixels[535] = &BallPixel{col: 1, row: 16, x: -4.465635037806353, y: 2.2100728352864585, z: 0.4359073814121078, lat: 26.25, lon: 5.625}
	pixels[536] = &BallPixel{col: 1, row: 17, x: -4.14102282637032, y: 2.779083251953125, z: 0.40422076621325714, lat: 33.75, lon: 5.625}
	pixels[537] = &BallPixel{col: 2, row: 19, x: -3.236775735206905, y: 3.762969970703125, z: 0.6401530476287013, lat: 48.75, lon: 11.25}
	pixels[538] = &BallPixel{col: 2, row: 18, x: -3.6911510797217497, y: 3.2997538248697915, z: 0.730017093010247, lat: 41.25, lon: 11.25}
	pixels[539] = &BallPixel{col: 2, row: 17, x: -4.081505161710086, y: 2.779083251953125, z: 0.8072193386033183, lat: 33.75, lon: 11.25}
	pixels[540] = &BallPixel{col: 2, row: 16, x: -4.401451820321384, y: 2.2100728352864585, z: 0.8704967619851237, lat: 26.25, lon: 11.25}
	pixels[541] = &BallPixel{col: 2, row: 15, x: -4.6458821268752235, y: 1.6031392415364585, z: 0.918838945217431, lat: 18.75, lon: 11.25}
	pixels[542] = &BallPixel{col: 2, row: 14, x: -4.810964384861291, y: 0.970001220703125, z: 0.9514880748465657, lat: 11.25, lon: 11.25}
	pixels[543] = &BallPixel{col: 2, row: 13, x: -4.894144129939379, y: 0.32367960611979163, z: 0.9679389419034128, lat: 3.75, lon: 11.25}
	pixels[544] = &BallPixel{col: 2, row: 12, x: -4.894144129939376, y: -0.32367960611979163, z: 0.9679389419034122, lat: -3.75, lon: 11.25}
	pixels[545] = &BallPixel{col: 2, row: 11, x: -4.8109643848612915, y: -0.970001220703125, z: 0.951488074846566, lat: -11.25, lon: 11.25}
	pixels[546] = &BallPixel{col: 2, row: 10, x: -4.645882126875221, y: -1.6031392415364585, z: 0.9188389452174305, lat: -18.75, lon: 11.25}
	pixels[547] = &BallPixel{col: 2, row: 9, x: -4.401451820321381, y: -2.2100728352864585, z: 0.8704967619851232, lat: -26.25, lon: 11.25}
	pixels[548] = &BallPixel{col: 2, row: 8, x: -4.0815051617100835, y: -2.779083251953125, z: 0.8072193386033177, lat: -33.75, lon: 11.25}
	pixels[549] = &BallPixel{col: 2, row: 7, x: -3.6911510797217493, y: -3.2997538248697915, z: 0.7300170930102469, lat: -41.25, lon: 11.25}
	pixels[550] = &BallPixel{col: 2, row: 6, x: -3.236775735206902, y: -3.762969970703125, z: 0.6401530476287008, lat: -48.75, lon: 11.25}
	pixels[551] = &BallPixel{col: 2, row: 5, x: -2.726042521186173, y: -4.160919189453125, z: 0.5391428293660283, lat: -56.25, lon: 11.25}
	pixels[552] = &BallPixel{col: 2, row: 4, x: -2.1678920628502962, y: -4.487091064453125, z: 0.42875466961413616, lat: -63.75, lon: 11.25}
	pixels[553] = &BallPixel{col: 2, row: 3, x: -1.572542217560113, y: -4.736277262369791, z: 0.3110094042494894, lat: -71.25, lon: 11.25}
	pixels[554] = &BallPixel{col: 3, row: 5, x: -2.6601709628594117, y: -4.160919189453125, z: 0.8041694404673763, lat: -56.25, lon: 16.875}
	pixels[555] = &BallPixel{col: 3, row: 6, x: -3.1585629193849565, y: -3.762969970703125, z: 0.9548332836595366, lat: -48.75, lon: 16.875}
	pixels[556] = &BallPixel{col: 3, row: 7, x: -3.6019588269409737, y: -3.2997538248697915, z: 1.0888718262431214, lat: -41.25, lon: 16.875}
	pixels[557] = &BallPixel{col: 3, row: 8, x: -3.98288046923699, y: -2.779083251953125, z: 1.2040244041127153, lat: -33.75, lon: 16.875}
	pixels[558] = &BallPixel{col: 3, row: 9, x: -4.2950959993642766, y: -2.2100728352864585, z: 1.2984071305138059, lat: -26.25, lon: 16.875}
	pixels[559] = &BallPixel{col: 3, row: 10, x: -4.533619939795852, y: -1.6031392415364585, z: 1.3705128960427826, lat: -18.75, lon: 16.875}
	pixels[560] = &BallPixel{col: 3, row: 11, x: -4.6947131823864785, y: -0.970001220703125, z: 1.4192113686469388, lat: -11.25, lon: 16.875}
	pixels[561] = &BallPixel{col: 3, row: 12, x: -4.775882988372664, y: -0.32367960611979163, z: 1.4437489936244674, lat: -3.75, lon: 16.875}
	pixels[562] = &BallPixel{col: 3, row: 13, x: -4.775882988372667, y: 0.32367960611979163, z: 1.443748993624468, lat: 3.75, lon: 16.875}
	pixels[563] = &BallPixel{col: 3, row: 14, x: -4.694713182386478, y: 0.970001220703125, z: 1.4192113686469383, lat: 11.25, lon: 16.875}
	pixels[564] = &BallPixel{col: 3, row: 15, x: -4.533619939795853, y: 1.6031392415364585, z: 1.3705128960427835, lat: 18.75, lon: 16.875}
	pixels[565] = &BallPixel{col: 3, row: 16, x: -4.295095999364279, y: 2.2100728352864585, z: 1.2984071305138067, lat: 26.25, lon: 16.875}
	pixels[566] = &BallPixel{col: 3, row: 17, x: -3.982880469236992, y: 2.779083251953125, z: 1.2040244041127162, lat: 33.75, lon: 16.875}
	pixels[567] = &BallPixel{col: 4, row: 19, x: -3.0500165969133413, y: 3.762969970703125, z: 1.2609313199917487, lat: 48.75, lon: 22.5}
	pixels[568] = &BallPixel{col: 4, row: 18, x: -3.478174880146982, y: 3.2997538248697915, z: 1.4379395991563801, lat: 41.25, lon: 22.5}
	pixels[569] = &BallPixel{col: 4, row: 17, x: -3.846005871891979, y: 2.779083251953125, z: 1.5900074988603603, lat: 33.75, lon: 22.5}
	pixels[570] = &BallPixel{col: 4, row: 16, x: -4.147491887211803, y: 2.2100728352864585, z: 1.714647200703622, lat: 26.25, lon: 22.5}
	pixels[571] = &BallPixel{col: 4, row: 15, x: -4.377818778157237, y: 1.6031392415364585, z: 1.8098684499661135, lat: 18.75, lon: 22.5}
	pixels[572] = &BallPixel{col: 4, row: 14, x: -4.533375933766366, y: 0.970001220703125, z: 1.8741785556077957, lat: 11.25, lon: 22.5}
	pixels[573] = &BallPixel{col: 4, row: 13, x: -4.611756280064586, y: 0.32367960611979163, z: 1.9065823902686447, lat: 3.75, lon: 22.5}
	pixels[574] = &BallPixel{col: 4, row: 12, x: -4.611756280064584, y: -0.32367960611979163, z: 1.9065823902686436, lat: -3.75, lon: 22.5}
	pixels[575] = &BallPixel{col: 4, row: 11, x: -4.533375933766367, y: -0.970001220703125, z: 1.8741785556077961, lat: -11.25, lon: 22.5}
	pixels[576] = &BallPixel{col: 4, row: 10, x: -4.377818778157235, y: -1.6031392415364585, z: 1.8098684499661126, lat: -18.75, lon: 22.5}
	pixels[577] = &BallPixel{col: 4, row: 9, x: -4.1474918872118005, y: -2.2100728352864585, z: 1.7146472007036209, lat: -26.25, lon: 22.5}
	pixels[578] = &BallPixel{col: 4, row: 8, x: -3.8460058718919763, y: -2.779083251953125, z: 1.5900074988603592, lat: -33.75, lon: 22.5}
	pixels[579] = &BallPixel{col: 4, row: 7, x: -3.4781748801469816, y: -3.2997538248697915, z: 1.43793959915638, lat: -41.25, lon: 22.5}
	pixels[580] = &BallPixel{col: 4, row: 6, x: -3.0500165969133386, y: -3.762969970703125, z: 1.2609313199917478, lat: -48.75, lon: 22.5}
	pixels[581] = &BallPixel{col: 4, row: 5, x: -2.5687522441148762, y: -4.160919189453125, z: 1.0619680434465408, lat: -56.25, lon: 22.5}
	pixels[582] = &BallPixel{col: 4, row: 4, x: -2.0428065806627274, y: -4.487091064453125, z: 0.8445327152808506, lat: -63.75, lon: 22.5}
	pixels[583] = &BallPixel{col: 4, row: 3, x: -1.4818079024553308, y: -4.736277262369791, z: 0.6126058449347817, lat: -71.25, lon: 22.5}
	pixels[584] = &BallPixel{col: 4, row: 2, x: -0.8965880423784258, y: -4.904571533203126, z: 0.37066550552845, lat: -78.75, lon: 22.5}
	pixels[585] = &BallPixel{col: 4, row: 1, x: -0.2991823703050611, y: -4.989369710286458, z: 0.12368733386198667, lat: -86.25, lon: 22.5}
	pixels[600] = &BallPixel{col: 5, row: 17, x: -3.672081341792365, y: 2.779083251953125, z: 1.9608830081415376, lat: 33.75, lon: 28.125}
	pixels[601] = &BallPixel{col: 5, row: 16, x: -3.959933521051427, y: 2.2100728352864585, z: 2.114595411170741, lat: 26.25, lon: 28.125}
	pixels[602] = &BallPixel{col: 5, row: 15, x: -4.179844542231877, y: 1.6031392415364585, z: 2.2320273917284803, lat: 18.75, lon: 28.125}
	pixels[603] = &BallPixel{col: 5, row: 14, x: -4.328367073845583, y: 0.970001220703125, z: 2.3113380827126093, lat: 11.25, lon: 28.125}
	pixels[604] = &BallPixel{col: 5, row: 13, x: -4.403202894900461, y: 0.32367960611979163, z: 2.3513002393883666, lat: 3.75, lon: 28.125}
	pixels[605] = &BallPixel{col: 5, row: 12, x: -4.403202894900458, y: -0.32367960611979163, z: 2.3513002393883653, lat: -3.75, lon: 28.125}
	pixels[606] = &BallPixel{col: 5, row: 11, x: -4.328367073845584, y: -0.970001220703125, z: 2.3113380827126098, lat: -11.25, lon: 28.125}
	pixels[607] = &BallPixel{col: 5, row: 10, x: -4.179844542231875, y: -1.6031392415364585, z: 2.232027391728479, lat: -18.75, lon: 28.125}
	pixels[608] = &BallPixel{col: 5, row: 9, x: -3.9599335210514246, y: -2.2100728352864585, z: 2.1145954111707397, lat: -26.25, lon: 28.125}
	pixels[609] = &BallPixel{col: 5, row: 8, x: -3.672081341792363, y: -2.779083251953125, z: 1.9608830081415363, lat: -33.75, lon: 28.125}
	pixels[610] = &BallPixel{col: 5, row: 7, x: -3.320884446438867, y: -3.2997538248697915, z: 1.7733446721103976, lat: -41.25, lon: 28.125}
	pixels[611] = &BallPixel{col: 5, row: 6, x: -2.9120883874711576, y: -3.762969970703125, z: 1.5550485149142335, lat: -48.75, lon: 28.125}
	pixels[612] = &BallPixel{col: 5, row: 5, x: -2.4525878278654996, y: -4.160919189453125, z: 1.3096762707573362, lat: -56.25, lon: 28.125}
	pixels[613] = &BallPixel{col: 6, row: 3, x: -1.334106566694877, y: -4.736277262369791, z: 0.8910514833405617, lat: -71.25, lon: 33.75}
	pixels[614] = &BallPixel{col: 6, row: 4, x: -1.839186894086501, y: -4.487091064453125, z: 1.2283952804282305, lat: -63.75, lon: 33.75}
	pixels[615] = &BallPixel{col: 6, row: 5, x: -2.3127081664279117, y: -4.160919189453125, z: 1.5446607442572713, lat: -56.25, lon: 33.75}
	pixels[616] = &BallPixel{col: 6, row: 6, x: -2.7460018020744146, y: -3.762969970703125, z: 1.8340581180527809, lat: -48.75, lon: 33.75}
	pixels[617] = &BallPixel{col: 6, row: 7, x: -3.131482792086902, y: -3.2997538248697915, z: 2.0915213646367197, lat: -41.25, lon: 33.75}
	pixels[618] = &BallPixel{col: 6, row: 8, x: -3.4626497002318524, y: -2.779083251953125, z: 2.3127081664279103, lat: -33.75, lon: 33.75}
	pixels[619] = &BallPixel{col: 6, row: 9, x: -3.734084662981334, y: -2.2100728352864585, z: 2.49399992544204, lat: -26.25, lon: 33.75}
	pixels[620] = &BallPixel{col: 6, row: 10, x: -3.941453389512998, y: -1.6031392415364585, z: 2.6325017632916565, lat: -18.75, lon: 33.75}
	pixels[621] = &BallPixel{col: 6, row: 11, x: -4.081505161710087, y: -0.970001220703125, z: 2.7260425211861734, lat: -11.25, lon: 33.75}
	pixels[622] = &BallPixel{col: 6, row: 12, x: -4.152072834161423, y: -0.32367960611979163, z: 2.7731747599318624, lat: -3.75, lon: 33.75}
	pixels[623] = &BallPixel{col: 6, row: 13, x: -4.152072834161426, y: 0.32367960611979163, z: 2.7731747599318637, lat: 3.75, lon: 33.75}
	pixels[624] = &BallPixel{col: 6, row: 14, x: -4.081505161710086, y: 0.970001220703125, z: 2.726042521186173, lat: 11.25, lon: 33.75}
	pixels[625] = &BallPixel{col: 6, row: 15, x: -3.941453389513, y: 1.6031392415364585, z: 2.6325017632916583, lat: 18.75, lon: 33.75}
	pixels[626] = &BallPixel{col: 6, row: 16, x: -3.7340846629813362, y: 2.2100728352864585, z: 2.4939999254420413, lat: 26.25, lon: 33.75}
	pixels[627] = &BallPixel{col: 6, row: 17, x: -3.4626497002318546, y: 2.779083251953125, z: 2.3127081664279117, lat: 33.75, lon: 33.75}
	pixels[628] = &BallPixel{col: 6, row: 18, x: -3.131482792086902, y: 3.2997538248697915, z: 2.0915213646367197, lat: 41.25, lon: 33.75}
	pixels[629] = &BallPixel{col: 6, row: 19, x: -2.746001802074417, y: 3.762969970703125, z: 1.8340581180527822, lat: 48.75, lon: 33.75}
	pixels[630] = &BallPixel{col: 7, row: 17, x: -3.2195966176805126, y: 2.779083251953125, z: 2.641883057367524, lat: 33.75, lon: 39.375}
	pixels[631] = &BallPixel{col: 7, row: 16, x: -3.471978800010406, y: 2.2100728352864585, z: 2.8489786319551076, lat: 26.25, lon: 39.375}
	pixels[632] = &BallPixel{col: 7, row: 15, x: -3.6647917347145307, y: 1.6031392415364585, z: 3.007193863838136, lat: 18.75, lon: 39.375}
	pixels[633] = &BallPixel{col: 7, row: 14, x: -3.7950128806871386, y: 0.970001220703125, z: 3.1140485664946027, lat: 11.25, lon: 39.375}
	pixels[634] = &BallPixel{col: 7, row: 13, x: -3.8606272105244033, y: 0.32367960611979163, z: 3.1678892822431726, lat: 3.75, lon: 39.375}
	pixels[635] = &BallPixel{col: 7, row: 12, x: -3.860627210524401, y: -0.32367960611979163, z: 3.167889282243171, lat: -3.75, lon: 39.375}
	pixels[636] = &BallPixel{col: 7, row: 11, x: -3.7950128806871395, y: -0.970001220703125, z: 3.114048566494603, lat: -11.25, lon: 39.375}
	pixels[637] = &BallPixel{col: 7, row: 10, x: -3.6647917347145285, y: -1.6031392415364585, z: 3.0071938638381344, lat: -18.75, lon: 39.375}
	pixels[638] = &BallPixel{col: 7, row: 9, x: -3.471978800010404, y: -2.2100728352864585, z: 2.848978631955106, lat: -26.25, lon: 39.375}
	pixels[639] = &BallPixel{col: 7, row: 8, x: -3.219596617680511, y: -2.779083251953125, z: 2.6418830573675223, lat: -33.75, lon: 39.375}
	pixels[640] = &BallPixel{col: 7, row: 7, x: -2.9116752425325125, y: -3.2997538248697915, z: 2.3892140554380608, lat: -41.25, lon: 39.375}
	pixels[641] = &BallPixel{col: 7, row: 6, x: -2.5532522430759874, y: -3.762969970703125, z: 2.095105270370065, lat: -48.75, lon: 39.375}
	pixels[642] = &BallPixel{col: 7, row: 5, x: -2.15037270152243, y: -4.160919189453125, z: 1.76451707520755, lat: -56.25, lon: 39.375}
	pixels[643] = &BallPixel{col: 9, row: 5, x: -1.7645170752075487, y: -4.160919189453125, z: 2.1503727015224285, lat: -56.25, lon: 50.625}
	pixels[644] = &BallPixel{col: 9, row: 6, x: -2.0951052703700634, y: -3.762969970703125, z: 2.5532522430759856, lat: -48.75, lon: 50.625}
	pixels[645] = &BallPixel{col: 9, row: 7, x: -2.389214055438059, y: -3.2997538248697915, z: 2.9116752425325103, lat: -41.25, lon: 50.625}
	pixels[646] = &BallPixel{col: 9, row: 8, x: -2.6418830573675205, y: -2.779083251953125, z: 3.2195966176805086, lat: -33.75, lon: 50.625}
	pixels[647] = &BallPixel{col: 9, row: 9, x: -2.848978631955104, y: -2.2100728352864585, z: 3.4719788000104015, lat: -26.25, lon: 50.625}
	pixels[648] = &BallPixel{col: 9, row: 10, x: -3.0071938638381326, y: -1.6031392415364585, z: 3.664791734714526, lat: -18.75, lon: 50.625}
	pixels[649] = &BallPixel{col: 9, row: 11, x: -3.114048566494601, y: -0.970001220703125, z: 3.795012880687137, lat: -11.25, lon: 50.625}
	pixels[650] = &BallPixel{col: 9, row: 12, x: -3.1678892822431686, y: -0.32367960611979163, z: 3.8606272105243984, lat: -3.75, lon: 50.625}
	pixels[651] = &BallPixel{col: 9, row: 13, x: -3.1678892822431703, y: 0.32367960611979163, z: 3.8606272105244006, lat: 3.75, lon: 50.625}
	pixels[652] = &BallPixel{col: 9, row: 14, x: -3.1140485664946005, y: 0.970001220703125, z: 3.795012880687136, lat: 11.25, lon: 50.625}
	pixels[653] = &BallPixel{col: 9, row: 15, x: -3.0071938638381344, y: 1.6031392415364585, z: 3.664791734714528, lat: 18.75, lon: 50.625}
	pixels[654] = &BallPixel{col: 9, row: 16, x: -2.8489786319551054, y: 2.2100728352864585, z: 3.4719788000104037, lat: 26.25, lon: 50.625}
	pixels[655] = &BallPixel{col: 9, row: 17, x: -2.6418830573675223, y: 2.779083251953125, z: 3.2195966176805104, lat: 33.75, lon: 50.625}
	pixels[656] = &BallPixel{col: 8, row: 19, x: -2.3356070041656514, y: 3.762969970703125, z: 2.3356070041656514, lat: 48.75, lon: 45.0}
	pixels[657] = &BallPixel{col: 8, row: 18, x: -2.663477182388306, y: 3.2997538248697915, z: 2.663477182388306, lat: 41.25, lon: 45.0}
	pixels[658] = &BallPixel{col: 8, row: 17, x: -2.945150613784792, y: 2.779083251953125, z: 2.945150613784792, lat: 33.75, lon: 45.0}
	pixels[659] = &BallPixel{col: 8, row: 16, x: -3.176019144058229, y: 2.2100728352864585, z: 3.176019144058229, lat: 26.25, lon: 45.0}
	pixels[660] = &BallPixel{col: 8, row: 15, x: -3.3523962497711195, y: 1.6031392415364585, z: 3.3523962497711195, lat: 18.75, lon: 45.0}
	pixels[661] = &BallPixel{col: 8, row: 14, x: -3.4715170383453366, y: 0.970001220703125, z: 3.4715170383453366, lat: 11.25, lon: 45.0}
	pixels[662] = &BallPixel{col: 8, row: 13, x: -3.531538248062135, y: 0.32367960611979163, z: 3.531538248062135, lat: 3.75, lon: 45.0}
	pixels[663] = &BallPixel{col: 8, row: 12, x: -3.5315382480621333, y: -0.32367960611979163, z: 3.5315382480621333, lat: -3.75, lon: 45.0}
	pixels[664] = &BallPixel{col: 8, row: 11, x: -3.4715170383453375, y: -0.970001220703125, z: 3.4715170383453375, lat: -11.25, lon: 45.0}
	pixels[665] = &BallPixel{col: 8, row: 10, x: -3.3523962497711177, y: -1.6031392415364585, z: 3.3523962497711177, lat: -18.75, lon: 45.0}
	pixels[666] = &BallPixel{col: 8, row: 9, x: -3.1760191440582273, y: -2.2100728352864585, z: 3.1760191440582273, lat: -26.25, lon: 45.0}
	pixels[667] = &BallPixel{col: 8, row: 8, x: -2.94515061378479, y: -2.779083251953125, z: 2.94515061378479, lat: -33.75, lon: 45.0}
	pixels[668] = &BallPixel{col: 8, row: 7, x: -2.6634771823883057, y: -3.2997538248697915, z: 2.6634771823883057, lat: -41.25, lon: 45.0}
	pixels[669] = &BallPixel{col: 8, row: 6, x: -2.3356070041656496, y: -3.762969970703125, z: 2.3356070041656496, lat: -48.75, lon: 45.0}
	pixels[670] = &BallPixel{col: 8, row: 5, x: -1.967069864273071, y: -4.160919189453125, z: 1.967069864273071, lat: -56.25, lon: 45.0}
	pixels[671] = &BallPixel{col: 8, row: 4, x: -1.564317178726196, y: -4.487091064453125, z: 1.564317178726196, lat: -63.75, lon: 45.0}
	pixels[672] = &BallPixel{col: 8, row: 3, x: -1.1347219944000249, y: -4.736277262369791, z: 1.1347219944000249, lat: -71.25, lon: 45.0}
	pixels[673] = &BallPixel{col: 8, row: 2, x: -0.6865789890289307, y: -4.904571533203126, z: 0.6865789890289307, lat: -78.75, lon: 45.0}
	pixels[674] = &BallPixel{col: 8, row: 1, x: -0.2291044712066648, y: -4.989369710286458, z: 0.2291044712066648, lat: -86.25, lon: 45.0}
	pixels[675] = &BallPixel{col: 8, row: 0, x: 0.22910447120666555, y: -4.989369710286458, z: -0.22910447120666555, lat: -93.75, lon: 45.0}
	pixels[676] = &BallPixel{col: 40, row: 0, x: -0.22910447120666555, y: -4.989369710286458, z: 0.22910447120666555, lat: -93.75, lon: -135.0}
	pixels[677] = &BallPixel{col: 40, row: 1, x: 0.2291044712066648, y: -4.989369710286458, z: -0.2291044712066648, lat: -86.25, lon: -135.0}
	pixels[678] = &BallPixel{col: 40, row: 2, x: 0.6865789890289307, y: -4.904571533203126, z: -0.6865789890289307, lat: -78.75, lon: -135.0}
	pixels[679] = &BallPixel{col: 40, row: 3, x: 1.1347219944000249, y: -4.736277262369791, z: -1.1347219944000249, lat: -71.25, lon: -135.0}
	pixels[680] = &BallPixel{col: 40, row: 4, x: 1.564317178726196, y: -4.487091064453125, z: -1.564317178726196, lat: -63.75, lon: -135.0}
	pixels[681] = &BallPixel{col: 40, row: 5, x: 1.967069864273071, y: -4.160919189453125, z: -1.967069864273071, lat: -56.25, lon: -135.0}
	pixels[682] = &BallPixel{col: 40, row: 6, x: 2.3356070041656496, y: -3.762969970703125, z: -2.3356070041656496, lat: -48.75, lon: -135.0}
	pixels[683] = &BallPixel{col: 40, row: 7, x: 2.6634771823883057, y: -3.2997538248697915, z: -2.6634771823883057, lat: -41.25, lon: -135.0}
	pixels[684] = &BallPixel{col: 40, row: 8, x: 2.94515061378479, y: -2.779083251953125, z: -2.94515061378479, lat: -33.75, lon: -135.0}
	pixels[685] = &BallPixel{col: 40, row: 9, x: 3.1760191440582273, y: -2.2100728352864585, z: -3.1760191440582273, lat: -26.25, lon: -135.0}
	pixels[686] = &BallPixel{col: 40, row: 10, x: 3.3523962497711177, y: -1.6031392415364585, z: -3.3523962497711177, lat: -18.75, lon: -135.0}
	pixels[687] = &BallPixel{col: 40, row: 11, x: 3.4715170383453375, y: -0.970001220703125, z: -3.4715170383453375, lat: -11.25, lon: -135.0}
	pixels[688] = &BallPixel{col: 40, row: 12, x: 3.5315382480621333, y: -0.32367960611979163, z: -3.5315382480621333, lat: -3.75, lon: -135.0}
	pixels[689] = &BallPixel{col: 40, row: 13, x: 3.531538248062135, y: 0.32367960611979163, z: -3.531538248062135, lat: 3.75, lon: -135.0}
	pixels[690] = &BallPixel{col: 40, row: 14, x: 3.4715170383453366, y: 0.970001220703125, z: -3.4715170383453366, lat: 11.25, lon: -135.0}
	pixels[691] = &BallPixel{col: 40, row: 15, x: 3.3523962497711195, y: 1.6031392415364585, z: -3.3523962497711195, lat: 18.75, lon: -135.0}
	pixels[692] = &BallPixel{col: 40, row: 16, x: 3.176019144058229, y: 2.2100728352864585, z: -3.176019144058229, lat: 26.25, lon: -135.0}
	pixels[693] = &BallPixel{col: 40, row: 17, x: 2.945150613784792, y: 2.779083251953125, z: -2.945150613784792, lat: 33.75, lon: -135.0}
	pixels[694] = &BallPixel{col: 40, row: 18, x: 2.663477182388306, y: 3.2997538248697915, z: -2.663477182388306, lat: 41.25, lon: -135.0}
	pixels[695] = &BallPixel{col: 40, row: 19, x: 2.3356070041656514, y: 3.762969970703125, z: -2.3356070041656514, lat: 48.75, lon: -135.0}
	pixels[696] = &BallPixel{col: 39, row: 17, x: 3.2195966176805118, y: 2.779083251953125, z: -2.6418830573675223, lat: 33.75, lon: -140.625}
	pixels[697] = &BallPixel{col: 39, row: 16, x: 3.471978800010405, y: 2.2100728352864585, z: -2.8489786319551054, lat: 26.25, lon: -140.625}
	pixels[698] = &BallPixel{col: 39, row: 15, x: 3.66479173471453, y: 1.6031392415364585, z: -3.0071938638381344, lat: 18.75, lon: -140.625}
	pixels[699] = &BallPixel{col: 39, row: 14, x: 3.7950128806871377, y: 0.970001220703125, z: -3.1140485664946005, lat: 11.25, lon: -140.625}
	pixels[700] = &BallPixel{col: 39, row: 13, x: 3.860627210524402, y: 0.32367960611979163, z: -3.1678892822431703, lat: 3.75, lon: -140.625}
	pixels[701] = &BallPixel{col: 39, row: 12, x: 3.8606272105244, y: -0.32367960611979163, z: -3.1678892822431686, lat: -3.75, lon: -140.625}
	pixels[702] = &BallPixel{col: 39, row: 11, x: 3.795012880687138, y: -0.970001220703125, z: -3.114048566494601, lat: -11.25, lon: -140.625}
	pixels[703] = &BallPixel{col: 39, row: 10, x: 3.6647917347145276, y: -1.6031392415364585, z: -3.0071938638381326, lat: -18.75, lon: -140.625}
	pixels[704] = &BallPixel{col: 39, row: 9, x: 3.4719788000104033, y: -2.2100728352864585, z: -2.848978631955104, lat: -26.25, lon: -140.625}
	pixels[705] = &BallPixel{col: 39, row: 8, x: 3.21959661768051, y: -2.779083251953125, z: -2.6418830573675205, lat: -33.75, lon: -140.625}
	pixels[706] = &BallPixel{col: 39, row: 7, x: 2.9116752425325116, y: -3.2997538248697915, z: -2.389214055438059, lat: -41.25, lon: -140.625}
	pixels[707] = &BallPixel{col: 39, row: 6, x: 2.5532522430759865, y: -3.762969970703125, z: -2.0951052703700634, lat: -48.75, lon: -140.625}
	pixels[708] = &BallPixel{col: 39, row: 5, x: 2.1503727015224294, y: -4.160919189453125, z: -1.7645170752075487, lat: -56.25, lon: -140.625}
	pixels[709] = &BallPixel{col: 37, row: 5, x: 2.4525878278655004, y: -4.160919189453125, z: -1.3096762707573375, lat: -56.25, lon: -151.875}
	pixels[710] = &BallPixel{col: 37, row: 6, x: 2.9120883874711585, y: -3.762969970703125, z: -1.5550485149142348, lat: -48.75, lon: -151.875}
	pixels[711] = &BallPixel{col: 37, row: 7, x: 3.3208844464388685, y: -3.2997538248697915, z: -1.7733446721103991, lat: -41.25, lon: -151.875}
	pixels[712] = &BallPixel{col: 37, row: 8, x: 3.672081341792364, y: -2.779083251953125, z: -1.960883008141538, lat: -33.75, lon: -151.875}
	pixels[713] = &BallPixel{col: 37, row: 9, x: 3.959933521051426, y: -2.2100728352864585, z: -2.1145954111707415, lat: -26.25, lon: -151.875}
	pixels[714] = &BallPixel{col: 37, row: 10, x: 4.179844542231876, y: -1.6031392415364585, z: -2.232027391728481, lat: -18.75, lon: -151.875}
	pixels[715] = &BallPixel{col: 37, row: 11, x: 4.328367073845585, y: -0.970001220703125, z: -2.311338082712612, lat: -11.25, lon: -151.875}
	pixels[716] = &BallPixel{col: 37, row: 12, x: 4.40320289490046, y: -0.32367960611979163, z: -2.3513002393883675, lat: -3.75, lon: -151.875}
	pixels[717] = &BallPixel{col: 37, row: 13, x: 4.4032028949004625, y: 0.32367960611979163, z: -2.351300239388369, lat: 3.75, lon: -151.875}
	pixels[718] = &BallPixel{col: 37, row: 14, x: 4.328367073845585, y: 0.970001220703125, z: -2.3113380827126115, lat: 11.25, lon: -151.875}
	pixels[719] = &BallPixel{col: 37, row: 15, x: 4.179844542231879, y: 1.6031392415364585, z: -2.2320273917284825, lat: 18.75, lon: -151.875}
	pixels[720] = &BallPixel{col: 37, row: 16, x: 3.959933521051428, y: 2.2100728352864585, z: -2.114595411170743, lat: 26.25, lon: -151.875}
	pixels[721] = &BallPixel{col: 37, row: 17, x: 3.6720813417923663, y: 2.779083251953125, z: -1.9608830081415394, lat: 33.75, lon: -151.875}
	pixels[722] = &BallPixel{col: 36, row: 19, x: 3.0500165969133404, y: 3.762969970703125, z: -1.26093131999175, lat: 48.75, lon: -157.5}
	pixels[723] = &BallPixel{col: 36, row: 18, x: 3.478174880146981, y: 3.2997538248697915, z: -1.4379395991563815, lat: 41.25, lon: -157.5}
	pixels[724] = &BallPixel{col: 36, row: 17, x: 3.846005871891978, y: 2.779083251953125, z: -1.5900074988603619, lat: 33.75, lon: -157.5}
	pixels[725] = &BallPixel{col: 36, row: 16, x: 4.147491887211802, y: 2.2100728352864585, z: -1.7146472007036238, lat: 26.25, lon: -157.5}
	pixels[726] = &BallPixel{col: 36, row: 15, x: 4.377818778157236, y: 1.6031392415364585, z: -1.8098684499661155, lat: 18.75, lon: -157.5}
	pixels[727] = &BallPixel{col: 36, row: 14, x: 4.533375933766365, y: 0.970001220703125, z: -1.8741785556077977, lat: 11.25, lon: -157.5}
	pixels[728] = &BallPixel{col: 36, row: 13, x: 4.611756280064585, y: 0.32367960611979163, z: -1.9065823902686465, lat: 3.75, lon: -157.5}
	pixels[729] = &BallPixel{col: 36, row: 12, x: 4.611756280064583, y: -0.32367960611979163, z: -1.9065823902686456, lat: -3.75, lon: -157.5}
	pixels[730] = &BallPixel{col: 36, row: 11, x: 4.533375933766366, y: -0.970001220703125, z: -1.8741785556077981, lat: -11.25, lon: -157.5}
	pixels[731] = &BallPixel{col: 36, row: 10, x: 4.377818778157233, y: -1.6031392415364585, z: -1.8098684499661144, lat: -18.75, lon: -157.5}
	pixels[732] = &BallPixel{col: 36, row: 9, x: 4.1474918872118, y: -2.2100728352864585, z: -1.7146472007036226, lat: -26.25, lon: -157.5}
	pixels[733] = &BallPixel{col: 36, row: 8, x: 3.8460058718919754, y: -2.779083251953125, z: -1.5900074988603607, lat: -33.75, lon: -157.5}
	pixels[734] = &BallPixel{col: 36, row: 7, x: 3.4781748801469807, y: -3.2997538248697915, z: -1.4379395991563815, lat: -41.25, lon: -157.5}
	pixels[735] = &BallPixel{col: 36, row: 6, x: 3.050016596913338, y: -3.762969970703125, z: -1.260931319991749, lat: -48.75, lon: -157.5}
	pixels[736] = &BallPixel{col: 36, row: 5, x: 2.568752244114876, y: -4.160919189453125, z: -1.061968043446542, lat: -56.25, lon: -157.5}
	pixels[737] = &BallPixel{col: 36, row: 4, x: 2.042806580662727, y: -4.487091064453125, z: -0.8445327152808515, lat: -63.75, lon: -157.5}
	pixels[738] = &BallPixel{col: 36, row: 3, x: 1.4818079024553303, y: -4.736277262369791, z: -0.6126058449347822, lat: -71.25, lon: -157.5}
	pixels[739] = &BallPixel{col: 36, row: 2, x: 0.8965880423784256, y: -4.904571533203126, z: -0.3706655055284504, lat: -78.75, lon: -157.5}
	pixels[740] = &BallPixel{col: 36, row: 1, x: 0.29918237030506106, y: -4.989369710286458, z: -0.12368733386198681, lat: -86.25, lon: -157.5}
	pixels[750] = &BallPixel{col: 35, row: 17, x: 3.9828804692369917, y: 2.779083251953125, z: -1.2040244041127162, lat: 33.75, lon: -163.125}
	pixels[751] = &BallPixel{col: 35, row: 16, x: 4.295095999364279, y: 2.2100728352864585, z: -1.2984071305138067, lat: 26.25, lon: -163.125}
	pixels[752] = &BallPixel{col: 35, row: 15, x: 4.533619939795853, y: 1.6031392415364585, z: -1.3705128960427835, lat: 18.75, lon: -163.125}
	pixels[753] = &BallPixel{col: 35, row: 14, x: 4.694713182386478, y: 0.970001220703125, z: -1.4192113686469383, lat: 11.25, lon: -163.125}
	pixels[754] = &BallPixel{col: 35, row: 13, x: 4.775882988372666, y: 0.32367960611979163, z: -1.443748993624468, lat: 3.75, lon: -163.125}
	pixels[755] = &BallPixel{col: 35, row: 12, x: 4.775882988372663, y: -0.32367960611979163, z: -1.4437489936244674, lat: -3.75, lon: -163.125}
	pixels[756] = &BallPixel{col: 35, row: 11, x: 4.6947131823864785, y: -0.970001220703125, z: -1.4192113686469388, lat: -11.25, lon: -163.125}
	pixels[757] = &BallPixel{col: 35, row: 10, x: 4.533619939795851, y: -1.6031392415364585, z: -1.3705128960427826, lat: -18.75, lon: -163.125}
	pixels[758] = &BallPixel{col: 35, row: 9, x: 4.2950959993642766, y: -2.2100728352864585, z: -1.2984071305138059, lat: -26.25, lon: -163.125}
	pixels[759] = &BallPixel{col: 35, row: 8, x: 3.9828804692369895, y: -2.779083251953125, z: -1.2040244041127153, lat: -33.75, lon: -163.125}
	pixels[760] = &BallPixel{col: 35, row: 7, x: 3.6019588269409732, y: -3.2997538248697915, z: -1.0888718262431214, lat: -41.25, lon: -163.125}
	pixels[761] = &BallPixel{col: 35, row: 6, x: 3.158562919384956, y: -3.762969970703125, z: -0.9548332836595366, lat: -48.75, lon: -163.125}
	pixels[762] = &BallPixel{col: 35, row: 5, x: 2.6601709628594112, y: -4.160919189453125, z: -0.8041694404673763, lat: -56.25, lon: -163.125}
	pixels[763] = &BallPixel{col: 34, row: 3, x: 1.572542217560113, y: -4.736277262369791, z: -0.3110094042494907, lat: -71.25, lon: -168.75}
	pixels[764] = &BallPixel{col: 34, row: 4, x: 2.1678920628502962, y: -4.487091064453125, z: -0.4287546696141379, lat: -63.75, lon: -168.75}
	pixels[765] = &BallPixel{col: 34, row: 5, x: 2.726042521186173, y: -4.160919189453125, z: -0.5391428293660304, lat: -56.25, lon: -168.75}
	pixels[766] = &BallPixel{col: 34, row: 6, x: 3.236775735206902, y: -3.762969970703125, z: -0.6401530476287034, lat: -48.75, lon: -168.75}
	pixels[767] = &BallPixel{col: 34, row: 7, x: 3.6911510797217493, y: -3.2997538248697915, z: -0.7300170930102498, lat: -41.25, lon: -168.75}
	pixels[768] = &BallPixel{col: 34, row: 8, x: 4.0815051617100835, y: -2.779083251953125, z: -0.807219338603321, lat: -33.75, lon: -168.75}
	pixels[769] = &BallPixel{col: 34, row: 9, x: 4.401451820321381, y: -2.2100728352864585, z: -0.8704967619851266, lat: -26.25, lon: -168.75}
	pixels[770] = &BallPixel{col: 34, row: 10, x: 4.645882126875221, y: -1.6031392415364585, z: -0.9188389452174341, lat: -18.75, lon: -168.75}
	pixels[771] = &BallPixel{col: 34, row: 11, x: 4.8109643848612915, y: -0.970001220703125, z: -0.9514880748465697, lat: -11.25, lon: -168.75}
	pixels[772] = &BallPixel{col: 34, row: 12, x: 4.894144129939376, y: -0.32367960611979163, z: -0.9679389419034161, lat: -3.75, lon: -168.75}
	pixels[773] = &BallPixel{col: 34, row: 13, x: 4.894144129939379, y: 0.32367960611979163, z: -0.9679389419034167, lat: 3.75, lon: -168.75}
	pixels[774] = &BallPixel{col: 34, row: 14, x: 4.810964384861291, y: 0.970001220703125, z: -0.9514880748465695, lat: 11.25, lon: -168.75}
	pixels[775] = &BallPixel{col: 34, row: 15, x: 4.6458821268752235, y: 1.6031392415364585, z: -0.9188389452174347, lat: 18.75, lon: -168.75}
	pixels[776] = &BallPixel{col: 34, row: 16, x: 4.401451820321384, y: 2.2100728352864585, z: -0.8704967619851272, lat: 26.25, lon: -168.75}
	pixels[777] = &BallPixel{col: 34, row: 17, x: 4.081505161710086, y: 2.779083251953125, z: -0.8072193386033215, lat: 33.75, lon: -168.75}
	pixels[778] = &BallPixel{col: 34, row: 18, x: 3.6911510797217497, y: 3.2997538248697915, z: -0.7300170930102499, lat: 41.25, lon: -168.75}
	pixels[779] = &BallPixel{col: 34, row: 19, x: 3.236775735206905, y: 3.762969970703125, z: -0.6401530476287038, lat: 48.75, lon: -168.75}
	pixels[780] = &BallPixel{col: 33, row: 17, x: 4.141022826370321, y: 2.779083251953125, z: -0.40422076621325864, lat: 33.75, lon: -174.375}
	pixels[781] = &BallPixel{col: 33, row: 16, x: 4.465635037806354, y: 2.2100728352864585, z: -0.4359073814121094, lat: 26.25, lon: -174.375}
	pixels[782] = &BallPixel{col: 33, row: 15, x: 4.713629696343563, y: 1.6031392415364585, z: -0.46011506996971435, lat: 18.75, lon: -174.375}
	pixels[783] = &BallPixel{col: 33, row: 14, x: 4.881119230587502, y: 0.970001220703125, z: -0.47646435146453037, lat: 11.25, lon: -174.375}
	pixels[784] = &BallPixel{col: 33, row: 13, x: 4.965511926275216, y: 0.32367960611979163, z: -0.48470223894886255, lat: 3.75, lon: -174.375}
	pixels[785] = &BallPixel{col: 33, row: 12, x: 4.965511926275213, y: -0.32367960611979163, z: -0.4847022389488623, lat: -3.75, lon: -174.375}
	pixels[786] = &BallPixel{col: 33, row: 11, x: 4.881119230587503, y: -0.970001220703125, z: -0.4764643514645304, lat: -11.25, lon: -174.375}
	pixels[787] = &BallPixel{col: 33, row: 10, x: 4.713629696343561, y: -1.6031392415364585, z: -0.4601150699697141, lat: -18.75, lon: -174.375}
	pixels[788] = &BallPixel{col: 33, row: 9, x: 4.465635037806352, y: -2.2100728352864585, z: -0.4359073814121091, lat: -26.25, lon: -174.375}
	pixels[789] = &BallPixel{col: 33, row: 8, x: 4.141022826370318, y: -2.779083251953125, z: -0.40422076621325836, lat: -33.75, lon: -174.375}
	pixels[790] = &BallPixel{col: 33, row: 7, x: 3.744976490561386, y: -3.2997538248697915, z: -0.36556119826855277, lat: -41.25, lon: -174.375}
	pixels[791] = &BallPixel{col: 33, row: 6, x: 3.283975316036959, y: -3.762969970703125, z: -0.3205611449472308, lat: -48.75, lon: -174.375}
	pixels[792] = &BallPixel{col: 33, row: 5, x: 2.7657944455859256, y: -4.160919189453125, z: -0.2699795670923787, lat: -56.25, lon: -174.375}
	pixels[793] = &BallPixel{col: 31, row: 5, x: 2.765794445585925, y: -4.160919189453125, z: 0.2699795670923787, lat: -56.25, lon: 174.375}
	pixels[794] = &BallPixel{col: 31, row: 6, x: 3.2839753160369587, y: -3.762969970703125, z: 0.3205611449472308, lat: -48.75, lon: 174.375}
	pixels[795] = &BallPixel{col: 31, row: 7, x: 3.744976490561385, y: -3.2997538248697915, z: 0.36556119826855277, lat: -41.25, lon: 174.375}
	pixels[796] = &BallPixel{col: 31, row: 8, x: 4.1410228263703175, y: -2.779083251953125, z: 0.40422076621325836, lat: -33.75, lon: 174.375}
	pixels[797] = &BallPixel{col: 31, row: 9, x: 4.465635037806351, y: -2.2100728352864585, z: 0.4359073814121091, lat: -26.25, lon: 174.375}
	pixels[798] = &BallPixel{col: 31, row: 10, x: 4.713629696343559, y: -1.6031392415364585, z: 0.4601150699697141, lat: -18.75, lon: 174.375}
	pixels[799] = &BallPixel{col: 31, row: 11, x: 4.881119230587502, y: -0.970001220703125, z: 0.4764643514645304, lat: -11.25, lon: 174.375}
	pixels[800] = &BallPixel{col: 31, row: 12, x: 4.965511926275212, y: -0.32367960611979163, z: 0.4847022389488623, lat: -3.75, lon: 174.375}
	pixels[801] = &BallPixel{col: 31, row: 13, x: 4.965511926275215, y: 0.32367960611979163, z: 0.48470223894886255, lat: 3.75, lon: 174.375}
	pixels[802] = &BallPixel{col: 31, row: 14, x: 4.881119230587501, y: 0.970001220703125, z: 0.47646435146453037, lat: 11.25, lon: 174.375}
	pixels[803] = &BallPixel{col: 31, row: 15, x: 4.7136296963435615, y: 1.6031392415364585, z: 0.46011506996971435, lat: 18.75, lon: 174.375}
	pixels[804] = &BallPixel{col: 31, row: 16, x: 4.465635037806353, y: 2.2100728352864585, z: 0.4359073814121094, lat: 26.25, lon: 174.375}
	pixels[805] = &BallPixel{col: 31, row: 17, x: 4.14102282637032, y: 2.779083251953125, z: 0.40422076621325864, lat: 33.75, lon: 174.375}
	pixels[806] = &BallPixel{col: 30, row: 19, x: 3.2367757352069058, y: 3.762969970703125, z: 0.6401530476287038, lat: 48.75, lon: 168.75}
	pixels[807] = &BallPixel{col: 30, row: 18, x: 3.6911510797217506, y: 3.2997538248697915, z: 0.7300170930102499, lat: 41.25, lon: 168.75}
	pixels[808] = &BallPixel{col: 30, row: 17, x: 4.081505161710087, y: 2.779083251953125, z: 0.8072193386033215, lat: 33.75, lon: 168.75}
	pixels[809] = &BallPixel{col: 30, row: 16, x: 4.401451820321385, y: 2.2100728352864585, z: 0.8704967619851272, lat: 26.25, lon: 168.75}
	pixels[810] = &BallPixel{col: 30, row: 15, x: 4.645882126875225, y: 1.6031392415364585, z: 0.9188389452174347, lat: 18.75, lon: 168.75}
	pixels[811] = &BallPixel{col: 30, row: 14, x: 4.8109643848612915, y: 0.970001220703125, z: 0.9514880748465695, lat: 11.25, lon: 168.75}
	pixels[812] = &BallPixel{col: 30, row: 13, x: 4.894144129939381, y: 0.32367960611979163, z: 0.9679389419034167, lat: 3.75, lon: 168.75}
	pixels[813] = &BallPixel{col: 30, row: 12, x: 4.894144129939378, y: -0.32367960611979163, z: 0.9679389419034161, lat: -3.75, lon: 168.75}
	pixels[814] = &BallPixel{col: 30, row: 11, x: 4.810964384861292, y: -0.970001220703125, z: 0.9514880748465697, lat: -11.25, lon: 168.75}
	pixels[815] = &BallPixel{col: 30, row: 10, x: 4.645882126875223, y: -1.6031392415364585, z: 0.9188389452174341, lat: -18.75, lon: 168.75}
	pixels[816] = &BallPixel{col: 30, row: 9, x: 4.401451820321382, y: -2.2100728352864585, z: 0.8704967619851266, lat: -26.25, lon: 168.75}
	pixels[817] = &BallPixel{col: 30, row: 8, x: 4.081505161710084, y: -2.779083251953125, z: 0.807219338603321, lat: -33.75, lon: 168.75}
	pixels[818] = &BallPixel{col: 30, row: 7, x: 3.69115107972175, y: -3.2997538248697915, z: 0.7300170930102498, lat: -41.25, lon: 168.75}
	pixels[819] = &BallPixel{col: 30, row: 6, x: 3.236775735206903, y: -3.762969970703125, z: 0.6401530476287034, lat: -48.75, lon: 168.75}
	pixels[820] = &BallPixel{col: 30, row: 5, x: 2.726042521186174, y: -4.160919189453125, z: 0.5391428293660304, lat: -56.25, lon: 168.75}
	pixels[821] = &BallPixel{col: 30, row: 4, x: 2.1678920628502967, y: -4.487091064453125, z: 0.4287546696141379, lat: -63.75, lon: 168.75}
	pixels[822] = &BallPixel{col: 30, row: 3, x: 1.5725422175601136, y: -4.736277262369791, z: 0.3110094042494907, lat: -71.25, lon: 168.75}
	pixels[823] = &BallPixel{col: 29, row: 5, x: 2.6601709628594117, y: -4.160919189453125, z: 0.8041694404673763, lat: -56.25, lon: 163.125}
	pixels[824] = &BallPixel{col: 29, row: 6, x: 3.1585629193849565, y: -3.762969970703125, z: 0.9548332836595366, lat: -48.75, lon: 163.125}
	pixels[825] = &BallPixel{col: 29, row: 7, x: 3.6019588269409737, y: -3.2997538248697915, z: 1.0888718262431214, lat: -41.25, lon: 163.125}
	pixels[826] = &BallPixel{col: 29, row: 8, x: 3.98288046923699, y: -2.779083251953125, z: 1.2040244041127153, lat: -33.75, lon: 163.125}
	pixels[827] = &BallPixel{col: 29, row: 9, x: 4.2950959993642766, y: -2.2100728352864585, z: 1.2984071305138059, lat: -26.25, lon: 163.125}
	pixels[828] = &BallPixel{col: 29, row: 10, x: 4.533619939795852, y: -1.6031392415364585, z: 1.3705128960427826, lat: -18.75, lon: 163.125}
	pixels[829] = &BallPixel{col: 29, row: 11, x: 4.6947131823864785, y: -0.970001220703125, z: 1.4192113686469388, lat: -11.25, lon: 163.125}
	pixels[830] = &BallPixel{col: 29, row: 12, x: 4.775882988372664, y: -0.32367960611979163, z: 1.4437489936244674, lat: -3.75, lon: 163.125}
	pixels[831] = &BallPixel{col: 29, row: 13, x: 4.775882988372667, y: 0.32367960611979163, z: 1.443748993624468, lat: 3.75, lon: 163.125}
	pixels[832] = &BallPixel{col: 29, row: 14, x: 4.694713182386478, y: 0.970001220703125, z: 1.4192113686469383, lat: 11.25, lon: 163.125}
	pixels[833] = &BallPixel{col: 29, row: 15, x: 4.533619939795853, y: 1.6031392415364585, z: 1.3705128960427835, lat: 18.75, lon: 163.125}
	pixels[834] = &BallPixel{col: 29, row: 16, x: 4.295095999364279, y: 2.2100728352864585, z: 1.2984071305138067, lat: 26.25, lon: 163.125}
	pixels[835] = &BallPixel{col: 29, row: 17, x: 3.982880469236992, y: 2.779083251953125, z: 1.2040244041127162, lat: 33.75, lon: 163.125}
	pixels[836] = &BallPixel{col: 28, row: 19, x: 3.0500165969133413, y: 3.762969970703125, z: 1.26093131999175, lat: 48.75, lon: 157.5}
	pixels[837] = &BallPixel{col: 28, row: 18, x: 3.478174880146982, y: 3.2997538248697915, z: 1.4379395991563815, lat: 41.25, lon: 157.5}
	pixels[838] = &BallPixel{col: 28, row: 17, x: 3.846005871891979, y: 2.779083251953125, z: 1.5900074988603619, lat: 33.75, lon: 157.5}
	pixels[839] = &BallPixel{col: 28, row: 16, x: 4.147491887211803, y: 2.2100728352864585, z: 1.7146472007036238, lat: 26.25, lon: 157.5}
	pixels[840] = &BallPixel{col: 28, row: 15, x: 4.377818778157237, y: 1.6031392415364585, z: 1.8098684499661155, lat: 18.75, lon: 157.5}
	pixels[841] = &BallPixel{col: 28, row: 14, x: 4.533375933766366, y: 0.970001220703125, z: 1.8741785556077977, lat: 11.25, lon: 157.5}
	pixels[842] = &BallPixel{col: 28, row: 13, x: 4.611756280064586, y: 0.32367960611979163, z: 1.9065823902686465, lat: 3.75, lon: 157.5}
	pixels[843] = &BallPixel{col: 28, row: 12, x: 4.611756280064584, y: -0.32367960611979163, z: 1.9065823902686456, lat: -3.75, lon: 157.5}
	pixels[844] = &BallPixel{col: 28, row: 11, x: 4.533375933766367, y: -0.970001220703125, z: 1.8741785556077981, lat: -11.25, lon: 157.5}
	pixels[845] = &BallPixel{col: 28, row: 10, x: 4.377818778157235, y: -1.6031392415364585, z: 1.8098684499661144, lat: -18.75, lon: 157.5}
	pixels[846] = &BallPixel{col: 28, row: 9, x: 4.1474918872118005, y: -2.2100728352864585, z: 1.7146472007036226, lat: -26.25, lon: 157.5}
	pixels[847] = &BallPixel{col: 28, row: 8, x: 3.8460058718919763, y: -2.779083251953125, z: 1.5900074988603607, lat: -33.75, lon: 157.5}
	pixels[848] = &BallPixel{col: 28, row: 7, x: 3.4781748801469816, y: -3.2997538248697915, z: 1.4379395991563815, lat: -41.25, lon: 157.5}
	pixels[849] = &BallPixel{col: 28, row: 6, x: 3.0500165969133386, y: -3.762969970703125, z: 1.260931319991749, lat: -48.75, lon: 157.5}
	pixels[850] = &BallPixel{col: 28, row: 5, x: 2.5687522441148762, y: -4.160919189453125, z: 1.061968043446542, lat: -56.25, lon: 157.5}
	pixels[851] = &BallPixel{col: 28, row: 4, x: 2.0428065806627274, y: -4.487091064453125, z: 0.8445327152808515, lat: -63.75, lon: 157.5}
	pixels[852] = &BallPixel{col: 28, row: 3, x: 1.4818079024553308, y: -4.736277262369791, z: 0.6126058449347822, lat: -71.25, lon: 157.5}
	pixels[853] = &BallPixel{col: 28, row: 2, x: 0.8965880423784258, y: -4.904571533203126, z: 0.3706655055284504, lat: -78.75, lon: 157.5}
	pixels[854] = &BallPixel{col: 28, row: 1, x: 0.2991823703050611, y: -4.989369710286458, z: 0.12368733386198681, lat: -86.25, lon: 157.5}
	pixels[855] = &BallPixel{col: 27, row: 5, x: 2.4525878278654996, y: -4.160919189453125, z: 1.3096762707573375, lat: -56.25, lon: 151.875}
	pixels[856] = &BallPixel{col: 27, row: 6, x: 2.9120883874711576, y: -3.762969970703125, z: 1.5550485149142348, lat: -48.75, lon: 151.875}
	pixels[857] = &BallPixel{col: 27, row: 7, x: 3.320884446438867, y: -3.2997538248697915, z: 1.7733446721103991, lat: -41.25, lon: 151.875}
	pixels[858] = &BallPixel{col: 27, row: 8, x: 3.672081341792363, y: -2.779083251953125, z: 1.960883008141538, lat: -33.75, lon: 151.875}
	pixels[859] = &BallPixel{col: 27, row: 9, x: 3.9599335210514246, y: -2.2100728352864585, z: 2.1145954111707415, lat: -26.25, lon: 151.875}
	pixels[860] = &BallPixel{col: 27, row: 10, x: 4.179844542231875, y: -1.6031392415364585, z: 2.232027391728481, lat: -18.75, lon: 151.875}
	pixels[861] = &BallPixel{col: 27, row: 11, x: 4.328367073845584, y: -0.970001220703125, z: 2.311338082712612, lat: -11.25, lon: 151.875}
	pixels[862] = &BallPixel{col: 27, row: 12, x: 4.403202894900458, y: -0.32367960611979163, z: 2.3513002393883675, lat: -3.75, lon: 151.875}
	pixels[863] = &BallPixel{col: 27, row: 13, x: 4.403202894900461, y: 0.32367960611979163, z: 2.351300239388369, lat: 3.75, lon: 151.875}
	pixels[864] = &BallPixel{col: 27, row: 14, x: 4.328367073845583, y: 0.970001220703125, z: 2.3113380827126115, lat: 11.25, lon: 151.875}
	pixels[865] = &BallPixel{col: 27, row: 15, x: 4.179844542231877, y: 1.6031392415364585, z: 2.2320273917284825, lat: 18.75, lon: 151.875}
	pixels[866] = &BallPixel{col: 27, row: 16, x: 3.959933521051427, y: 2.2100728352864585, z: 2.114595411170743, lat: 26.25, lon: 151.875}
	pixels[867] = &BallPixel{col: 27, row: 17, x: 3.672081341792365, y: 2.779083251953125, z: 1.9608830081415394, lat: 33.75, lon: 151.875}
	pixels[868] = &BallPixel{col: 26, row: 19, x: 2.746001802074417, y: 3.762969970703125, z: 1.8340581180527822, lat: 48.75, lon: 146.25}
	pixels[869] = &BallPixel{col: 26, row: 18, x: 3.131482792086902, y: 3.2997538248697915, z: 2.0915213646367197, lat: 41.25, lon: 146.25}
	pixels[870] = &BallPixel{col: 26, row: 17, x: 3.4626497002318546, y: 2.779083251953125, z: 2.3127081664279117, lat: 33.75, lon: 146.25}
	pixels[871] = &BallPixel{col: 26, row: 16, x: 3.7340846629813362, y: 2.2100728352864585, z: 2.4939999254420413, lat: 26.25, lon: 146.25}
	pixels[872] = &BallPixel{col: 26, row: 15, x: 3.941453389513, y: 1.6031392415364585, z: 2.6325017632916583, lat: 18.75, lon: 146.25}
	pixels[873] = &BallPixel{col: 26, row: 14, x: 4.081505161710086, y: 0.970001220703125, z: 2.726042521186173, lat: 11.25, lon: 146.25}
	pixels[874] = &BallPixel{col: 26, row: 13, x: 4.152072834161426, y: 0.32367960611979163, z: 2.7731747599318637, lat: 3.75, lon: 146.25}
	pixels[875] = &BallPixel{col: 26, row: 12, x: 4.152072834161423, y: -0.32367960611979163, z: 2.7731747599318624, lat: -3.75, lon: 146.25}
	pixels[876] = &BallPixel{col: 26, row: 11, x: 4.081505161710087, y: -0.970001220703125, z: 2.7260425211861734, lat: -11.25, lon: 146.25}
	pixels[877] = &BallPixel{col: 26, row: 10, x: 3.941453389512998, y: -1.6031392415364585, z: 2.6325017632916565, lat: -18.75, lon: 146.25}
	pixels[878] = &BallPixel{col: 26, row: 9, x: 3.734084662981334, y: -2.2100728352864585, z: 2.49399992544204, lat: -26.25, lon: 146.25}
	pixels[879] = &BallPixel{col: 26, row: 8, x: 3.4626497002318524, y: -2.779083251953125, z: 2.3127081664279103, lat: -33.75, lon: 146.25}
	pixels[880] = &BallPixel{col: 26, row: 7, x: 3.131482792086902, y: -3.2997538248697915, z: 2.0915213646367197, lat: -41.25, lon: 146.25}
	pixels[881] = &BallPixel{col: 26, row: 6, x: 2.7460018020744146, y: -3.762969970703125, z: 1.8340581180527809, lat: -48.75, lon: 146.25}
	pixels[882] = &BallPixel{col: 26, row: 5, x: 2.3127081664279117, y: -4.160919189453125, z: 1.5446607442572713, lat: -56.25, lon: 146.25}
	pixels[883] = &BallPixel{col: 26, row: 4, x: 1.839186894086501, y: -4.487091064453125, z: 1.2283952804282305, lat: -63.75, lon: 146.25}
	pixels[884] = &BallPixel{col: 26, row: 3, x: 1.334106566694877, y: -4.736277262369791, z: 0.8910514833405617, lat: -71.25, lon: 146.25}
	pixels[900] = &BallPixel{col: 25, row: 17, x: 3.2195966176805126, y: 2.779083251953125, z: 2.6418830573675223, lat: 33.75, lon: 140.625}
	pixels[901] = &BallPixel{col: 25, row: 16, x: 3.471978800010406, y: 2.2100728352864585, z: 2.8489786319551054, lat: 26.25, lon: 140.625}
	pixels[902] = &BallPixel{col: 25, row: 15, x: 3.6647917347145307, y: 1.6031392415364585, z: 3.0071938638381344, lat: 18.75, lon: 140.625}
	pixels[903] = &BallPixel{col: 25, row: 14, x: 3.7950128806871386, y: 0.970001220703125, z: 3.1140485664946005, lat: 11.25, lon: 140.625}
	pixels[904] = &BallPixel{col: 25, row: 13, x: 3.8606272105244033, y: 0.32367960611979163, z: 3.1678892822431703, lat: 3.75, lon: 140.625}
	pixels[905] = &BallPixel{col: 25, row: 12, x: 3.860627210524401, y: -0.32367960611979163, z: 3.1678892822431686, lat: -3.75, lon: 140.625}
	pixels[906] = &BallPixel{col: 25, row: 11, x: 3.7950128806871395, y: -0.970001220703125, z: 3.114048566494601, lat: -11.25, lon: 140.625}
	pixels[907] = &BallPixel{col: 25, row: 10, x: 3.6647917347145285, y: -1.6031392415364585, z: 3.0071938638381326, lat: -18.75, lon: 140.625}
	pixels[908] = &BallPixel{col: 25, row: 9, x: 3.471978800010404, y: -2.2100728352864585, z: 2.848978631955104, lat: -26.25, lon: 140.625}
	pixels[909] = &BallPixel{col: 25, row: 8, x: 3.219596617680511, y: -2.779083251953125, z: 2.6418830573675205, lat: -33.75, lon: 140.625}
	pixels[910] = &BallPixel{col: 25, row: 7, x: 2.9116752425325125, y: -3.2997538248697915, z: 2.389214055438059, lat: -41.25, lon: 140.625}
	pixels[911] = &BallPixel{col: 25, row: 6, x: 2.5532522430759874, y: -3.762969970703125, z: 2.0951052703700634, lat: -48.75, lon: 140.625}
	pixels[912] = &BallPixel{col: 25, row: 5, x: 2.15037270152243, y: -4.160919189453125, z: 1.7645170752075487, lat: -56.25, lon: 140.625}
	pixels[913] = &BallPixel{col: 23, row: 5, x: 1.7645170752075487, y: -4.160919189453125, z: 2.15037270152243, lat: -56.25, lon: 129.375}
	pixels[914] = &BallPixel{col: 23, row: 6, x: 2.0951052703700634, y: -3.762969970703125, z: 2.5532522430759874, lat: -48.75, lon: 129.375}
	pixels[915] = &BallPixel{col: 23, row: 7, x: 2.389214055438059, y: -3.2997538248697915, z: 2.9116752425325125, lat: -41.25, lon: 129.375}
	pixels[916] = &BallPixel{col: 23, row: 8, x: 2.6418830573675205, y: -2.779083251953125, z: 3.219596617680511, lat: -33.75, lon: 129.375}
	pixels[917] = &BallPixel{col: 23, row: 9, x: 2.848978631955104, y: -2.2100728352864585, z: 3.471978800010404, lat: -26.25, lon: 129.375}
	pixels[918] = &BallPixel{col: 23, row: 10, x: 3.0071938638381326, y: -1.6031392415364585, z: 3.6647917347145285, lat: -18.75, lon: 129.375}
	pixels[919] = &BallPixel{col: 23, row: 11, x: 3.114048566494601, y: -0.970001220703125, z: 3.7950128806871395, lat: -11.25, lon: 129.375}
	pixels[920] = &BallPixel{col: 23, row: 12, x: 3.1678892822431686, y: -0.32367960611979163, z: 3.860627210524401, lat: -3.75, lon: 129.375}
	pixels[921] = &BallPixel{col: 23, row: 13, x: 3.1678892822431703, y: 0.32367960611979163, z: 3.8606272105244033, lat: 3.75, lon: 129.375}
	pixels[922] = &BallPixel{col: 23, row: 14, x: 3.1140485664946005, y: 0.970001220703125, z: 3.7950128806871386, lat: 11.25, lon: 129.375}
	pixels[923] = &BallPixel{col: 23, row: 15, x: 3.0071938638381344, y: 1.6031392415364585, z: 3.6647917347145307, lat: 18.75, lon: 129.375}
	pixels[924] = &BallPixel{col: 23, row: 16, x: 2.8489786319551054, y: 2.2100728352864585, z: 3.471978800010406, lat: 26.25, lon: 129.375}
	pixels[925] = &BallPixel{col: 23, row: 17, x: 2.6418830573675223, y: 2.779083251953125, z: 3.2195966176805126, lat: 33.75, lon: 129.375}
	pixels[926] = &BallPixel{col: 22, row: 19, x: 1.8340581180527822, y: 3.762969970703125, z: 2.746001802074417, lat: 48.75, lon: 123.75}
	pixels[927] = &BallPixel{col: 22, row: 18, x: 2.0915213646367197, y: 3.2997538248697915, z: 3.131482792086902, lat: 41.25, lon: 123.75}
	pixels[928] = &BallPixel{col: 22, row: 17, x: 2.3127081664279117, y: 2.779083251953125, z: 3.4626497002318546, lat: 33.75, lon: 123.75}
	pixels[929] = &BallPixel{col: 22, row: 16, x: 2.4939999254420413, y: 2.2100728352864585, z: 3.7340846629813362, lat: 26.25, lon: 123.75}
	pixels[930] = &BallPixel{col: 22, row: 15, x: 2.6325017632916583, y: 1.6031392415364585, z: 3.941453389513, lat: 18.75, lon: 123.75}
	pixels[931] = &BallPixel{col: 22, row: 14, x: 2.726042521186173, y: 0.970001220703125, z: 4.081505161710086, lat: 11.25, lon: 123.75}
	pixels[932] = &BallPixel{col: 22, row: 13, x: 2.7731747599318637, y: 0.32367960611979163, z: 4.152072834161426, lat: 3.75, lon: 123.75}
	pixels[933] = &BallPixel{col: 22, row: 12, x: 2.7731747599318624, y: -0.32367960611979163, z: 4.152072834161423, lat: -3.75, lon: 123.75}
	pixels[934] = &BallPixel{col: 22, row: 11, x: 2.7260425211861734, y: -0.970001220703125, z: 4.081505161710087, lat: -11.25, lon: 123.75}
	pixels[935] = &BallPixel{col: 22, row: 10, x: 2.6325017632916565, y: -1.6031392415364585, z: 3.941453389512998, lat: -18.75, lon: 123.75}
	pixels[936] = &BallPixel{col: 22, row: 9, x: 2.49399992544204, y: -2.2100728352864585, z: 3.734084662981334, lat: -26.25, lon: 123.75}
	pixels[937] = &BallPixel{col: 22, row: 8, x: 2.3127081664279103, y: -2.779083251953125, z: 3.4626497002318524, lat: -33.75, lon: 123.75}
	pixels[938] = &BallPixel{col: 22, row: 7, x: 2.0915213646367197, y: -3.2997538248697915, z: 3.131482792086902, lat: -41.25, lon: 123.75}
	pixels[939] = &BallPixel{col: 22, row: 6, x: 1.8340581180527809, y: -3.762969970703125, z: 2.7460018020744146, lat: -48.75, lon: 123.75}
	pixels[940] = &BallPixel{col: 22, row: 5, x: 1.5446607442572713, y: -4.160919189453125, z: 2.3127081664279117, lat: -56.25, lon: 123.75}
	pixels[941] = &BallPixel{col: 22, row: 4, x: 1.2283952804282305, y: -4.487091064453125, z: 1.839186894086501, lat: -63.75, lon: 123.75}
	pixels[942] = &BallPixel{col: 22, row: 3, x: 0.8910514833405617, y: -4.736277262369791, z: 1.334106566694877, lat: -71.25, lon: 123.75}
	pixels[943] = &BallPixel{col: 21, row: 5, x: 1.3096762707573375, y: -4.160919189453125, z: 2.4525878278654996, lat: -56.25, lon: 118.125}
	pixels[944] = &BallPixel{col: 21, row: 6, x: 1.5550485149142348, y: -3.762969970703125, z: 2.9120883874711576, lat: -48.75, lon: 118.125}
	pixels[945] = &BallPixel{col: 21, row: 7, x: 1.7733446721103991, y: -3.2997538248697915, z: 3.320884446438867, lat: -41.25, lon: 118.125}
	pixels[946] = &BallPixel{col: 21, row: 8, x: 1.960883008141538, y: -2.779083251953125, z: 3.672081341792363, lat: -33.75, lon: 118.125}
	pixels[947] = &BallPixel{col: 21, row: 9, x: 2.1145954111707415, y: -2.2100728352864585, z: 3.9599335210514246, lat: -26.25, lon: 118.125}
	pixels[948] = &BallPixel{col: 21, row: 10, x: 2.232027391728481, y: -1.6031392415364585, z: 4.179844542231875, lat: -18.75, lon: 118.125}
	pixels[949] = &BallPixel{col: 21, row: 11, x: 2.311338082712612, y: -0.970001220703125, z: 4.328367073845584, lat: -11.25, lon: 118.125}
	pixels[950] = &BallPixel{col: 21, row: 12, x: 2.3513002393883675, y: -0.32367960611979163, z: 4.403202894900458, lat: -3.75, lon: 118.125}
	pixels[951] = &BallPixel{col: 21, row: 13, x: 2.351300239388369, y: 0.32367960611979163, z: 4.403202894900461, lat: 3.75, lon: 118.125}
	pixels[952] = &BallPixel{col: 21, row: 14, x: 2.3113380827126115, y: 0.970001220703125, z: 4.328367073845583, lat: 11.25, lon: 118.125}
	pixels[953] = &BallPixel{col: 21, row: 15, x: 2.2320273917284825, y: 1.6031392415364585, z: 4.179844542231877, lat: 18.75, lon: 118.125}
	pixels[954] = &BallPixel{col: 21, row: 16, x: 2.114595411170743, y: 2.2100728352864585, z: 3.959933521051427, lat: 26.25, lon: 118.125}
	pixels[955] = &BallPixel{col: 21, row: 17, x: 1.9608830081415394, y: 2.779083251953125, z: 3.672081341792365, lat: 33.75, lon: 118.125}
	pixels[956] = &BallPixel{col: 20, row: 19, x: 1.26093131999175, y: 3.762969970703125, z: 3.0500165969133413, lat: 48.75, lon: 112.5}
	pixels[957] = &BallPixel{col: 20, row: 18, x: 1.4379395991563815, y: 3.2997538248697915, z: 3.478174880146982, lat: 41.25, lon: 112.5}
	pixels[958] = &BallPixel{col: 20, row: 17, x: 1.5900074988603619, y: 2.779083251953125, z: 3.846005871891979, lat: 33.75, lon: 112.5}
	pixels[959] = &BallPixel{col: 20, row: 16, x: 1.7146472007036238, y: 2.2100728352864585, z: 4.147491887211803, lat: 26.25, lon: 112.5}
	pixels[960] = &BallPixel{col: 20, row: 15, x: 1.8098684499661155, y: 1.6031392415364585, z: 4.377818778157237, lat: 18.75, lon: 112.5}
	pixels[961] = &BallPixel{col: 20, row: 14, x: 1.8741785556077977, y: 0.970001220703125, z: 4.533375933766366, lat: 11.25, lon: 112.5}
	pixels[962] = &BallPixel{col: 20, row: 13, x: 1.9065823902686465, y: 0.32367960611979163, z: 4.611756280064586, lat: 3.75, lon: 112.5}
	pixels[963] = &BallPixel{col: 20, row: 12, x: 1.9065823902686456, y: -0.32367960611979163, z: 4.611756280064584, lat: -3.75, lon: 112.5}
	pixels[964] = &BallPixel{col: 20, row: 11, x: 1.8741785556077981, y: -0.970001220703125, z: 4.533375933766367, lat: -11.25, lon: 112.5}
	pixels[965] = &BallPixel{col: 20, row: 10, x: 1.8098684499661144, y: -1.6031392415364585, z: 4.377818778157235, lat: -18.75, lon: 112.5}
	pixels[966] = &BallPixel{col: 20, row: 9, x: 1.7146472007036226, y: -2.2100728352864585, z: 4.1474918872118005, lat: -26.25, lon: 112.5}
	pixels[967] = &BallPixel{col: 20, row: 8, x: 1.5900074988603607, y: -2.779083251953125, z: 3.8460058718919763, lat: -33.75, lon: 112.5}
	pixels[968] = &BallPixel{col: 20, row: 7, x: 1.4379395991563815, y: -3.2997538248697915, z: 3.4781748801469816, lat: -41.25, lon: 112.5}
	pixels[969] = &BallPixel{col: 20, row: 6, x: 1.260931319991749, y: -3.762969970703125, z: 3.0500165969133386, lat: -48.75, lon: 112.5}
	pixels[970] = &BallPixel{col: 20, row: 5, x: 1.061968043446542, y: -4.160919189453125, z: 2.5687522441148762, lat: -56.25, lon: 112.5}
	pixels[971] = &BallPixel{col: 20, row: 4, x: 0.8445327152808515, y: -4.487091064453125, z: 2.0428065806627274, lat: -63.75, lon: 112.5}
	pixels[972] = &BallPixel{col: 20, row: 3, x: 0.6126058449347822, y: -4.736277262369791, z: 1.4818079024553308, lat: -71.25, lon: 112.5}
	pixels[973] = &BallPixel{col: 20, row: 2, x: 0.3706655055284504, y: -4.904571533203126, z: 0.8965880423784258, lat: -78.75, lon: 112.5}
	pixels[974] = &BallPixel{col: 20, row: 1, x: 0.12368733386198681, y: -4.989369710286458, z: 0.2991823703050611, lat: -86.25, lon: 112.5}
	pixels[975] = &BallPixel{col: 19, row: 5, x: 0.8041694404673763, y: -4.160919189453125, z: 2.6601709628594117, lat: -56.25, lon: 106.875}
	pixels[976] = &BallPixel{col: 19, row: 6, x: 0.9548332836595366, y: -3.762969970703125, z: 3.1585629193849565, lat: -48.75, lon: 106.875}
	pixels[977] = &BallPixel{col: 19, row: 7, x: 1.0888718262431214, y: -3.2997538248697915, z: 3.6019588269409737, lat: -41.25, lon: 106.875}
	pixels[978] = &BallPixel{col: 19, row: 8, x: 1.2040244041127153, y: -2.779083251953125, z: 3.98288046923699, lat: -33.75, lon: 106.875}
	pixels[979] = &BallPixel{col: 19, row: 9, x: 1.2984071305138059, y: -2.2100728352864585, z: 4.2950959993642766, lat: -26.25, lon: 106.875}
	pixels[980] = &BallPixel{col: 19, row: 10, x: 1.3705128960427826, y: -1.6031392415364585, z: 4.533619939795852, lat: -18.75, lon: 106.875}
	pixels[981] = &BallPixel{col: 19, row: 11, x: 1.4192113686469388, y: -0.970001220703125, z: 4.6947131823864785, lat: -11.25, lon: 106.875}
	pixels[982] = &BallPixel{col: 19, row: 12, x: 1.4437489936244674, y: -0.32367960611979163, z: 4.775882988372664, lat: -3.75, lon: 106.875}
	pixels[983] = &BallPixel{col: 19, row: 13, x: 1.443748993624468, y: 0.32367960611979163, z: 4.775882988372667, lat: 3.75, lon: 106.875}
	pixels[984] = &BallPixel{col: 19, row: 14, x: 1.4192113686469383, y: 0.970001220703125, z: 4.694713182386478, lat: 11.25, lon: 106.875}
	pixels[985] = &BallPixel{col: 19, row: 15, x: 1.3705128960427835, y: 1.6031392415364585, z: 4.533619939795853, lat: 18.75, lon: 106.875}
	pixels[986] = &BallPixel{col: 19, row: 16, x: 1.2984071305138067, y: 2.2100728352864585, z: 4.295095999364279, lat: 26.25, lon: 106.875}
	pixels[987] = &BallPixel{col: 19, row: 17, x: 1.2040244041127162, y: 2.779083251953125, z: 3.982880469236992, lat: 33.75, lon: 106.875}
	pixels[988] = &BallPixel{col: 18, row: 19, x: 0.6401530476287038, y: 3.762969970703125, z: 3.236775735206905, lat: 48.75, lon: 101.25}
	pixels[989] = &BallPixel{col: 18, row: 18, x: 0.7300170930102499, y: 3.2997538248697915, z: 3.6911510797217497, lat: 41.25, lon: 101.25}
	pixels[990] = &BallPixel{col: 18, row: 17, x: 0.8072193386033215, y: 2.779083251953125, z: 4.081505161710086, lat: 33.75, lon: 101.25}
	pixels[991] = &BallPixel{col: 18, row: 16, x: 0.8704967619851272, y: 2.2100728352864585, z: 4.401451820321384, lat: 26.25, lon: 101.25}
	pixels[992] = &BallPixel{col: 18, row: 15, x: 0.9188389452174347, y: 1.6031392415364585, z: 4.6458821268752235, lat: 18.75, lon: 101.25}
	pixels[993] = &BallPixel{col: 18, row: 14, x: 0.9514880748465695, y: 0.970001220703125, z: 4.810964384861291, lat: 11.25, lon: 101.25}
	pixels[994] = &BallPixel{col: 18, row: 13, x: 0.9679389419034167, y: 0.32367960611979163, z: 4.894144129939379, lat: 3.75, lon: 101.25}
	pixels[995] = &BallPixel{col: 18, row: 12, x: 0.9679389419034161, y: -0.32367960611979163, z: 4.894144129939376, lat: -3.75, lon: 101.25}
	pixels[996] = &BallPixel{col: 18, row: 11, x: 0.9514880748465697, y: -0.970001220703125, z: 4.8109643848612915, lat: -11.25, lon: 101.25}
	pixels[997] = &BallPixel{col: 18, row: 10, x: 0.9188389452174341, y: -1.6031392415364585, z: 4.645882126875221, lat: -18.75, lon: 101.25}
	pixels[998] = &BallPixel{col: 18, row: 9, x: 0.8704967619851266, y: -2.2100728352864585, z: 4.401451820321381, lat: -26.25, lon: 101.25}
	pixels[999] = &BallPixel{col: 18, row: 8, x: 0.807219338603321, y: -2.779083251953125, z: 4.0815051617100835, lat: -33.75, lon: 101.25}
	pixels[1000] = &BallPixel{col: 18, row: 7, x: 0.7300170930102498, y: -3.2997538248697915, z: 3.6911510797217493, lat: -41.25, lon: 101.25}
	pixels[1001] = &BallPixel{col: 18, row: 6, x: 0.6401530476287034, y: -3.762969970703125, z: 3.236775735206902, lat: -48.75, lon: 101.25}
	pixels[1002] = &BallPixel{col: 18, row: 5, x: 0.5391428293660304, y: -4.160919189453125, z: 2.726042521186173, lat: -56.25, lon: 101.25}
	pixels[1003] = &BallPixel{col: 18, row: 4, x: 0.4287546696141379, y: -4.487091064453125, z: 2.1678920628502962, lat: -63.75, lon: 101.25}
	pixels[1004] = &BallPixel{col: 18, row: 3, x: 0.3110094042494907, y: -4.736277262369791, z: 1.572542217560113, lat: -71.25, lon: 101.25}
	pixels[1005] = &BallPixel{col: 17, row: 5, x: 0.2699795670923787, y: -4.160919189453125, z: 2.765794445585925, lat: -56.25, lon: 95.625}
	pixels[1006] = &BallPixel{col: 17, row: 6, x: 0.3205611449472308, y: -3.762969970703125, z: 3.2839753160369587, lat: -48.75, lon: 95.625}
	pixels[1007] = &BallPixel{col: 17, row: 7, x: 0.36556119826855277, y: -3.2997538248697915, z: 3.744976490561385, lat: -41.25, lon: 95.625}
	pixels[1008] = &BallPixel{col: 17, row: 8, x: 0.40422076621325836, y: -2.779083251953125, z: 4.1410228263703175, lat: -33.75, lon: 95.625}
	pixels[1009] = &BallPixel{col: 17, row: 9, x: 0.4359073814121091, y: -2.2100728352864585, z: 4.465635037806351, lat: -26.25, lon: 95.625}
	pixels[1010] = &BallPixel{col: 17, row: 10, x: 0.4601150699697141, y: -1.6031392415364585, z: 4.713629696343559, lat: -18.75, lon: 95.625}
	pixels[1011] = &BallPixel{col: 17, row: 11, x: 0.4764643514645304, y: -0.970001220703125, z: 4.881119230587502, lat: -11.25, lon: 95.625}
	pixels[1012] = &BallPixel{col: 17, row: 12, x: 0.4847022389488623, y: -0.32367960611979163, z: 4.965511926275212, lat: -3.75, lon: 95.625}
	pixels[1013] = &BallPixel{col: 17, row: 13, x: 0.48470223894886255, y: 0.32367960611979163, z: 4.965511926275215, lat: 3.75, lon: 95.625}
	pixels[1014] = &BallPixel{col: 17, row: 14, x: 0.47646435146453037, y: 0.970001220703125, z: 4.881119230587501, lat: 11.25, lon: 95.625}
	pixels[1015] = &BallPixel{col: 17, row: 15, x: 0.46011506996971435, y: 1.6031392415364585, z: 4.7136296963435615, lat: 18.75, lon: 95.625}
	pixels[1016] = &BallPixel{col: 17, row: 16, x: 0.4359073814121094, y: 2.2100728352864585, z: 4.465635037806353, lat: 26.25, lon: 95.625}
	pixels[1017] = &BallPixel{col: 17, row: 17, x: 0.40422076621325864, y: 2.779083251953125, z: 4.14102282637032, lat: 33.75, lon: 95.625}
	pixels[1018] = &BallPixel{col: 15, row: 17, x: -0.40422076621325864, y: 2.779083251953125, z: 4.141022826370321, lat: 33.75, lon: 84.375}
	pixels[1019] = &BallPixel{col: 15, row: 16, x: -0.4359073814121094, y: 2.2100728352864585, z: 4.465635037806354, lat: 26.25, lon: 84.375}
	pixels[1020] = &BallPixel{col: 15, row: 15, x: -0.46011506996971435, y: 1.6031392415364585, z: 4.713629696343563, lat: 18.75, lon: 84.375}
	pixels[1021] = &BallPixel{col: 15, row: 14, x: -0.47646435146453037, y: 0.970001220703125, z: 4.881119230587502, lat: 11.25, lon: 84.375}
	pixels[1022] = &BallPixel{col: 15, row: 13, x: -0.48470223894886255, y: 0.32367960611979163, z: 4.965511926275216, lat: 3.75, lon: 84.375}
	pixels[1023] = &BallPixel{col: 15, row: 12, x: -0.4847022389488623, y: -0.32367960611979163, z: 4.965511926275213, lat: -3.75, lon: 84.375}
	pixels[1024] = &BallPixel{col: 15, row: 11, x: -0.4764643514645304, y: -0.970001220703125, z: 4.881119230587503, lat: -11.25, lon: 84.375}
	pixels[1025] = &BallPixel{col: 15, row: 10, x: -0.4601150699697141, y: -1.6031392415364585, z: 4.713629696343561, lat: -18.75, lon: 84.375}
	pixels[1026] = &BallPixel{col: 15, row: 9, x: -0.4359073814121091, y: -2.2100728352864585, z: 4.465635037806352, lat: -26.25, lon: 84.375}
	pixels[1027] = &BallPixel{col: 15, row: 8, x: -0.40422076621325836, y: -2.779083251953125, z: 4.141022826370318, lat: -33.75, lon: 84.375}
	pixels[1028] = &BallPixel{col: 15, row: 7, x: -0.36556119826855277, y: -3.2997538248697915, z: 3.744976490561386, lat: -41.25, lon: 84.375}
	pixels[1029] = &BallPixel{col: 15, row: 6, x: -0.3205611449472308, y: -3.762969970703125, z: 3.283975316036959, lat: -48.75, lon: 84.375}
	pixels[1030] = &BallPixel{col: 15, row: 5, x: -0.2699795670923787, y: -4.160919189453125, z: 2.7657944455859256, lat: -56.25, lon: 84.375}
	pixels[1050] = &BallPixel{col: 13, row: 17, x: -1.2040244041127162, y: 2.779083251953125, z: 3.9828804692369917, lat: 33.75, lon: 73.125}
	pixels[1051] = &BallPixel{col: 13, row: 16, x: -1.2984071305138067, y: 2.2100728352864585, z: 4.295095999364279, lat: 26.25, lon: 73.125}
	pixels[1052] = &BallPixel{col: 13, row: 15, x: -1.3705128960427835, y: 1.6031392415364585, z: 4.533619939795853, lat: 18.75, lon: 73.125}
	pixels[1053] = &BallPixel{col: 13, row: 14, x: -1.4192113686469383, y: 0.970001220703125, z: 4.694713182386478, lat: 11.25, lon: 73.125}
	pixels[1054] = &BallPixel{col: 13, row: 13, x: -1.443748993624468, y: 0.32367960611979163, z: 4.775882988372666, lat: 3.75, lon: 73.125}
	pixels[1055] = &BallPixel{col: 13, row: 12, x: -1.4437489936244674, y: -0.32367960611979163, z: 4.775882988372663, lat: -3.75, lon: 73.125}
	pixels[1056] = &BallPixel{col: 13, row: 11, x: -1.4192113686469388, y: -0.970001220703125, z: 4.6947131823864785, lat: -11.25, lon: 73.125}
	pixels[1057] = &BallPixel{col: 13, row: 10, x: -1.3705128960427826, y: -1.6031392415364585, z: 4.533619939795851, lat: -18.75, lon: 73.125}
	pixels[1058] = &BallPixel{col: 13, row: 9, x: -1.2984071305138059, y: -2.2100728352864585, z: 4.2950959993642766, lat: -26.25, lon: 73.125}
	pixels[1059] = &BallPixel{col: 13, row: 8, x: -1.2040244041127153, y: -2.779083251953125, z: 3.9828804692369895, lat: -33.75, lon: 73.125}
	pixels[1060] = &BallPixel{col: 13, row: 7, x: -1.0888718262431214, y: -3.2997538248697915, z: 3.6019588269409732, lat: -41.25, lon: 73.125}
	pixels[1061] = &BallPixel{col: 13, row: 6, x: -0.9548332836595366, y: -3.762969970703125, z: 3.158562919384956, lat: -48.75, lon: 73.125}
	pixels[1062] = &BallPixel{col: 13, row: 5, x: -0.8041694404673763, y: -4.160919189453125, z: 2.6601709628594112, lat: -56.25, lon: 73.125}
	pixels[1063] = &BallPixel{col: 14, row: 3, x: -0.3110094042494907, y: -4.736277262369791, z: 1.5725422175601134, lat: -71.25, lon: 78.75}
	pixels[1064] = &BallPixel{col: 14, row: 4, x: -0.4287546696141379, y: -4.487091064453125, z: 2.1678920628502967, lat: -63.75, lon: 78.75}
	pixels[1065] = &BallPixel{col: 14, row: 5, x: -0.5391428293660304, y: -4.160919189453125, z: 2.7260425211861734, lat: -56.25, lon: 78.75}
	pixels[1066] = &BallPixel{col: 14, row: 6, x: -0.6401530476287034, y: -3.762969970703125, z: 3.236775735206903, lat: -48.75, lon: 78.75}
	pixels[1067] = &BallPixel{col: 14, row: 7, x: -0.7300170930102498, y: -3.2997538248697915, z: 3.6911510797217497, lat: -41.25, lon: 78.75}
	pixels[1068] = &BallPixel{col: 14, row: 8, x: -0.807219338603321, y: -2.779083251953125, z: 4.081505161710084, lat: -33.75, lon: 78.75}
	pixels[1069] = &BallPixel{col: 14, row: 9, x: -0.8704967619851266, y: -2.2100728352864585, z: 4.401451820321382, lat: -26.25, lon: 78.75}
	pixels[1070] = &BallPixel{col: 14, row: 10, x: -0.9188389452174341, y: -1.6031392415364585, z: 4.645882126875222, lat: -18.75, lon: 78.75}
	pixels[1071] = &BallPixel{col: 14, row: 11, x: -0.9514880748465697, y: -0.970001220703125, z: 4.810964384861292, lat: -11.25, lon: 78.75}
	pixels[1072] = &BallPixel{col: 14, row: 12, x: -0.9679389419034161, y: -0.32367960611979163, z: 4.894144129939378, lat: -3.75, lon: 78.75}
	pixels[1073] = &BallPixel{col: 14, row: 13, x: -0.9679389419034167, y: 0.32367960611979163, z: 4.894144129939381, lat: 3.75, lon: 78.75}
	pixels[1074] = &BallPixel{col: 14, row: 14, x: -0.9514880748465695, y: 0.970001220703125, z: 4.8109643848612915, lat: 11.25, lon: 78.75}
	pixels[1075] = &BallPixel{col: 14, row: 15, x: -0.9188389452174347, y: 1.6031392415364585, z: 4.645882126875224, lat: 18.75, lon: 78.75}
	pixels[1076] = &BallPixel{col: 14, row: 16, x: -0.8704967619851272, y: 2.2100728352864585, z: 4.401451820321385, lat: 26.25, lon: 78.75}
	pixels[1077] = &BallPixel{col: 14, row: 17, x: -0.8072193386033215, y: 2.779083251953125, z: 4.081505161710087, lat: 33.75, lon: 78.75}
	pixels[1078] = &BallPixel{col: 14, row: 18, x: -0.7300170930102499, y: 3.2997538248697915, z: 3.69115107972175, lat: 41.25, lon: 78.75}
	pixels[1079] = &BallPixel{col: 14, row: 19, x: -0.6401530476287038, y: 3.762969970703125, z: 3.2367757352069053, lat: 48.75, lon: 78.75}
	pixels[1080] = &BallPixel{col: 12, row: 19, x: -1.26093131999175, y: 3.762969970703125, z: 3.0500165969133404, lat: 48.75, lon: 67.5}
	pixels[1081] = &BallPixel{col: 12, row: 18, x: -1.4379395991563815, y: 3.2997538248697915, z: 3.478174880146981, lat: 41.25, lon: 67.5}
	pixels[1082] = &BallPixel{col: 12, row: 17, x: -1.5900074988603619, y: 2.779083251953125, z: 3.846005871891978, lat: 33.75, lon: 67.5}
	pixels[1083] = &BallPixel{col: 12, row: 16, x: -1.7146472007036238, y: 2.2100728352864585, z: 4.147491887211802, lat: 26.25, lon: 67.5}
	pixels[1084] = &BallPixel{col: 12, row: 15, x: -1.8098684499661155, y: 1.6031392415364585, z: 4.377818778157236, lat: 18.75, lon: 67.5}
	pixels[1085] = &BallPixel{col: 12, row: 14, x: -1.8741785556077977, y: 0.970001220703125, z: 4.533375933766365, lat: 11.25, lon: 67.5}
	pixels[1086] = &BallPixel{col: 12, row: 13, x: -1.9065823902686465, y: 0.32367960611979163, z: 4.611756280064585, lat: 3.75, lon: 67.5}
	pixels[1087] = &BallPixel{col: 12, row: 12, x: -1.9065823902686456, y: -0.32367960611979163, z: 4.611756280064583, lat: -3.75, lon: 67.5}
	pixels[1088] = &BallPixel{col: 12, row: 11, x: -1.8741785556077981, y: -0.970001220703125, z: 4.533375933766366, lat: -11.25, lon: 67.5}
	pixels[1089] = &BallPixel{col: 12, row: 10, x: -1.8098684499661144, y: -1.6031392415364585, z: 4.377818778157233, lat: -18.75, lon: 67.5}
	pixels[1090] = &BallPixel{col: 12, row: 9, x: -1.7146472007036226, y: -2.2100728352864585, z: 4.1474918872118, lat: -26.25, lon: 67.5}
	pixels[1091] = &BallPixel{col: 12, row: 8, x: -1.5900074988603607, y: -2.779083251953125, z: 3.8460058718919754, lat: -33.75, lon: 67.5}
	pixels[1092] = &BallPixel{col: 12, row: 7, x: -1.4379395991563815, y: -3.2997538248697915, z: 3.4781748801469807, lat: -41.25, lon: 67.5}
	pixels[1093] = &BallPixel{col: 12, row: 6, x: -1.260931319991749, y: -3.762969970703125, z: 3.050016596913338, lat: -48.75, lon: 67.5}
	pixels[1094] = &BallPixel{col: 12, row: 5, x: -1.061968043446542, y: -4.160919189453125, z: 2.568752244114876, lat: -56.25, lon: 67.5}
	pixels[1095] = &BallPixel{col: 12, row: 4, x: -0.8445327152808515, y: -4.487091064453125, z: 2.042806580662727, lat: -63.75, lon: 67.5}
	pixels[1096] = &BallPixel{col: 12, row: 3, x: -0.6126058449347822, y: -4.736277262369791, z: 1.4818079024553303, lat: -71.25, lon: 67.5}
	pixels[1097] = &BallPixel{col: 12, row: 2, x: -0.3706655055284504, y: -4.904571533203126, z: 0.8965880423784256, lat: -78.75, lon: 67.5}
	pixels[1098] = &BallPixel{col: 12, row: 1, x: -0.12368733386198681, y: -4.989369710286458, z: 0.29918237030506106, lat: -86.25, lon: 67.5}
	pixels[1099] = &BallPixel{col: 10, row: 3, x: -0.8910514833405617, y: -4.736277262369791, z: 1.3341065666948762, lat: -71.25, lon: 56.25}
	pixels[1100] = &BallPixel{col: 10, row: 4, x: -1.2283952804282305, y: -4.487091064453125, z: 1.8391868940864997, lat: -63.75, lon: 56.25}
	pixels[1101] = &BallPixel{col: 10, row: 5, x: -1.5446607442572713, y: -4.160919189453125, z: 2.3127081664279103, lat: -56.25, lon: 56.25}
	pixels[1102] = &BallPixel{col: 10, row: 6, x: -1.8340581180527809, y: -3.762969970703125, z: 2.746001802074413, lat: -48.75, lon: 56.25}
	pixels[1103] = &BallPixel{col: 10, row: 7, x: -2.0915213646367197, y: -3.2997538248697915, z: 3.1314827920868997, lat: -41.25, lon: 56.25}
	pixels[1104] = &BallPixel{col: 10, row: 8, x: -2.3127081664279103, y: -2.779083251953125, z: 3.46264970023185, lat: -33.75, lon: 56.25}
	pixels[1105] = &BallPixel{col: 10, row: 9, x: -2.49399992544204, y: -2.2100728352864585, z: 3.7340846629813313, lat: -26.25, lon: 56.25}
	pixels[1106] = &BallPixel{col: 10, row: 10, x: -2.6325017632916565, y: -1.6031392415364585, z: 3.9414533895129953, lat: -18.75, lon: 56.25}
	pixels[1107] = &BallPixel{col: 10, row: 11, x: -2.7260425211861734, y: -0.970001220703125, z: 4.081505161710084, lat: -11.25, lon: 56.25}
	pixels[1108] = &BallPixel{col: 10, row: 12, x: -2.7731747599318624, y: -0.32367960611979163, z: 4.15207283416142, lat: -3.75, lon: 56.25}
	pixels[1109] = &BallPixel{col: 10, row: 13, x: -2.7731747599318637, y: 0.32367960611979163, z: 4.152072834161423, lat: 3.75, lon: 56.25}
	pixels[1110] = &BallPixel{col: 10, row: 14, x: -2.726042521186173, y: 0.970001220703125, z: 4.0815051617100835, lat: 11.25, lon: 56.25}
	pixels[1111] = &BallPixel{col: 10, row: 15, x: -2.6325017632916583, y: 1.6031392415364585, z: 3.9414533895129975, lat: 18.75, lon: 56.25}
	pixels[1112] = &BallPixel{col: 10, row: 16, x: -2.4939999254420413, y: 2.2100728352864585, z: 3.7340846629813336, lat: 26.25, lon: 56.25}
	pixels[1113] = &BallPixel{col: 10, row: 17, x: -2.3127081664279117, y: 2.779083251953125, z: 3.4626497002318524, lat: 33.75, lon: 56.25}
	pixels[1114] = &BallPixel{col: 10, row: 18, x: -2.0915213646367197, y: 3.2997538248697915, z: 3.1314827920869, lat: 41.25, lon: 56.25}
	pixels[1115] = &BallPixel{col: 10, row: 19, x: -1.8340581180527822, y: 3.762969970703125, z: 2.746001802074415, lat: 48.75, lon: 56.25}
	pixels[1116] = &BallPixel{col: 11, row: 17, x: -1.9608830081415394, y: 2.779083251953125, z: 3.6720813417923663, lat: 33.75, lon: 61.875}
	pixels[1117] = &BallPixel{col: 11, row: 16, x: -2.114595411170743, y: 2.2100728352864585, z: 3.959933521051428, lat: 26.25, lon: 61.875}
	pixels[1118] = &BallPixel{col: 11, row: 15, x: -2.2320273917284825, y: 1.6031392415364585, z: 4.179844542231879, lat: 18.75, lon: 61.875}
	pixels[1119] = &BallPixel{col: 11, row: 14, x: -2.3113380827126115, y: 0.970001220703125, z: 4.328367073845585, lat: 11.25, lon: 61.875}
	pixels[1120] = &BallPixel{col: 11, row: 13, x: -2.351300239388369, y: 0.32367960611979163, z: 4.4032028949004625, lat: 3.75, lon: 61.875}
	pixels[1121] = &BallPixel{col: 11, row: 12, x: -2.3513002393883675, y: -0.32367960611979163, z: 4.40320289490046, lat: -3.75, lon: 61.875}
	pixels[1122] = &BallPixel{col: 11, row: 11, x: -2.311338082712612, y: -0.970001220703125, z: 4.328367073845585, lat: -11.25, lon: 61.875}
	pixels[1123] = &BallPixel{col: 11, row: 10, x: -2.232027391728481, y: -1.6031392415364585, z: 4.179844542231876, lat: -18.75, lon: 61.875}
	pixels[1124] = &BallPixel{col: 11, row: 9, x: -2.1145954111707415, y: -2.2100728352864585, z: 3.959933521051426, lat: -26.25, lon: 61.875}
	pixels[1125] = &BallPixel{col: 11, row: 8, x: -1.960883008141538, y: -2.779083251953125, z: 3.672081341792364, lat: -33.75, lon: 61.875}
	pixels[1126] = &BallPixel{col: 11, row: 7, x: -1.7733446721103991, y: -3.2997538248697915, z: 3.3208844464388685, lat: -41.25, lon: 61.875}
	pixels[1127] = &BallPixel{col: 11, row: 6, x: -1.5550485149142348, y: -3.762969970703125, z: 2.9120883874711585, lat: -48.75, lon: 61.875}
	pixels[1128] = &BallPixel{col: 11, row: 5, x: -1.3096762707573375, y: -4.160919189453125, z: 2.4525878278655004, lat: -56.25, lon: 61.875}
}
