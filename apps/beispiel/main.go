package main

import (
	"fmt"
	"os"
	"time"

	"s3dgo/lib3d"
)

// ====================================================================
// Beispiel ohne Physik: Wuerfel, Pyramide und Kugel.
//
// Alle Koerper werden ueber einmal erzeugte Meshes (DrawMesh) gezeichnet.
// Die gesamte Szene dreht sich starr um die Y-Achse - von oben (+Y) gesehen
// im Uhrzeigersinn, also mit fallendem Winkel (angle -= speed).
// ====================================================================

func main() {
	scene := lib3d.NewScene()
	scene.Cam.Pos = lib3d.Vec3{X: 50, Y: 100, Z: 200}
	scene.Cam.Target = lib3d.Vec3{X: 0, Y: 0, Z: 0}
	scene.Cam.Up = lib3d.Vec3{X: 0, Y: 1, Z: 0}
	scene.Cam.FovY = 1.2
	scene.Cam.Near = 0.1
	scene.Cam.Far = 1000
	scene.SunDir = lib3d.Vec3{X: 0.5, Y: 1.0, Z: 0.3}
	scene.SceneFog = lib3d.Fog{Enabled: true, Near: 150, Far: 500, Color: lib3d.ColorState{R: 0.25, G: 0.25, B: 0.25, A: 1}}

	cubeStyle := lib3d.ShapeStyle{
		Stroke:    lib3d.Hex("#1b1b1b"),
		Fill:      lib3d.Hex("#d3ff44"),
		LineWidth: 1,
		Mode:      lib3d.StyleBoth,
	}
	pyramidStyle := lib3d.ShapeStyle{
		Stroke:    lib3d.Hex("#2a1400"),
		Fill:      lib3d.Hex("#ff8844"),
		LineWidth: 1,
		Mode:      lib3d.StyleBoth,
	}
	sphereStyle := lib3d.ShapeStyle{
		Stroke:    lib3d.Hex("#06263f"),
		Fill:      lib3d.Hex("#44aaff"),
		LineWidth: 1,
		Mode:      lib3d.StyleBoth,
	}
	gridStyle := lib3d.ShapeStyle{
		Stroke:    lib3d.Hex("#777774"),
		LineWidth: 1,
	}

	// Feste Positionen (lokal, Weltkoordinaten vor der Szenen-Rotation).
	const cubeX, cubeY, cubeZ, cubeSize = -45.0, 15.0, 0.0, 30.0
	const pyrX, pyrY, pyrZ, pyrBase, pyrHeight = 45.0, 0.0, 0.0, 34.0, 48.0
	const sphX, sphY, sphZ, sphRadius = 0.0, 18.0, -45.0, 16.0

	// Meshes EINMAL erzeugen und im Frame teilen.
	// Kugel-Aufloesung: slices = 24, stacks = 12.
	cubeMesh := lib3d.MakeCubeMesh(cubeSize, cubeSize, cubeSize)
	pyramidMesh := lib3d.MakePyramidMesh(pyrBase, pyrHeight)
	sphereMesh := lib3d.MakeSphereMesh(sphRadius, 24, 12)

	// Bodengitter (fest). EINMAL erzeugen.
	gridMesh := lib3d.MakeGridMesh(600, 24)

	angle := 0.0

	fpsFrames := 0
	fpsLastTime := time.Now()

	updateFps := func() {
		fpsFrames++
		now := time.Now()
		elapsed := now.Sub(fpsLastTime).Milliseconds()
		if elapsed >= 500 {
			fmt.Printf("FPS: %d\n", int(int64(fpsFrames)*1000/elapsed))
			fpsFrames = 0
			fpsLastTime = now
		}
	}

	/* Loop */
	draw := func() {
		updateFps()

		lib3d.BeginScene(scene)

		// Gesamte Szene im Uhrzeigersinn drehen (von oben gesehen).
		angle -= 0.01

		// Bodengitter (fest, XZ-Ebene bei y=0) als Referenz.
		lib3d.DrawMesh(gridMesh, gridStyle)

		// Szenen-Rotation als explizite Modell-Matrix; Position je Koerper dazu.
		sceneRot := lib3d.RotateMatrix(0, angle, 0)
		lib3d.DrawMesh(cubeMesh, cubeStyle, lib3d.MultMatrix(sceneRot, lib3d.TranslateMatrix(cubeX, cubeY, cubeZ)))
		lib3d.DrawMesh(pyramidMesh, pyramidStyle, lib3d.MultMatrix(sceneRot, lib3d.TranslateMatrix(pyrX, pyrY, pyrZ)))
		lib3d.DrawMesh(sphereMesh, sphereStyle, lib3d.MultMatrix(sceneRot, lib3d.TranslateMatrix(sphX, sphY, sphZ)))
	}

	/* Start */
	if err := lib3d.Init(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	lib3d.SetBatchSort(true)

	lib3d.StartAnimation(draw)
}
