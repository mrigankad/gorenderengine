package main

import (
	"fmt"
	stdmath "math"

	"render-engine/core"
	"render-engine/math"
	"render-engine/renderer"
	"render-engine/scene"
)

// dayPalette holds all the sky/light values for one key time of day.
type dayPalette struct {
	t            float32    // normalised time 0..1
	zenith       core.Color // sky overhead
	horizon      core.Color // sky at eye level
	ground       core.Color // sky/ground below horizon
	fogColor     core.Color // fog blend color (should match horizon)
	fogDensity   float32
	sunColor     core.Color
	sunIntensity float32
	ambient      core.Color
}

// palettes defines the key sky/light states throughout the day.
// t is ordered 0→1 and wraps (0 == 1).
var palettes = []dayPalette{
	{ // 0.00 — noon (bright midday)
		t:            0.00,
		zenith:       core.Color{R: 0.20, G: 0.42, B: 0.90, A: 1},
		horizon:      core.Color{R: 0.58, G: 0.75, B: 0.95, A: 1},
		ground:       core.Color{R: 0.12, G: 0.10, B: 0.08, A: 1},
		fogColor:     core.Color{R: 0.62, G: 0.78, B: 0.95, A: 1},
		fogDensity:   0.011,
		sunColor:     core.Color{R: 1.00, G: 0.98, B: 0.92, A: 1},
		sunIntensity: 1.20,
		ambient:      core.Color{R: 0.16, G: 0.18, B: 0.26, A: 1},
	},
	{ // 0.22 — late afternoon / golden hour
		t:            0.22,
		zenith:       core.Color{R: 0.14, G: 0.20, B: 0.60, A: 1},
		horizon:      core.Color{R: 0.90, G: 0.52, B: 0.18, A: 1},
		ground:       core.Color{R: 0.08, G: 0.07, B: 0.06, A: 1},
		fogColor:     core.Color{R: 0.85, G: 0.55, B: 0.25, A: 1},
		fogDensity:   0.018,
		sunColor:     core.Color{R: 1.00, G: 0.65, B: 0.25, A: 1},
		sunIntensity: 0.90,
		ambient:      core.Color{R: 0.10, G: 0.12, B: 0.20, A: 1},
	},
	{ // 0.30 — dusk / twilight
		t:            0.30,
		zenith:       core.Color{R: 0.08, G: 0.10, B: 0.28, A: 1},
		horizon:      core.Color{R: 0.50, G: 0.22, B: 0.28, A: 1},
		ground:       core.Color{R: 0.04, G: 0.03, B: 0.04, A: 1},
		fogColor:     core.Color{R: 0.35, G: 0.18, B: 0.22, A: 1},
		fogDensity:   0.020,
		sunColor:     core.Color{R: 0.70, G: 0.40, B: 0.55, A: 1},
		sunIntensity: 0.25,
		ambient:      core.Color{R: 0.06, G: 0.07, B: 0.14, A: 1},
	},
	{ // 0.50 — midnight
		t:            0.50,
		zenith:       core.Color{R: 0.02, G: 0.03, B: 0.10, A: 1},
		horizon:      core.Color{R: 0.04, G: 0.04, B: 0.08, A: 1},
		ground:       core.Color{R: 0.01, G: 0.01, B: 0.02, A: 1},
		fogColor:     core.Color{R: 0.03, G: 0.03, B: 0.06, A: 1},
		fogDensity:   0.010,
		sunColor:     core.Color{R: 0.40, G: 0.45, B: 0.65, A: 1}, // moonlight
		sunIntensity: 0.12,
		ambient:      core.Color{R: 0.03, G: 0.04, B: 0.09, A: 1},
	},
	{ // 0.70 — pre-dawn
		t:            0.70,
		zenith:       core.Color{R: 0.06, G: 0.08, B: 0.25, A: 1},
		horizon:      core.Color{R: 0.40, G: 0.18, B: 0.24, A: 1},
		ground:       core.Color{R: 0.03, G: 0.03, B: 0.04, A: 1},
		fogColor:     core.Color{R: 0.30, G: 0.15, B: 0.20, A: 1},
		fogDensity:   0.020,
		sunColor:     core.Color{R: 0.75, G: 0.42, B: 0.60, A: 1},
		sunIntensity: 0.20,
		ambient:      core.Color{R: 0.06, G: 0.07, B: 0.14, A: 1},
	},
	{ // 0.78 — sunrise / dawn
		t:            0.78,
		zenith:       core.Color{R: 0.12, G: 0.18, B: 0.55, A: 1},
		horizon:      core.Color{R: 0.88, G: 0.45, B: 0.22, A: 1},
		ground:       core.Color{R: 0.08, G: 0.06, B: 0.05, A: 1},
		fogColor:     core.Color{R: 0.75, G: 0.40, B: 0.20, A: 1},
		fogDensity:   0.015,
		sunColor:     core.Color{R: 1.00, G: 0.60, B: 0.28, A: 1},
		sunIntensity: 0.70,
		ambient:      core.Color{R: 0.09, G: 0.10, B: 0.17, A: 1},
	},
}

// DayNight drives the animated day/night cycle.
type DayNight struct {
	Time   float32 // 0..1: 0=noon, 0.25=sunset, 0.5=midnight, 0.75=sunrise
	Speed  float32 // full-cycle duration in seconds (default 120)
	Active bool    // auto-advance when true
}

func NewDayNight() *DayNight {
	return &DayNight{
		Time:   0.0, // start at noon
		Speed:  120.0,
		Active: true,
	}
}

func (dn *DayNight) Update(dt float32) {
	if !dn.Active {
		return
	}
	dn.Time += dt / dn.Speed
	if dn.Time > 1.0 {
		dn.Time -= 1.0
	}
}

// lerpColor linearly interpolates between two colours.
func lerpColor(a, b core.Color, t float32) core.Color {
	return core.Color{
		R: a.R + (b.R-a.R)*t,
		G: a.G + (b.G-a.G)*t,
		B: a.B + (b.B-a.B)*t,
		A: 1,
	}
}

// samplePalette returns a linearly interpolated palette for the given time t (0..1).
func samplePalette(t float32) dayPalette {
	n := len(palettes)
	// Find the two surrounding keyframes (wrap-around between last and first)
	var a, b dayPalette
	var localT float32
	for i := 0; i < n; i++ {
		next := (i + 1) % n
		ta := palettes[i].t
		tb := palettes[next].t
		if next == 0 {
			tb = 1.0 // wrap: last key → noon (1.0 == 0.0)
		}
		// Handle wrap-around segment (last key → first key)
		if next == 0 {
			if t >= ta || t < palettes[0].t {
				a = palettes[i]
				b = palettes[0]
				if t >= ta {
					localT = (t - ta) / (tb - ta)
				} else {
					localT = (t + 1.0 - ta) / (tb - ta)
				}
				break
			}
		} else {
			if t >= ta && t < tb {
				a = palettes[i]
				b = palettes[next]
				localT = (t - ta) / (tb - ta)
				break
			}
		}
	}

	return dayPalette{
		zenith:       lerpColor(a.zenith, b.zenith, localT),
		horizon:      lerpColor(a.horizon, b.horizon, localT),
		ground:       lerpColor(a.ground, b.ground, localT),
		fogColor:     lerpColor(a.fogColor, b.fogColor, localT),
		fogDensity:   a.fogDensity + (b.fogDensity-a.fogDensity)*localT,
		sunColor:     lerpColor(a.sunColor, b.sunColor, localT),
		sunIntensity: a.sunIntensity + (b.sunIntensity-a.sunIntensity)*localT,
		ambient:      lerpColor(a.ambient, b.ambient, localT),
	}
}

// Apply pushes the current time's sky/light state to the render engine and scene.
// sun is the scene's directional light (may be nil — will be skipped).
func (dn *DayNight) Apply(re *renderer.RenderEngine, s *scene.Scene, sun *scene.Light) {
	p := samplePalette(dn.Time)

	// Sun direction: full rotation in the XY plane, tilted along Z
	angle := float64(dn.Time * 2 * stdmath.Pi)
	sunDir := math.Vec3{
		X: float32(stdmath.Sin(angle)),
		Y: -float32(stdmath.Cos(angle)), // -1 = noon (overhead), +1 = midnight
		Z: 0.35,
	}.Normalize()

	if sun != nil {
		sun.Direction = sunDir
		sun.Color     = p.sunColor
		sun.Intensity = p.sunIntensity
	}

	s.Ambient   = p.ambient
	s.SkyColor  = p.horizon // fallback clear color

	re.SetSkyboxColors(p.zenith, p.horizon, p.ground)
	re.SetFog(true, p.fogDensity, p.fogColor)
}

// TimeOfDayStr returns a human-readable time label.
func (dn *DayNight) TimeOfDayStr() string {
	hours := dn.Time * 24.0
	h := int(hours) % 24
	m := int((hours - float32(h)) * 60)
	period := "AM"
	displayH := h
	if h == 0 {
		displayH = 12
	} else if h == 12 {
		period = "PM"
	} else if h > 12 {
		displayH = h - 12
		period = "PM"
	}
	return fmt.Sprintf("%02d:%02d %s", displayH, m, period)
}
