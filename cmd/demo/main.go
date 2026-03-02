package main

import (
	"fmt"
	stdmath "math"
	"time"

	"render-engine/core"
	"render-engine/math"
	"render-engine/renderer"
	"render-engine/scene"
)

// collBox is an axis-aligned rectangle in XZ used for player collision.
type collBox struct {
	minX, maxX, minZ, maxZ float32
}

const playerRadius = float32(0.35) // player XZ footprint radius

// resolvePlayerCollision pushes pos outside every overlapping collBox.
func resolvePlayerCollision(pos math.Vec3, boxes []collBox) math.Vec3 {
	for _, b := range boxes {
		eMinX := b.minX - playerRadius
		eMaxX := b.maxX + playerRadius
		eMinZ := b.minZ - playerRadius
		eMaxZ := b.maxZ + playerRadius

		if pos.X <= eMinX || pos.X >= eMaxX || pos.Z <= eMinZ || pos.Z >= eMaxZ {
			continue // no overlap
		}

		// Depth of penetration on each face of the expanded box
		dLeft  := pos.X - eMinX
		dRight := eMaxX - pos.X
		dFront := pos.Z - eMinZ
		dBack  := eMaxZ - pos.Z

		// Push along the axis of minimum penetration
		switch {
		case dLeft <= dRight && dLeft <= dFront && dLeft <= dBack:
			pos.X = eMinX
		case dRight <= dLeft && dRight <= dFront && dRight <= dBack:
			pos.X = eMaxX
		case dFront <= dLeft && dFront <= dRight && dFront <= dBack:
			pos.Z = eMinZ
		default:
			pos.Z = eMaxZ
		}
	}
	return pos
}

// CameraController handles keyboard/mouse input with gravity and ground collision.
type CameraController struct {
	moveSpeed      float32
	lookSpeed      float32
	lastMouseX     float64
	lastMouseY     float64
	firstMouse     bool
	rightMouseDown bool
	yaw            float32
	pitch          float32

	// Physics
	velocityY      float32 // vertical velocity (m/s)
	onGround       bool
	eyeHeight      float32 // camera height above ground
	jumpKeyWasDown bool

	// Collision
	CollBoxes []collBox // world-space XZ AABBs the player cannot walk through
}

const (
	gravity    = -18.0 // m/s²
	jumpSpeed  = 7.0   // initial upward velocity on jump
)

func NewCameraController() *CameraController {
	return &CameraController{
		moveSpeed:  6.0,
		lookSpeed:  0.003,
		firstMouse: true,
		yaw:        -90.0,
		pitch:      0.0,
		eyeHeight:  1.7,
		onGround:   true,
	}
}

func (cc *CameraController) Update(window *core.Window, camera *scene.Camera, deltaTime float32) {
	// Cap deltaTime to avoid huge physics steps on first frames or hitches
	if deltaTime > 0.05 {
		deltaTime = 0.05
	}

	// Mouse look (right mouse drag)
	cc.rightMouseDown = window.IsMouseButtonPressed(1)
	if cc.rightMouseDown {
		mouseX, mouseY := window.GetCursorPos()
		if cc.firstMouse {
			cc.lastMouseX = mouseX
			cc.lastMouseY = mouseY
			cc.firstMouse = false
		}
		cc.yaw   += float32(mouseX-cc.lastMouseX) * cc.lookSpeed
		cc.pitch += float32(cc.lastMouseY-mouseY) * cc.lookSpeed
		if cc.pitch > 88.0  { cc.pitch = 88.0  }
		if cc.pitch < -88.0 { cc.pitch = -88.0 }
		cc.lastMouseX = mouseX
		cc.lastMouseY = mouseY
	} else {
		cc.firstMouse = true
	}

	// Compute view vectors
	yawRad   := cc.yaw   * stdmath.Pi / 180.0
	pitchRad := cc.pitch * stdmath.Pi / 180.0

	forward := math.Vec3{
		X: float32(stdmath.Cos(float64(yawRad)) * stdmath.Cos(float64(pitchRad))),
		Y: float32(stdmath.Sin(float64(pitchRad))),
		Z: float32(stdmath.Sin(float64(yawRad)) * stdmath.Cos(float64(pitchRad))),
	}.Normalize()

	// Horizontal move direction (ignore pitch so strafing stays level)
	moveForward := math.Vec3{
		X: float32(stdmath.Cos(float64(yawRad))),
		Y: 0,
		Z: float32(stdmath.Sin(float64(yawRad))),
	}.Normalize()
	right := math.Vec3{
		X: float32(stdmath.Cos(float64(yawRad - stdmath.Pi/2))),
		Y: 0,
		Z: float32(stdmath.Sin(float64(yawRad - stdmath.Pi/2))),
	}.Normalize()

	// Horizontal movement (WASD)
	hMove := math.Vec3{}
	if window.IsKeyPressed(core.KeyW) { hMove = hMove.Add(moveForward.Mul(cc.moveSpeed * deltaTime)) }
	if window.IsKeyPressed(core.KeyS) { hMove = hMove.Add(moveForward.Mul(-cc.moveSpeed * deltaTime)) }
	if window.IsKeyPressed(core.KeyD) { hMove = hMove.Add(right.Mul(cc.moveSpeed * deltaTime)) }
	if window.IsKeyPressed(core.KeyA) { hMove = hMove.Add(right.Mul(-cc.moveSpeed * deltaTime)) }

	// Jump (Space — debounced so it fires once per press)
	spaceDown := window.IsKeyPressed(core.KeySpace)
	if spaceDown && !cc.jumpKeyWasDown && cc.onGround {
		cc.velocityY = jumpSpeed
		cc.onGround  = false
	}
	cc.jumpKeyWasDown = spaceDown

	// Gravity
	if !cc.onGround {
		cc.velocityY += gravity * deltaTime
	}

	// Vertical position
	newPos := camera.Position.Add(hMove)
	newPos.Y += cc.velocityY * deltaTime

	// Ground collision (eye is at eyeHeight above Y=0 floor)
	groundY := cc.eyeHeight
	if newPos.Y <= groundY {
		newPos.Y    = groundY
		cc.velocityY = 0
		cc.onGround  = true
	}

	// Building/wall collision (XZ push-out)
	newPos = resolvePlayerCollision(newPos, cc.CollBoxes)

	camera.SetPosition(newPos)
	up := forward.Cross(right).Normalize()
	if up.Y < 0 { up.Y = -up.Y } // keep up-vector pointing up
	camera.LookAt(newPos.Add(forward), up)
}

func main() {
	fmt.Println("Starting shapes showcase...")

	windowConfig := core.DefaultWindowConfig()
	windowConfig.Title = "Render Engine - Shapes"
	windowConfig.Width = 1280
	windowConfig.Height = 720

	window, err := core.NewWindow(windowConfig)
	if err != nil {
		fmt.Printf("Failed to create window: %v\n", err)
		return
	}
	defer window.Destroy()

	renderEngine, err := renderer.NewRenderEngine(window)
	if err != nil {
		fmt.Printf("Failed to create render engine: %v\n", err)
		return
	}
	defer renderEngine.Destroy()

	// Enable directional shadow mapping (2048×2048 depth map with PCF)
	if err := renderEngine.EnableShadows(); err != nil {
		fmt.Printf("Shadow map init failed (continuing without shadows): %v\n", err)
	} else {
		fmt.Println("Shadow mapping enabled (2048x2048, PCF 3x3)")
	}

	// Enable HDR post-processing (Reinhard tone mapping + gamma 2.2)
	if err := renderEngine.EnablePostProcess(); err != nil {
		fmt.Printf("Post-process init failed (continuing without it): %v\n", err)
	} else {
		fmt.Println("Post-processing enabled (HDR RGBA16F, Reinhard tone mapping)")
		// Bloom requires post-processing to be active
		if err := renderEngine.EnableBloom(); err != nil {
			fmt.Printf("Bloom init failed (continuing without it): %v\n", err)
		} else {
			fmt.Println("Bloom enabled (bright-pass + 4x Gaussian blur)")
		}
	}

	// Enable SSAO (screen-space ambient occlusion)
	if err := renderEngine.EnableSSAO(); err != nil {
		fmt.Printf("SSAO init failed (continuing without it): %v\n", err)
	} else {
		fmt.Println("SSAO enabled (64-sample hemisphere, 5x5 blur)")
	}

	// Enable procedural gradient skybox
	if err := renderEngine.EnableSkybox(); err != nil {
		fmt.Printf("Skybox init failed (continuing without it): %v\n", err)
	} else {
		fmt.Println("Skybox enabled (procedural gradient: zenith/horizon/ground)")
	}

	// IBL — must be called after EnableSkybox; SetSkyboxColors will sync colours
	renderEngine.EnableIBL()
	fmt.Println("IBL enabled (sky-gradient irradiance for PBR + Phong ambient)")

	// ── Scene setup ───────────────────────────────────────────────────────────
	s := scene.NewScene()
	s.Ambient  = core.Color{R: 0.10, G: 0.12, B: 0.20, A: 1} // cool twilight ambient
	s.SkyColor = core.Color{R: 0.18, G: 0.22, B: 0.50, A: 1}

	camera := scene.NewCamera(float32(stdmath.Pi)/3, 16.0/9.0, 0.1, 500.0)
	camera.SetPosition(math.Vec3{X: 0, Y: 1.7, Z: 12})
	camera.LookAt(math.Vec3{X: 0, Y: 1.7, Z: 0}, math.Vec3Up)
	s.SetCamera(camera)

	// Skybox colors are managed by the DayNight cycle below.

	// ── Materials ─────────────────────────────────────────────────────────────
	matGround := scene.NewMaterial("Ground", core.Color{R: 0.62, G: 0.58, B: 0.52, A: 1})
	matGround.Shininess = 4
	matGround.Specular  = core.Color{R: 0.05, G: 0.05, B: 0.05, A: 1}

	matStone := scene.NewMaterial("Stone", core.Color{R: 0.58, G: 0.55, B: 0.50, A: 1})
	matStone.Shininess = 8

	matBrick := scene.NewMaterial("Brick", core.Color{R: 0.70, G: 0.43, B: 0.30, A: 1})
	matBrick.Shininess = 4

	matPlaster := scene.NewMaterial("Plaster", core.Color{R: 0.90, G: 0.87, B: 0.78, A: 1})
	matPlaster.Shininess = 16

	matRoof := scene.NewMaterial("Roof", core.Color{R: 0.32, G: 0.30, B: 0.28, A: 1})

	matTrunk := scene.NewMaterial("Trunk", core.Color{R: 0.42, G: 0.28, B: 0.13, A: 1})
	matTrunk.Shininess = 4

	matLeaves := scene.NewMaterial("Leaves", core.Color{R: 0.12, G: 0.42, B: 0.15, A: 1})
	matLeaves.Shininess = 4

	// PBR materials
	matMarble := scene.NewPBRMaterial("Marble", core.Color{R: 0.92, G: 0.90, B: 0.86, A: 1}, 0.0, 0.25)
	matWater  := scene.NewPBRMaterial("Water",  core.Color{R: 0.28, G: 0.52, B: 0.72, A: 1}, 0.0, 0.08)
	matMetal  := scene.NewPBRMaterial("Metal",  core.Color{R: 0.14, G: 0.14, B: 0.12, A: 1}, 0.95, 0.15)

	matLamp := scene.NewPBRMaterial("LampGlow", core.Color{R: 1.0, G: 0.85, B: 0.45, A: 1}, 0.0, 0.5)
	matLamp.EmissiveColor = core.Color{R: 3.0, G: 2.0, B: 0.6, A: 1} // bright emissive → bloom

	// PBR materials toggled by P key
	pbrMaterials := []*scene.Material{matMarble, matWater, matMetal, matLamp}

	// ── Helper: place a scaled cube ───────────────────────────────────────────
	addBox := func(name string, pos math.Vec3, sx, sy, sz float32, mat *scene.Material) {
		m := scene.CreateCube(1.0)
		m.Material = mat
		n := scene.NewNode(name)
		n.Mesh = m
		n.SetPosition(pos)
		n.SetScale(math.Vec3{X: sx, Y: sy, Z: sz})
		s.AddNode(n)
	}

	// ── Ground plane ─────────────────────────────────────────────────────────
	groundMesh := scene.CreatePlane(80, 80, 1)
	groundMesh.Material = matGround
	groundNode := scene.NewNode("Ground")
	groundNode.Mesh = groundMesh
	s.AddNode(groundNode)

	gridMesh := scene.CreateGrid(80, 40)
	gridNode := scene.NewNode("Grid")
	gridNode.Mesh = gridMesh
	s.AddNode(gridNode)

	// ── Buildings ─────────────────────────────────────────────────────────────
	// NW — tall stone tower
	addBox("Bldg_NW", math.Vec3{X: -15, Y: 4.5, Z: -15}, 9, 9, 9, matStone)
	addBox("Bldg_NW_roof", math.Vec3{X: -15, Y: 9.5, Z: -15}, 10, 1, 10, matRoof)

	// NE — wide brick block
	addBox("Bldg_NE", math.Vec3{X: 16, Y: 3.5, Z: -15}, 12, 7, 10, matBrick)
	addBox("Bldg_NE_roof", math.Vec3{X: 16, Y: 7.5, Z: -15}, 13, 1, 11, matRoof)

	// SW — cream plaster house
	addBox("Bldg_SW", math.Vec3{X: -15, Y: 3, Z: 16}, 8, 6, 8, matPlaster)
	addBox("Bldg_SW_roof", math.Vec3{X: -15, Y: 6.5, Z: 16}, 9, 1, 9, matRoof)

	// SE — long market hall
	addBox("Bldg_SE", math.Vec3{X: 16, Y: 2.5, Z: 16}, 14, 5, 8, matStone)
	addBox("Bldg_SE_roof", math.Vec3{X: 16, Y: 5.5, Z: 16}, 15, 1, 9, matRoof)

	// Low wall / barrier around the square
	for i, wx := range []float32{-10, 10} {
		wm := scene.CreateCube(1.0)
		wm.Material = matStone
		wn := scene.NewNode(fmt.Sprintf("Wall_%d", i))
		wn.Mesh = wm
		wn.SetPosition(math.Vec3{X: wx, Y: 0.5, Z: 0})
		wn.SetScale(math.Vec3{X: 0.5, Y: 1, Z: 18})
		s.AddNode(wn)
	}

	// ── Fountain (center) ─────────────────────────────────────────────────────
	{
		base := scene.CreateCylinder(3.4, 0.4, 24)
		base.Material = matMarble
		bn := scene.NewNode("Fountain_Base")
		bn.Mesh = base
		bn.SetPosition(math.Vec3{X: 0, Y: 0.2, Z: 0})
		s.AddNode(bn)

		bowl := scene.CreateCylinder(3.0, 0.6, 24)
		bowl.Material = matMarble
		bo := scene.NewNode("Fountain_Bowl")
		bo.Mesh = bowl
		bo.SetPosition(math.Vec3{X: 0, Y: 0.7, Z: 0})
		s.AddNode(bo)

		water := scene.CreateCylinder(2.7, 0.12, 24)
		water.Material = matWater
		wo := scene.NewNode("Fountain_Water")
		wo.Mesh = water
		wo.SetPosition(math.Vec3{X: 0, Y: 0.46, Z: 0})
		s.AddNode(wo)

		pillar := scene.CreateCylinder(0.38, 2.8, 16)
		pillar.Material = matMarble
		pn := scene.NewNode("Fountain_Pillar")
		pn.Mesh = pillar
		pn.SetPosition(math.Vec3{X: 0, Y: 1.4, Z: 0})
		s.AddNode(pn)

		top := scene.CreateSphere(0.5, 16, 8)
		top.Material = matMarble
		tn := scene.NewNode("Fountain_Top")
		tn.Mesh = top
		tn.SetPosition(math.Vec3{X: 0, Y: 3.1, Z: 0})
		s.AddNode(tn)
	}

	// ── Trees ─────────────────────────────────────────────────────────────────
	treePos := []math.Vec3{
		{X: -8, Y: 0, Z: -5}, {X: 8, Y: 0, Z: -6},
		{X: -9, Y: 0, Z: 6},  {X: 9, Y: 0, Z: 5},
		{X: -6, Y: 0, Z: -11},{X: 7, Y: 0, Z: -10},
	}
	for i, tp := range treePos {
		trunk := scene.CreateCylinder(0.22, 2.2, 8)
		trunk.Material = matTrunk
		tn := scene.NewNode(fmt.Sprintf("Trunk%d", i))
		tn.Mesh = trunk
		tn.SetPosition(math.Vec3{X: tp.X, Y: 1.1, Z: tp.Z})
		s.AddNode(tn)

		canopy := scene.CreateCone(1.7, 3.0, 16)
		canopy.Material = matLeaves
		cn := scene.NewNode(fmt.Sprintf("Canopy%d", i))
		cn.Mesh = canopy
		cn.SetPosition(math.Vec3{X: tp.X, Y: 3.1, Z: tp.Z})
		s.AddNode(cn)
	}

	// ── Lamp posts ────────────────────────────────────────────────────────────
	lampPos := []math.Vec3{
		{X: -5.5, Y: 0, Z: -5.5},
		{X: 5.5, Y: 0, Z: -5.5},
		{X: -5.5, Y: 0, Z: 5.5},
		{X: 5.5, Y: 0, Z: 5.5},
	}
	for i, lp := range lampPos {
		pole := scene.CreateCylinder(0.09, 4.8, 8)
		pole.Material = matMetal
		pn := scene.NewNode(fmt.Sprintf("LampPole%d", i))
		pn.Mesh = pole
		pn.SetPosition(math.Vec3{X: lp.X, Y: 2.4, Z: lp.Z})
		s.AddNode(pn)

		cap := scene.CreateSphere(0.28, 12, 6)
		cap.Material = matLamp
		cn := scene.NewNode(fmt.Sprintf("LampCap%d", i))
		cn.Mesh = cap
		cn.SetPosition(math.Vec3{X: lp.X, Y: 4.9, Z: lp.Z})
		s.AddNode(cn)

		s.AddLight(&scene.Light{
			Type:      scene.LightTypePoint,
			Position:  math.Vec3{X: lp.X, Y: 4.7, Z: lp.Z},
			Color:     core.Color{R: 1.0, G: 0.78, B: 0.35, A: 1},
			Intensity: 3.0,
			Range:     14.0,
		})
	}

	// ── Lights ────────────────────────────────────────────────────────────────
	// Sun — direction/color/intensity managed by the DayNight cycle each frame.
	sunLight := &scene.Light{
		Type:      scene.LightTypeDirectional,
		Direction: math.Vec3{X: 0.55, Y: -0.75, Z: -0.35}.Normalize(),
		Color:     core.Color{R: 1.0, G: 0.90, B: 0.70, A: 1},
		Intensity: 1.1,
	}
	s.AddLight(sunLight)

	// ── Particle emitters ──────────────────────────────────────────────────
	// Torch fires at the fountain base (4 corners)
	fireEmitter := scene.NewParticleEmitter(300)
	fireEmitter.Position = math.Vec3{X: 3.5, Y: 0.1, Z: 3.5}

	smokeEmitter := scene.NewSmokeEmitter(80)
	smokeEmitter.Position = math.Vec3{X: 3.5, Y: 0.7, Z: 3.5}

	// Water spray at top of fountain pillar
	magicEmitter := scene.NewParticleEmitter(200)
	magicEmitter.Position   = math.Vec3{X: 0, Y: 3.4, Z: 0}
	magicEmitter.StartColor = core.Color{R: 0.55, G: 0.80, B: 1.0, A: 0.9}
	magicEmitter.EndColor   = core.Color{R: 0.30, G: 0.60, B: 0.90, A: 0.0}
	magicEmitter.Gravity    = math.Vec3{Y: -4.5}
	magicEmitter.Spread     = 1.2
	magicEmitter.MinSpeed   = 2.5
	magicEmitter.MaxSpeed   = 4.5
	magicEmitter.MinLife = 0.6
	magicEmitter.MaxLife = 1.2
	magicEmitter.MinSize = 0.08
	magicEmitter.MaxSize = 0.20

	// Instanced cube mesh (shared geometry, 400 instances)
	instancedCubeMesh := scene.CreateCube(0.5)
	instancedCubeMat := scene.NewMaterial("InstanceCube", core.Color{R: 0.3, G: 0.8, B: 0.4, A: 1})
	instancedCubeMat.Shininess = 48
	instancedCubeMesh.Material = instancedCubeMat

	// Pre-build a 20×20 grid of instance transforms (updated each frame to spin)
	const instCols, instRows = 20, 20
	instanceModels := make([]math.Mat4, instCols*instRows)

	// ── Collision boxes (world-space XZ extents: center ± scale/2) ───────────
	// Buildings: CreateCube(1.0) → ±0.5 each axis, then scaled.
	sceneCollBoxes := []collBox{
		// NW stone tower:  pos(-15,_,-15)  scale(9,_,9)
		{minX: -19.5, maxX: -10.5, minZ: -19.5, maxZ: -10.5},
		// NE brick block:  pos(16,_,-15)   scale(12,_,10)
		{minX: 10.0, maxX: 22.0, minZ: -20.0, maxZ: -10.0},
		// SW plaster house: pos(-15,_,16)  scale(8,_,8)
		{minX: -19.0, maxX: -11.0, minZ: 12.0, maxZ: 20.0},
		// SE market hall:   pos(16,_,16)   scale(14,_,8)
		{minX: 9.0, maxX: 23.0, minZ: 12.0, maxZ: 20.0},
		// West wall:  pos(-10,_,0)  scale(0.5,_,18)
		{minX: -10.25, maxX: -9.75, minZ: -9.0, maxZ: 9.0},
		// East wall:  pos(10,_,0)   scale(0.5,_,18)
		{minX: 9.75, maxX: 10.25, minZ: -9.0, maxZ: 9.0},
		// Fountain bowl (cylinder r=3.0, approximated as square)
		{minX: -3.0, maxX: 3.0, minZ: -3.0, maxZ: 3.0},
	}

	renderEngine.SetScene(s)

	// Day/night cycle — starts at noon (t=0), 120s per full day
	dayNight := NewDayNight()
	dayNight.Apply(renderEngine, s, sunLight) // apply initial sky before first frame

	// Initialize camera controller and HUD
	camController := NewCameraController()
	camController.CollBoxes = sceneCollBoxes
	debugOverlay := &DebugOverlay{}

	frameCount := 0
	displayFPS := 0 // updated each second for HUD
	lastTime := time.Now()
	deltaTime := float32(0.016) // 60 FPS default
	fpsCounter := 0
	fpsLastTime := time.Now()

	fmt.Println("===========================================")
	fmt.Println("  Sonorlax Engine - Shapes Showcase")
	fmt.Println("===========================================")
	fmt.Println("")
	fmt.Println("CAMERA CONTROLS:")
	fmt.Println("  W / S           - Move forward / backward")
	fmt.Println("  A / D           - Strafe left / right")
	fmt.Println("  Space           - Jump")
	fmt.Println("  Right Mouse Drag - Look around")
	fmt.Println("")
	fmt.Println("VIEW TOGGLES:")
	fmt.Println("  Z              - Toggle wireframe mode")
	fmt.Println("  X              - Toggle AABB debug boxes (green wireframe)")
	fmt.Println("  I              - Toggle instanced cube grid (400 cubes, 1 draw call)")
	fmt.Println("  O              - Toggle SSAO (screen-space ambient occlusion)")
	fmt.Println("  P              - Toggle PBR (Cook-Torrance GGX) vs Phong on bottom row")
	fmt.Println("  E              - Toggle particle emitters (fire / smoke / magic)")
	fmt.Println("  N              - Pause / resume day/night cycle")
	fmt.Println("  , / .          - Slow down / speed up day/night cycle")

	fmt.Println("  [ / ]          - Decrease / increase HDR exposure")
	fmt.Println("  B              - Toggle bloom on/off")
	fmt.Println("  - / =          - Decrease / increase bloom strength")
	fmt.Println("  Shadows        - Always on (directional light, PCF 3x3)")
	fmt.Println("  Post-process   - HDR RGBA16F + Reinhard tone mapping")
	fmt.Println("  Skybox         - Procedural gradient sky (zenith/horizon/ground)")
	fmt.Println("  SSAO           - Screen-space ambient occlusion (64 samples + 5x5 blur)")
	fmt.Println("")
	fmt.Println("SCENE:")
	fmt.Println("  F5             - Save scene to scene.json")
	fmt.Println("  F9             - Load scene from scene.json")
	fmt.Println("")
	fmt.Println("EXIT: ESC")
	fmt.Println("===========================================")
	fmt.Println("")

	// Enable frustum culling now that AABBs are visualizable for verification
	renderEngine.FrustumCulling = true

	// Debounce state for toggle keys
	wireframeKeyWasDown  := false
	saveKeyWasDown       := false
	loadKeyWasDown       := false
	bloomKeyWasDown      := false
	aabbKeyWasDown       := false
	instancedKeyWasDown  := false
	ssaoKeyWasDown       := false
	pbrKeyWasDown        := false
	emitterKeyWasDown   := false
	dnKeyWasDown        := false
	const scenePath      = "scene.json"

	// PBR toggle — starts enabled (bottom 3 shapes already have UsePBR=true)
	pbrOn := true

	// Particle emitter toggle
	emittersOn := true

	// Instanced rendering toggle
	instancedOn  := false
	instanceTime := float32(0)

	// SSAO toggle
	ssaoOn       := true
	ssaoStrength := float32(1.0)

	// HDR exposure (adjusted with [ / ] keys)
	exposure := float32(1.0)
	renderEngine.SetExposure(exposure)

	// Bloom strength (adjusted with - / = keys when bloom is on)
	bloomStrength := float32(0.6)
	bloomOn       := true

	for !window.ShouldClose() {
		window.PollEvents()

		if window.IsKeyPressed(core.KeyEscape) {
			break
		}

		// Toggle wireframe on Z key press (debounced)
		zDown := window.IsKeyPressed(core.KeyZ)
		if zDown && !wireframeKeyWasDown {
			renderEngine.SetWireframe(!renderEngine.IsWireframe())
		}
		wireframeKeyWasDown = zDown

		// Save scene: F5
		f5Down := window.IsKeyPressed(core.KeyF5)
		if f5Down && !saveKeyWasDown {
			if err := scene.SaveScene(s, scenePath); err != nil {
				fmt.Printf("[Save] Error: %v\n", err)
			} else {
				fmt.Printf("[Save] Scene saved to %q\n", scenePath)
			}
		}
		saveKeyWasDown = f5Down

		// Exposure: [ to decrease, ] to increase
		if window.IsKeyPressed(core.KeyLeftBracket) {
			exposure -= 0.5 * deltaTime
			if exposure < 0.1 {
				exposure = 0.1
			}
			renderEngine.SetExposure(exposure)
		}
		if window.IsKeyPressed(core.KeyRightBracket) {
			exposure += 0.5 * deltaTime
			if exposure > 5.0 {
				exposure = 5.0
			}
			renderEngine.SetExposure(exposure)
		}

		// X key — toggle AABB wireframe debug draw
		xDown := window.IsKeyPressed(core.KeyX)
		if xDown && !aabbKeyWasDown {
			renderEngine.DrawAABBs = !renderEngine.DrawAABBs
			fmt.Printf("[AABB] %s\n", map[bool]string{true: "ON", false: "OFF"}[renderEngine.DrawAABBs])
		}
		aabbKeyWasDown = xDown

		// B key — toggle bloom on/off
		bDown := window.IsKeyPressed(core.KeyB)
		if bDown && !bloomKeyWasDown {
			bloomOn = !bloomOn
			if bloomOn {
				renderEngine.SetBloomStrength(bloomStrength)
			} else {
				renderEngine.SetBloomStrength(0)
			}
			fmt.Printf("[Bloom] %s\n", map[bool]string{true: "ON", false: "OFF"}[bloomOn])
		}
		bloomKeyWasDown = bDown

		// Bloom strength: - (decrease) / = (increase)
		if bloomOn {
			if window.IsKeyPressed(core.KeyMinus) {
				bloomStrength -= 0.3 * deltaTime
				if bloomStrength < 0 {
					bloomStrength = 0
				}
				renderEngine.SetBloomStrength(bloomStrength)
			}
			if window.IsKeyPressed(core.KeyEqual) {
				bloomStrength += 0.3 * deltaTime
				if bloomStrength > 3.0 {
					bloomStrength = 3.0
				}
				renderEngine.SetBloomStrength(bloomStrength)
			}
		}

		// Load scene: F9 (restores node transforms but not meshes)
		f9Down := window.IsKeyPressed(core.KeyF9)
		if f9Down && !loadKeyWasDown {
			sd, err := scene.LoadScene(scenePath)
			if err != nil {
				fmt.Printf("[Load] Error: %v\n", err)
			} else {
				sd.ApplyToScene(s)
				fmt.Printf("[Load] Scene loaded from %q (%d nodes)\n", scenePath, len(sd.Nodes))
			}
		}
		loadKeyWasDown = f9Down

		// I key — toggle instanced cube grid (20×20 = 400 cubes, 1 draw call)
		iDown := window.IsKeyPressed(core.KeyI)
		if iDown && !instancedKeyWasDown {
			instancedOn = !instancedOn
			fmt.Printf("[Instanced] %s (%d cubes, 1 draw call)\n",
				map[bool]string{true: "ON", false: "OFF"}[instancedOn],
				instCols*instRows)
		}
		instancedKeyWasDown = iDown

		// O key — toggle SSAO on/off
		oDown := window.IsKeyPressed(core.KeyO)
		if oDown && !ssaoKeyWasDown {
			ssaoOn = !ssaoOn
			if ssaoOn {
				renderEngine.SetSSAOStrength(ssaoStrength)
			} else {
				renderEngine.SetSSAOStrength(0)
			}
			fmt.Printf("[SSAO] %s\n", map[bool]string{true: "ON", false: "OFF"}[ssaoOn])
		}
		ssaoKeyWasDown = oDown

		// P key — toggle PBR on the bottom row of shapes
		pDown := window.IsKeyPressed(core.KeyP)
		if pDown && !pbrKeyWasDown {
			pbrOn = !pbrOn
			for _, m := range pbrMaterials {
				m.UsePBR = pbrOn
			}
			fmt.Printf("[PBR] %s\n", map[bool]string{true: "ON", false: "OFF (Phong fallback)"}[pbrOn])
		}
		pbrKeyWasDown = pDown

		// E key — toggle particle emitters
		eDown := window.IsKeyPressed(core.KeyE)
		if eDown && !emitterKeyWasDown {
			emittersOn = !emittersOn
			fireEmitter.Active  = emittersOn
			smokeEmitter.Active = emittersOn
			magicEmitter.Active = emittersOn
			fmt.Printf("[Particles] %s\n", map[bool]string{true: "ON", false: "OFF"}[emittersOn])
		}
		emitterKeyWasDown = eDown

		// N key — pause/resume day/night cycle
		nDown := window.IsKeyPressed(core.KeyN)
		if nDown && !dnKeyWasDown {
			dayNight.Active = !dayNight.Active
			fmt.Printf("[DayNight] %s\n", map[bool]string{true: "RUNNING", false: "PAUSED"}[dayNight.Active])
		}
		dnKeyWasDown = nDown

		// Comma/Period — slow down / speed up the cycle (larger Speed = slower)
		if window.IsKeyPressed(core.KeyComma) {
			dayNight.Speed += 20.0 * deltaTime
			if dayNight.Speed > 600 { dayNight.Speed = 600 }
		}
		if window.IsKeyPressed(core.KeyPeriod) {
			dayNight.Speed -= 20.0 * deltaTime
			if dayNight.Speed < 10 { dayNight.Speed = 10 }
		}

		// Advance cycle and push sky/light state to the renderer
		dayNight.Update(deltaTime)
		dayNight.Apply(renderEngine, s, sunLight)

		// Update camera with controller
		camController.Update(window, camera, deltaTime)

		// Advance instance animation timer
		instanceTime += deltaTime

		// Simulate particles every frame
		fireEmitter.Update(deltaTime)
		smokeEmitter.Update(deltaTime)
		magicEmitter.Update(deltaTime)

		if err := renderEngine.Render(); err != nil {
			width, height := window.GetFramebufferSize()
			if width > 0 && height > 0 {
				renderEngine.Resize(uint32(width), uint32(height))
			}
		}

		// ── Additional draw passes (before Present so they land in the HDR FBO) ──

		// Instanced rendering: 20×20 grid of spinning cubes in one draw call
		if instancedOn {
			for row := 0; row < instRows; row++ {
				for col := 0; col < instCols; col++ {
					x := float32(col-instCols/2) * 1.5
					z := float32(row-instRows/2) * 1.5
					angle := instanceTime*(0.5+float32(col+row)*0.03)
					t := math.Mat4Translation(math.Vec3{X: x, Y: 0.4, Z: z})
					ry := math.Mat4RotationY(angle)
					instanceModels[row*instCols+col] = ry.Mul(t)
				}
			}
			renderEngine.DrawMeshInstanced(instancedCubeMesh, instanceModels)
		}

		// Particle systems — rendered into HDR FBO (benefits from bloom + tone map)
		renderEngine.DrawParticles(fireEmitter)
		renderEngine.DrawParticles(smokeEmitter)
		renderEngine.DrawParticles(magicEmitter)

		// ── Build on-screen HUD (queued, flushed in Present after HDR blit) ──
		objects, verts, tris, culled := renderEngine.DrawStats()
		wireStr := ""
		if renderEngine.IsWireframe() {
			wireStr = " [WIRE]"
		}
		cullingStr := map[bool]string{true: "on", false: "off"}[renderEngine.FrustumCulling]

		debugOverlay.Clear()
		groundStr := map[bool]string{true: "grnd", false: "air"}[camController.onGround]
		debugOverlay.AddLine("FPS: %d   Pos: %.1f  %.1f  %.1f   Yaw: %.0f  Pitch: %.0f  %s%s",
			displayFPS, camera.Position.X, camera.Position.Y, camera.Position.Z,
			camController.yaw, camController.pitch, groundStr, wireStr)
		debugOverlay.AddLine("Draw: obj=%d  verts=%d  tris=%d  culled=%d  (culling %s)",
			objects, verts, tris, culled, cullingStr)
		bloomStatus := map[bool]string{true: fmt.Sprintf("ON  str=%.2f  (- / =)", bloomStrength), false: "OFF"}[bloomOn]
		debugOverlay.AddLine("Exposure: %.2f ([ ])   Bloom: %s (B)   SSAO: %s (O)",
			exposure, bloomStatus, map[bool]string{true: fmt.Sprintf("ON  str=%.2f", ssaoStrength), false: "OFF"}[ssaoOn])
		pbrStatus := map[bool]string{true: "ON (GGX)", false: "OFF (Phong)"}[pbrOn]
		instStatus := map[bool]string{true: fmt.Sprintf("ON %d cubes", instCols*instRows), false: "OFF"}[instancedOn]
		debugOverlay.AddLine("PBR: %s (P)   Instanced: %s (I)", pbrStatus, instStatus)
		ptotal := fireEmitter.Count() + smokeEmitter.Count() + magicEmitter.Count()
		if emittersOn {
			debugOverlay.AddLine("Particles: ON  fire=%d smoke=%d magic=%d total=%d (E)",
				fireEmitter.Count(), smokeEmitter.Count(), magicEmitter.Count(), ptotal)
		} else {
			debugOverlay.AddLine("Particles: OFF (E)")
		}
		dnStatus := map[bool]string{true: "running", false: "PAUSED"}[dayNight.Active]
		debugOverlay.AddLine("Day/Night: %s  Speed: %.0fs/cycle  (N=pause  ,/.=speed)",
			dayNight.TimeOfDayStr()+" "+dnStatus, dayNight.Speed)
		debugOverlay.AddLine("Z=wire  X=AABB  B=bloom  O=ssao  P=pbr  I=inst  E=particles  F5/F9=save/load  N=day/night")

		renderEngine.DrawText(debugOverlay.GetText(), 10, 10, 2, core.ColorWhite)

		// Resolve HDR FBO → screen, flush text overlay, swap buffers
		renderEngine.Present()

		frameCount++
		fpsCounter++
		now := time.Now()
		elapsed := now.Sub(lastTime)
		fpsDelta := now.Sub(fpsLastTime)

		// Update displayFPS and window title each second
		if elapsed.Seconds() >= 1.0 {
			displayFPS = frameCount
			window.SetTitle(fmt.Sprintf("Sonorlax Engine | FPS: %d | (%.1f, %.1f, %.1f)%s",
				frameCount, camera.Position.X, camera.Position.Y, camera.Position.Z, wireStr))
			frameCount = 0
			lastTime = now
		}

		// Periodic console log
		if fpsCounter%60 == 0 {
			fpsRate := float64(fpsCounter) / fpsDelta.Seconds()
			fmt.Printf("[Frame %d] FPS: %.1f | Pos: (%.2f, %.2f, %.2f) | Objs: %d Tris: %d Culled: %d%s\n",
				fpsCounter, fpsRate,
				camera.Position.X, camera.Position.Y, camera.Position.Z,
				objects, tris, culled, wireStr)
			fpsLastTime = now
		}

		deltaTime = float32(elapsed.Seconds())
	}

	renderEngine.WaitIdle()
	fmt.Println("Exiting...")
}
