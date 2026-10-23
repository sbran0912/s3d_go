package main

import (
	"fmt"
	"os"

	"s3dgo/lib3d"
)

// ====================================================================
// Vehic – Fahrzeug-Schwarm mit Steering-Behaviors und DNA
//
// Neu gebaut auf der aktuellen lib3d-API. Zehn Fahrzeuge (Pyramiden)
// steuern per Seek gutes Futter (gruen) und Gift (rot) an; die DNA bestimmt
// Staerke und Wahrnehmungsradius. Fressen veraendert die Gesundheit, Altern
// (1.5 % pro Frame) zehrt sie auf; bei Health < 0 stirbt ein Fahrzeug.
//
// Modernisierung gegenueber dem Original:
//   - Szene/Kamera/Nebel als Buendel (NewScene/BeginScene).
//   - GPU-Instancing: alle Futterkugeln in EINEM Draw-Call (Farbe pro
//     Instanz), alle Fahrzeuge in EINEM Draw-Call (Farbe nach Health).
//   - Expliziter FlushBatch() zwischen opaker Welt und Debug-Overlays, damit
//     die transparenten DNA-Kreise weiterhin zuletzt (obenauf) gezeichnet
//     werden.
// ====================================================================

func main() {
	scene := lib3d.NewScene()
	scene.Cam.Pos = lib3d.Vec3{X: 50, Y: 20, Z: 200}
	scene.Cam.Target = lib3d.Vec3{X: 0, Y: 0, Z: 0}
	scene.Cam.Up = lib3d.Vec3{X: 0, Y: 1, Z: 0}
	scene.Cam.FovY = 1.2
	scene.Cam.Near = 0.1
	scene.Cam.Far = 1000
	scene.SunDir = lib3d.Vec3{X: 0.5, Y: 1.0, Z: 0.3}
	scene.SceneFog = lib3d.Fog{Enabled: true, Near: 100, Far: 400, Color: lib3d.ColorState{R: 0.25, G: 0.25, B: 0.25, A: 1}}

	gridStyle := lib3d.ShapeStyle{
		Stroke:    lib3d.Hex("#777774"),
		LineWidth: 1,
	}
	// Futter/Fahrzeuge: Farbe kommt beim Instancing pro Instance; der Style
	// bestimmt nur Fill+Stroke und Linienbreite.
	foodStyle := lib3d.ShapeStyle{
		LineWidth: 1,
		Mode:      lib3d.StyleBoth,
	}
	vehicleStyle := lib3d.ShapeStyle{
		LineWidth: 1,
		Mode:      lib3d.StyleBoth,
	}

	// Meshes EINMAL erzeugen und ueber alle Instanzen teilen.
	gridMesh := lib3d.MakeGridMesh(600, 24)     // Bodengitter
	foodMesh := lib3d.MakeSphereMesh(3, 16, 16) // Futter-Kugel (Radius 3)

	// Fahrzeug-Pyramide: breit und kompakt (kein duenner Spiess), damit der
	// Koerper aus jeder Blickrichtung als 3D-Form lesbar ist. Basis bei y=0 ->
	// um die Position zentrieren.
	const vehicleBase, vehicleHeight = 6.0, 12.0
	vehicleMesh := centerMeshY(lib3d.MakePyramidMesh(vehicleBase, vehicleHeight), -vehicleHeight/2)

	// Instanz-Farben.
	poisonColor := lib3d.ParseHex("#FF0000")
	foodColor := lib3d.ParseHex("#44ff44")
	vehicleHealthy := lib3d.ParseHex("#ffffff")
	// Schwache Fahrzeuge bewusst NICHT rot (sonst mit dem roten Gift-Futter
	// verwechselbar), sondern orange.
	vehicleWeak := lib3d.ParseHex("#ff8800")

	// 30x Gift (rot) und 30x gutes Futter (gruen).
	poison := NewFood(30)
	food := NewFood(30)

	// 10 Fahrzeuge mit zufaelliger Startgeschwindigkeit.
	vehicles := make([]Vehicle, 0, 10)
	for i := 0; i < 10; i++ {
		v := NewVehicle(lib3d.MakeBody(vehicleMesh, lib3d.Vec3{X: 0, Y: 20, Z: 100}))
		v.Vel = lib3d.Vec3{
			X: lib3d.RandomFloat(-2, 2),
			Y: lib3d.RandomFloat(-2, 2),
			Z: lib3d.RandomFloat(-2, 2),
		}
		vehicles = append(vehicles, v)
	}

	// Wiederverwendete Instanz-Puffer.
	foodInstances := make([]lib3d.Instance, 0, 60)
	vehicleInstances := make([]lib3d.Instance, 0, 10)

	/* Loop */
	draw := func() {
		lib3d.BeginScene(scene)

		// Futter nachwachsen lassen (min 20, +30 pro Respawn).
		food = food.Respawn(20, 30)
		poison = poison.Respawn(20, 30)

		// Altern: mit 1.5% Wahrscheinlichkeit pro Frame altern alle Fahrzeuge.
		getOlder := lib3d.RandomFloat(0, 1) < 0.015

		// Fahrzeuge simulieren (rueckwaerts, damit Entfernen beim Iterieren ok
		// ist).
		for i := len(vehicles) - 1; i >= 0; i-- {
			v := &vehicles[i]

			v.Boundary()
			food = v.EatFood(food, false)    // gutes Futter
			poison = v.EatFood(poison, true) // Gift
			v.AlignToVelocity()
			v.Update()

			if getOlder {
				v.Health -= 0.05
			}

			if v.IsDead() {
				vehicles = append(vehicles[:i], vehicles[i+1:]...)
			}
		}

		// Bodengitter (XZ-Ebene bei y=0) aus dem geteilten Mesh.
		lib3d.DrawMesh(gridMesh, gridStyle)

		// Futter-Instanzen: Gift (rot) + gutes Futter (gruen) in einem Puffer.
		foodInstances = foodInstances[:0]
		for _, p := range poison {
			foodInstances = append(foodInstances, lib3d.Instance{
				Model: lib3d.TranslateMatrix(p.X, p.Y, p.Z),
				Color: poisonColor,
			})
		}
		for _, f := range food {
			foodInstances = append(foodInstances, lib3d.Instance{
				Model: lib3d.TranslateMatrix(f.X, f.Y, f.Z),
				Color: foodColor,
			})
		}

		// Fahrzeug-Instanzen: Farbe nach Gesundheit (rot wenn schwach).
		vehicleInstances = vehicleInstances[:0]
		for i := range vehicles {
			col := vehicleHealthy
			if vehicles[i].Health < 0.5 {
				col = vehicleWeak
			}
			vehicleInstances = append(vehicleInstances, lib3d.Instance{
				Model: vehicles[i].Body.ModelMatrix(),
				Color: col,
			})
		}

		// Je EIN Instancing-Draw-Call fuer Futter bzw. Fahrzeuge.
		lib3d.DrawMeshInstanced(foodMesh, foodStyle, foodInstances)
		lib3d.DrawMeshInstanced(vehicleMesh, vehicleStyle, vehicleInstances)

		// Opake Welt zeichnen, BEVOR die transparenten Debug-Overlays kommen.
		lib3d.FlushBatch()

		// Debug-Overlays (Batched Drawing): Heading-Pfeile + DNA-Radien.
		for i := range vehicles {
			v := &vehicles[i]
			hp := v.Body.Pos

			// Heading-Pfeil: kurze Linie vom Fahrzeug in Fahrtrichtung (kuerzer
			// als der Koerper, damit beides nicht zu einer Linie verschmilzt).
			end := hp.Add(v.Heading.Scale(5))
			arrowStyle := lib3d.PrimitiveStyle{Stroke: lib3d.Hex("#ffffff")}
			if v.Health < 0.5 {
				arrowStyle.Stroke = lib3d.Hex("#ff4444")
			}
			lib3d.DrawLine(hp.X, hp.Y, hp.Z, end.X, end.Y, end.Z, arrowStyle)

			// DNA-Radien als Kreise in der XZ-Ebene. DrawCircle(x,y,z) zeichnet
			// in der XY-Ebene – durch den Tausch (x, z, y) liegt der Kreis flach.
			lib3d.DrawCircle(hp.X, hp.Z, hp.Y, v.DNA[3], lib3d.ShapeStyle{
				Stroke:   lib3d.RGBA(51, 255, 51, 64),
				Segments: 48,
			}) // guter Food-Radius
			lib3d.DrawCircle(hp.X, hp.Z, hp.Y, v.DNA[2], lib3d.ShapeStyle{
				Stroke:   lib3d.RGBA(255, 51, 51, 64),
				Segments: 48,
			}) // Gift-Radius
		}
	}

	/* Start */
	if err := lib3d.Init(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	lib3d.StartAnimation(draw)
}
