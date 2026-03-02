package scene

import (
	stdmath "math"
	"math/rand"

	"render-engine/core"
	"render-engine/math"
)

// BlendMode controls how particle colours composite with the scene.
type BlendMode int

const (
	BlendAlpha    BlendMode = iota // standard alpha blend (smoke, mist, dust)
	BlendAdditive                   // additive blend (fire, sparks, glow, magic)
)

// Particle is a single live particle instance.
type Particle struct {
	Position math.Vec3
	Velocity math.Vec3
	Life     float32    // remaining lifetime in seconds
	MaxLife  float32    // total initial lifetime in seconds
	Size     float32    // world-space billboard half-size
	Color    core.Color // updated each frame by lerping StartColor→EndColor
}

// ParticleEmitter spawns and simulates CPU particles.
// Build quads on CPU; rendered as camera-facing billboards via DrawParticles.
type ParticleEmitter struct {
	// Spawn position + direction
	Position  math.Vec3
	Direction math.Vec3 // mean emission direction (must be normalised)
	Spread    float32   // half-angle cone spread in radians (0 = pencil, π/2 = hemisphere)

	// Spawn rate
	Rate int // particles per second

	// Per-particle random ranges
	MinLife, MaxLife   float32 // lifetime range (seconds)
	MinSpeed, MaxSpeed float32 // initial speed range (units/s)
	MinSize, MaxSize   float32 // billboard half-size range

	// Colour over lifetime: linearly interpolated from birth to death
	StartColor core.Color
	EndColor   core.Color

	// Physics — constant acceleration applied every frame
	Gravity math.Vec3

	// Rendering
	BlendMode BlendMode

	// Control
	Active bool // if false no new particles are spawned; existing ones finish out

	// Live particles (read by the renderer)
	Particles []Particle

	pool       int
	spawnAccum float32
	rng        *rand.Rand
}

// NewParticleEmitter returns a fire-like emitter with sensible defaults.
// Adjust fields before the first Update to customise behaviour.
func NewParticleEmitter(maxParticles int) *ParticleEmitter {
	return &ParticleEmitter{
		Direction:  math.Vec3{X: 0, Y: 1, Z: 0},
		Spread:     0.4,
		Rate:       80,
		MinLife:    0.6,
		MaxLife:    1.8,
		MinSpeed:   2.0,
		MaxSpeed:   5.0,
		MinSize:    0.06,
		MaxSize:    0.22,
		StartColor: core.Color{R: 1.0, G: 0.7, B: 0.15, A: 1.0},
		EndColor:   core.Color{R: 0.8, G: 0.05, B: 0.0, A: 0.0},
		Gravity:    math.Vec3{Y: 0.3},
		BlendMode:  BlendAdditive,
		Active:     true,
		Particles:  make([]Particle, 0, maxParticles),
		pool:       maxParticles,
		rng:        rand.New(rand.NewSource(42)),
	}
}

// NewSmokeEmitter returns a slow rising smoke emitter.
func NewSmokeEmitter(maxParticles int) *ParticleEmitter {
	return &ParticleEmitter{
		Direction:  math.Vec3{X: 0, Y: 1, Z: 0},
		Spread:     0.5,
		Rate:       20,
		MinLife:    2.0,
		MaxLife:    4.0,
		MinSpeed:   0.5,
		MaxSpeed:   1.5,
		MinSize:    0.15,
		MaxSize:    0.5,
		StartColor: core.Color{R: 0.3, G: 0.3, B: 0.3, A: 0.4},
		EndColor:   core.Color{R: 0.6, G: 0.6, B: 0.6, A: 0.0},
		Gravity:    math.Vec3{Y: 0.1},
		BlendMode:  BlendAlpha,
		Active:     true,
		Particles:  make([]Particle, 0, maxParticles),
		pool:       maxParticles,
		rng:        rand.New(rand.NewSource(99)),
	}
}

// Update advances the simulation by dt seconds.
// Call once per frame before DrawParticles.
func (e *ParticleEmitter) Update(dt float32) {
	// Spawn new particles
	if e.Active {
		e.spawnAccum += float32(e.Rate) * dt
		for e.spawnAccum >= 1.0 && len(e.Particles) < e.pool {
			e.spawnParticle()
			e.spawnAccum -= 1.0
		}
	}

	// Integrate and cull dead particles (compact in-place)
	write := 0
	for i := range e.Particles {
		p := &e.Particles[i]
		p.Life -= dt
		if p.Life <= 0 {
			continue
		}
		p.Velocity = p.Velocity.Add(e.Gravity.Mul(dt))
		p.Position = p.Position.Add(p.Velocity.Mul(dt))

		t := 1.0 - p.Life/p.MaxLife // 0 = just born, 1 = about to die
		p.Color = lerpColor(e.StartColor, e.EndColor, t)
		p.Size = e.MinSize + (e.MaxSize-e.MinSize)*(1.0-t)

		e.Particles[write] = *p
		write++
	}
	e.Particles = e.Particles[:write]
}

// Count returns the number of live particles.
func (e *ParticleEmitter) Count() int { return len(e.Particles) }

func (e *ParticleEmitter) spawnParticle() {
	life := e.MinLife + e.rng.Float32()*(e.MaxLife-e.MinLife)
	speed := e.MinSpeed + e.rng.Float32()*(e.MaxSpeed-e.MinSpeed)
	dir := randomInCone(e.Direction, e.Spread, e.rng)
	e.Particles = append(e.Particles, Particle{
		Position: e.Position,
		Velocity: dir.Mul(speed),
		Life:     life,
		MaxLife:  life,
		Size:     e.MinSize,
		Color:    e.StartColor,
	})
}

// randomInCone returns a uniformly-distributed unit vector within a cone of
// half-angle spread around axis.  Uses the concentric-disk → spherical cap
// mapping so the distribution is uniform (not polar-biased).
func randomInCone(axis math.Vec3, spread float32, rng *rand.Rand) math.Vec3 {
	phi := rng.Float32() * 2.0 * float32(stdmath.Pi)
	// Uniform distribution over the spherical cap
	cosMin := float32(stdmath.Cos(float64(spread)))
	cosTheta := cosMin + rng.Float32()*(1.0-cosMin)
	sinTheta := float32(stdmath.Sqrt(float64(1.0 - cosTheta*cosTheta)))

	// Build orthonormal frame around axis
	up := math.Vec3{X: 0, Y: 1, Z: 0}
	if stdmath.Abs(float64(axis.Dot(up))) > 0.99 {
		up = math.Vec3{X: 1, Y: 0, Z: 0}
	}
	right := axis.Cross(up).Normalize()
	up = right.Cross(axis).Normalize()

	sinPhi := float32(stdmath.Sin(float64(phi)))
	cosPhi := float32(stdmath.Cos(float64(phi)))
	return axis.Mul(cosTheta).
		Add(right.Mul(sinTheta * cosPhi)).
		Add(up.Mul(sinTheta * sinPhi)).
		Normalize()
}

func lerpColor(a, b core.Color, t float32) core.Color {
	return core.Color{
		R: a.R + (b.R-a.R)*t,
		G: a.G + (b.G-a.G)*t,
		B: a.B + (b.B-a.B)*t,
		A: a.A + (b.A-a.A)*t,
	}
}
