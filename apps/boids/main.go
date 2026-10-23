package main

import (
	"fmt"
	"os"

	"s3dgo/lib3d"
)

// ====================================================================
// Boids-Schwarm
//
// 80 Vehicles starten mit zufaelligen Positionen und Geschwindigkeiten.
// Pro Frame wirken die drei Flocking-Regeln (Separate, Align, Cohesion)
// als Steering-Kraefte; ApplyBoundary reflektiert die Geschwindigkeit an
// den Weltgrenzen, damit der Schwarm im Raum bleibt. Sieben grosse,
// verschiedenfarbige Hindernis-Boxen an festen (deterministischen)
// Positionen werden ueber AvoidObstacles umflogen.
// ====================================================================

func main() {
	scene := lib3d.NewScene()
	scene.Cam.Pos = lib3d.Vec3{X: 0, Y: 320, Z: 520}
	scene.Cam.Target = lib3d.Vec3{X: 0, Y: 0, Z: 0}
	scene.Cam.Up = lib3d.Vec3{X: 0, Y: 1, Z: 0}
	scene.Cam.FovY = 1.2
	scene.Cam.Near = 0.1
	scene.Cam.Far = 2000
	scene.SunDir = lib3d.Vec3{X: 0.5, Y: 1.0, Z: 0.3}
	scene.SceneFog = lib3d.Fog{Enabled: true, Near: 400, Far: 1400, Color: lib3d.ColorState{R: 0.25, G: 0.25, B: 0.25, A: 1}}

	gridStyle := lib3d.ShapeStyle{
		Stroke:    lib3d.Hex("#777774"),
		LineWidth: 1,
	}

	// Bodengitter EINMAL erzeugen und wiederverwenden.
	gridMesh := lib3d.MakeGridMesh(600, 24)
	// Vehicle: Pyramide (quadratische Basis bei y=0, Spitze bei y=height).
	pyramidMesh := lib3d.MakePyramidMesh(5, 10)
	// Hindernis-Box (um den Ursprung zentriert), deutlich groesser als die
	// Vehicles.
	boxMesh := lib3d.MakeCubeMesh(120, 120, 120)

	const obstacleCount = 7

	// Jede Hindernis-Box bekommt eine eigene Fuellfarbe.
	obstacleStyles := [obstacleCount]lib3d.ShapeStyle{
		{Stroke: lib3d.Hex("#101010"), Fill: lib3d.Hex("#3a6ea5"), LineWidth: 2, Mode: lib3d.StyleBoth},
		{Stroke: lib3d.Hex("#101010"), Fill: lib3d.Hex("#c0392b"), LineWidth: 2, Mode: lib3d.StyleBoth},
		{Stroke: lib3d.Hex("#101010"), Fill: lib3d.Hex("#27ae60"), LineWidth: 2, Mode: lib3d.StyleBoth},
		{Stroke: lib3d.Hex("#101010"), Fill: lib3d.Hex("#8e44ad"), LineWidth: 2, Mode: lib3d.StyleBoth},
		{Stroke: lib3d.Hex("#101010"), Fill: lib3d.Hex("#e67e22"), LineWidth: 2, Mode: lib3d.StyleBoth},
		{Stroke: lib3d.Hex("#101010"), Fill: lib3d.Hex("#16a085"), LineWidth: 2, Mode: lib3d.StyleBoth},
		{Stroke: lib3d.Hex("#101010"), Fill: lib3d.Hex("#d4a017"), LineWidth: 2, Mode: lib3d.StyleBoth},
	}

	// Schwebende Hindernisse an FESTEN, deterministischen Positionen.
	obstaclePositions := [obstacleCount]lib3d.Vec3{
		{X: -160.0, Y: -120.0, Z: -240.0},
		{X: 0.0, Y: 120.0, Z: -240.0},
		{X: 160.0, Y: -120.0, Z: -240.0},
		{X: -240.0, Y: 120.0, Z: 0.0},
		{X: 240.0, Y: 120.0, Z: 0.0},
		{X: -240.0, Y: -120.0, Z: 240.0},
		{X: 0.0, Y: -40.0, Z: 240.0},
	}

	obstacles := make([]FloatingBox, 0, obstacleCount)
	for i := 0; i < obstacleCount; i++ {
		obstacles = append(obstacles, NewFloatingBox(lib3d.MakeBody(boxMesh, obstaclePositions[i]), float64(i)*1.3, 0.01))
	}

	const vehicleCount = 80
	// Spawn-Bereich liegt innerhalb der Weltgrenzen aus ApplyBoundary.
	const spawnX = 180.0
	const spawnY = 110.0
	const spawnZ = 110.0

	// Farbe der Vehicles (Render-Info, gehoert nicht in den Body).
	vehicleStyle := lib3d.ShapeStyle{
		Stroke:    lib3d.Hex("#7a3b00"),
		Fill:      lib3d.Hex("#ff8800"),
		LineWidth: 1,
		Mode:      lib3d.StyleBoth,
	}

	// Vehicles starten mit zufaelliger Position und zufaelliger
	// Geschwindigkeit.
	vehicles := make([]Vehicle, 0, vehicleCount)
	for i := 0; i < vehicleCount; i++ {
		pos := lib3d.Vec3{
			X: lib3d.RandomFloat(-spawnX, spawnX),
			Y: lib3d.RandomFloat(-spawnY, spawnY),
			Z: lib3d.RandomFloat(-spawnZ, spawnZ),
		}
		velocity := lib3d.Vec3{
			X: lib3d.RandomFloat(-1.0, 1.0),
			Y: lib3d.RandomFloat(-1.0, 1.0),
			Z: lib3d.RandomFloat(-1.0, 1.0),
		}.Mag(1.0)

		v := NewVehicleVel(lib3d.MakeBody(pyramidMesh, pos), velocity)
		v.AlignToVelocity()
		vehicles = append(vehicles, v)
	}

	timeAccum := 0.0

	/* Loop */
	draw := func() {
		lib3d.BeginScene(scene)

		timeAccum += 0.02

		// Bodengitter (XZ-Ebene bei y=0) aus dem geteilten Mesh.
		lib3d.DrawMesh(gridMesh, gridStyle)

		// Hindernisse animieren (Schweben + Rotation).
		for i := range obstacles {
			obstacles[i].Update(timeAccum)
		}

		// Flocking: erst die Kraefte sammeln, dann integrieren.
		center := scene.Cam.Target
		for i := range vehicles {
			v := &vehicles[i]

			// Bei gedrueckter Maustaste wirkt zusaetzlich eine Kraft zum
			// Szenenzentrum (scene.Cam.Target) auf jedes Boid.
			if lib3d.IsMouseDown() {
				toCenter := center.Sub(v.Body.Pos).Normalize()
				v.ApplyForce(toCenter.Scale(0.05))
			}
			v.Separate(vehicles)
			v.Align(vehicles)
			v.Cohesion(vehicles)
			// margin = harte Sicherheitsdistanz, detect = Vorausschau-Radius
			v.AvoidObstacles(obstacles, 6.0, 30.0)
			v.ApplyBoundary()
			v.Update()
			v.AlignToVelocity()
		}

		for i := range obstacles {
			lib3d.DrawMesh(obstacles[i].Body.Geometry, obstacleStyles[i], obstacles[i].Body.ModelMatrix())
		}
		for i := range vehicles {
			lib3d.DrawMesh(vehicles[i].Body.Geometry, vehicleStyle, vehicles[i].Body.ModelMatrix())
		}
	}

	/* Start */
	if err := lib3d.Init(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	lib3d.SetBatchSort(true)
	lib3d.StartAnimation(draw)
}
