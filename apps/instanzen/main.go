package main

import (
	"fmt"
	"math"
	"os"
	"time"

	"s3dgo/lib3d"
)

// ====================================================================
// Beispiel: GPU-Instancing.
//
// Ein Raster aus Wuerfeln und ein Ring aus Kugeln werden mit je EINEM
// Instancing-Draw-Call gezeichnet (statt pro Objekt). Jede Instance traegt
// ihre eigene Modell-Matrix und Farbe. Die gesamte Szene dreht sich - von
// oben (+Y) gesehen - im Uhrzeigersinn.
// ====================================================================

func main() {
	scene := lib3d.NewScene()
	scene.Cam.Pos = lib3d.Vec3{X: 120, Y: 240, Z: 460}
	scene.Cam.Target = lib3d.Vec3{X: 0, Y: 0, Z: 0}
	scene.Cam.Up = lib3d.Vec3{X: 0, Y: 1, Z: 0}
	scene.Cam.FovY = 1.2
	scene.Cam.Near = 0.1
	scene.Cam.Far = 2000
	scene.SunDir = lib3d.Vec3{X: 0.5, Y: 1.0, Z: 0.3}
	scene.SceneFog = lib3d.Fog{Enabled: true, Near: 400, Far: 1200, Color: lib3d.ColorState{R: 0.25, G: 0.25, B: 0.25, A: 1}}

	cubeStyle := lib3d.ShapeStyle{
		Stroke:    lib3d.Hex("#101010"),
		Fill:      lib3d.Hex("#ffffff"), // wird pro Instance ueberschrieben
		LineWidth: 1,
		Mode:      lib3d.StyleBoth,
	}
	sphereStyle := lib3d.ShapeStyle{
		Stroke:    lib3d.Hex("#101010"),
		Fill:      lib3d.Hex("#ffffff"),
		LineWidth: 1,
		Mode:      lib3d.StyleBoth,
	}
	gridStyle := lib3d.ShapeStyle{
		Stroke:    lib3d.Hex("#777774"),
		LineWidth: 1,
	}

	// --- Wuerfel-Raster ---
	const gridN = 14
	const gridSpacing = 24.0
	const cubeSize = 12.0
	gridOffset := -float64(gridN-1) * gridSpacing / 2

	cubePos := make([]lib3d.Vec3, 0, gridN*gridN)
	cubeCol := make([]lib3d.ColorState, 0, gridN*gridN)
	for ix := 0; ix < gridN; ix++ {
		for iz := 0; iz < gridN; iz++ {
			px := gridOffset + float64(ix)*gridSpacing
			pz := gridOffset + float64(iz)*gridSpacing
			cubePos = append(cubePos, lib3d.Vec3{X: px, Y: cubeSize / 2, Z: pz})
			t := float64(ix) / float64(gridN-1)
			cubeCol = append(cubeCol, lib3d.ColorState{R: 0.25 + 0.6*t, G: 0.45, B: 0.9 - 0.6*t, A: 1})
		}
	}

	// --- Kugel-Ring ---
	const ringCount = 28
	const ringRadius = 240.0
	const sphereRadius = 11.0
	const sphereY = 34.0

	ringPos := make([]lib3d.Vec3, 0, ringCount)
	ringCol := make([]lib3d.ColorState, 0, ringCount)
	for i := 0; i < ringCount; i++ {
		a := (float64(i) / ringCount) * 2.0 * math.Pi
		ringPos = append(ringPos, lib3d.Vec3{X: ringRadius * math.Cos(a), Y: sphereY, Z: ringRadius * math.Sin(a)})
		t := float64(i) / ringCount
		ringCol = append(ringCol, lib3d.ColorState{R: 0.9 - 0.5*t, G: 0.8*t + 0.2, B: 0.35, A: 1})
	}

	// Meshes EINMAL erzeugen und ueber alle Instanzen teilen.
	// Aufloesung der Kugel: slices = 12, stacks = 6.
	cubeMesh := lib3d.MakeCubeMesh(cubeSize, cubeSize, cubeSize)
	sphereMesh := lib3d.MakeSphereMesh(sphereRadius, 12, 6)

	// Bodengitter EINMAL erzeugen und wiederverwenden.
	gridMesh := lib3d.MakeGridMesh(800, 32)

	angle := 0.0
	cubeInstances := make([]lib3d.Instance, len(cubePos))
	ringInstances := make([]lib3d.Instance, len(ringPos))

	fpsFrames := 0
	fpsLastTime := time.Now()
	updateFps := func() {
		fpsFrames++
		now := time.Now()
		elapsed := now.Sub(fpsLastTime).Milliseconds()
		if elapsed >= 500 {
			fmt.Printf("FPS: %d  Instanzen: %d\n", int(int64(fpsFrames)*1000/elapsed), len(cubeInstances)+len(ringInstances))
			fpsFrames = 0
			fpsLastTime = now
		}
	}

	/* Loop */
	draw := func() {
		updateFps()

		lib3d.BeginScene(scene)

		// Szene im Uhrzeigersinn drehen (von oben gesehen).
		angle -= 0.01

		// Bodengitter (fest) aus dem geteilten Mesh.
		lib3d.DrawMesh(gridMesh, gridStyle)

		// Instanz-Matrizen aktualisieren: Szene-Rotation * Platzierung * Spin.
		for i := range cubePos {
			place := lib3d.TranslateMatrix(cubePos[i].X, cubePos[i].Y, cubePos[i].Z)
			spin := lib3d.RotateMatrix(0, angle*2.0, 0)
			cubeInstances[i].Model = lib3d.MultMatrix(lib3d.MultMatrix(lib3d.RotateMatrix(0, angle, 0), place), spin)
			cubeInstances[i].Color = cubeCol[i]
		}
		for i := range ringPos {
			place := lib3d.TranslateMatrix(ringPos[i].X, ringPos[i].Y, ringPos[i].Z)
			ringInstances[i].Model = lib3d.MultMatrix(lib3d.RotateMatrix(0, angle, 0), place)
			ringInstances[i].Color = ringCol[i]
		}

		// Je EIN Instancing-Draw-Call fuer alle Wuerfel bzw. Kugeln.
		lib3d.DrawMeshInstanced(cubeMesh, cubeStyle, cubeInstances)
		lib3d.DrawMeshInstanced(sphereMesh, sphereStyle, ringInstances)
	}

	/* Start */
	if err := lib3d.Init(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	lib3d.StartAnimation(draw)
}
