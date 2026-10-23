package main

import (
	"fmt"
	"os"
	"time"

	"s3dgo/lib3d"
)

// ====================================================================
// Vorlage ohne Physik: Bodengitter und eine Kugel. Die Kugel wird ueber ein
// einmal erzeugtes Mesh (DrawMesh) gezeichnet, das Gitter ueber DrawLine.
// ====================================================================

func main() {
	scene := lib3d.NewScene()

	gridStyle := lib3d.ShapeStyle{
		Stroke:    lib3d.Hex("#777774"),
		LineWidth: 1,
	}
	ballStyle := lib3d.ShapeStyle{
		Stroke:    lib3d.Hex("#2f3a00"),
		Fill:      lib3d.Hex("#d3ff44"),
		LineWidth: 1,
		Mode:      lib3d.StyleBoth,
	}

	// Mesh EINMAL erzeugen und wiederverwenden (statt pro Frame).
	// Aufloesung: segments = 20 -> slices = 20, stacks = 10.
	ballMesh := lib3d.MakeSphereMesh(40, 40, 40)

	// Kugel als Body erstellen (wird sich drehen).
	ballBody := lib3d.MakeBody(ballMesh, lib3d.Vec3{X: 0, Y: 40, Z: 0})

	// Bodengitter EINMAL erzeugen und wiederverwenden.
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

		// Bodengitter (XZ-Ebene bei y=0) aus dem geteilten Mesh.
		lib3d.DrawMesh(gridMesh, gridStyle)

		// Kugel rotieren lassen.
		angle += 0.01
		ballBody.Rot.Y = angle
		lib3d.DrawMesh(ballBody.Geometry, ballStyle, ballBody.ModelMatrix())
	}

	/* Start */
	if err := lib3d.Init(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	lib3d.SetBatchSort(true)
	lib3d.StartAnimation(draw)
}
