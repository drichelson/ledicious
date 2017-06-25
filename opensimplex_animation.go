package main

import (
	"fmt"
	"github.com/launchdarkly/go-metrics"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/ojrac/opensimplex-go"
	"math"
	"time"
)

type OpenSimplexAnimation struct {
	noise    *opensimplex.Noise
	gradient GradientTable
	histo    metrics.Histogram
	min      float64
	max      float64
}

//http://www.rapidtables.com/web/color/color-picker.htm
func NewOpenSimplexAnimation() *OpenSimplexAnimation {
	gradientTable := GradientTable{
		{colorful.Hsv(0.0, 1.0, 0.3), 0.0}, // red
		{colorful.Hsv(0.0, 1.0, 0.0), 0.1},
		{colorful.Hsv(234.0, 0.0, 0.0), 0.90},
		{colorful.Hsv(234.0, 1.0, 0.3), 1.0}, // purple
	}

	return &OpenSimplexAnimation{
		noise:    opensimplex.NewWithSeed(time.Now().UnixNano()),
		gradient: gradientTable,
		histo:    metrics.GetOrRegisterHistogram("histo", metrics.DefaultRegistry, metrics.NewExpDecaySample(expectedPixelCount*10000, 1.0)),
	}
}

func (a *OpenSimplexAnimation) frame(time float64, frameCount int) {
	for _, p := range pixels.active {
		noiseVal := a.noise.Eval4(p.x, p.y, p.z, time/10.0)
		a.min = math.Min(a.min, noiseVal)
		a.max = math.Max(a.max, noiseVal)

		noiseValNormalized := a.normalizeNoiseValue(noiseVal)
		a.histo.Update(int64(noiseValNormalized * 1000.0))
		c := a.gradient.GetInterpolatedColorFor(noiseValNormalized)
		p.color = &c
		//fmt.Printf("%v\n", noiseVal)
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
	// adjust for the fact that the noise is clustered around the middle
	// 1.92 lines up the Percentile buckets..
	spreader := 1.92
	noiseVal = noiseVal * spreader
	noiseVal = math.Max(a.min, noiseVal)
	noiseVal = math.Min(a.max, noiseVal)
	histoDiff := a.max - a.min
	noiseValDistFromMin := noiseVal - a.min
	return noiseValDistFromMin / histoDiff
}
