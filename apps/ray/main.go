package main

import (
	"fmt"
	"math"
	"os"
	"time"

	"s3dgo/lib3d"
)

// ====================================================================
// Rayleigh-Beispiel (Port von simple3d/apps/ray):
//
// Vier Wuerfel (rot/cyan) und eine Pyramide (weiss) stehen in der Szene.
// Aus dem Ursprung laeuft ein rotierender Lichtkegel aus 20 Strahlen; jeder
// Strahl endet am naechsten Face-Treffer (Raycasting ueber lib3d).
// ====================================================================

// shiftMeshY verschiebt ein Mesh entlang Y. Die lib3d-Pyramide liegt mit der
// Basis bei y=0; die Referenz-Pyramide ist vertikal zentriert, daher hier um
// -height/2 verschieben.
func shiftMeshY(mesh lib3d.Mesh, dy float64) lib3d.Mesh {
	out := lib3d.Mesh{
		Triangles: append([]float32(nil), mesh.Triangles...),
		Lines:     append([]float32(nil), mesh.Lines...),
	}
	d := float32(dy)
	for i := 1; i < len(out.Triangles); i += 3 {
		out.Triangles[i] += d
	}
	for i := 1; i < len(out.Lines); i += 3 {
		out.Lines[i] += d
	}
	return out
}

// bodyStyle liefert den Zeichen-Stil eines Bodies. Farbe und Linienbreite
// sind Render-Information und werden daher nicht im Body gespeichert,
// sondern hier in den ShapeStyle gelegt (an DrawMesh uebergeben).
func bodyStyle(color lib3d.ColorState, lineWidth float64) lib3d.ShapeStyle {
	return lib3d.ShapeStyle{
		Stroke:    lib3d.ColorOf(color),
		Fill:      lib3d.ColorOf(color),
		LineWidth: lineWidth,
		Mode:      lib3d.StyleBoth,
	}
}

// SceneBody paart einen Body mit seinen Darstellungsdaten.
type SceneBody struct {
	Object    lib3d.Body
	Color     lib3d.ColorState
	LineWidth float64
}

func main() {
	const coneLines = 20
	const camHeight = 140.0

	scene := lib3d.NewScene()
	scene.SceneFog = lib3d.Fog{Enabled: true, Near: 100, Far: 600, Color: lib3d.ColorState{R: 0.25, G: 0.25, B: 0.25, A: 1}}

	camRadius := math.Sqrt(80*80 + 360*360) // Kamera weiter entfernt
	timeAccum := 0.0
	coneRotY := 0.0
	frameCount := 0
	fpsLast := time.Now()

	// Geometrie: Wuerfel wie lib3d; Pyramide vertikal zentriert (wie Referenz).
	boxMesh := lib3d.MakeCubeMesh(100, 80, 60)
	pyrMesh := shiftMeshY(lib3d.MakePyramidMesh(90, 120), -60)

	red := lib3d.ParseHex("#ff0000")
	cyan := lib3d.ParseHex("#00ffff")
	white := lib3d.ParseHex("#ffffff")

	sceneBodies := []SceneBody{
		{Object: lib3d.MakeBody(boxMesh, lib3d.Vec3{X: 150, Y: 0, Z: 50}), Color: red, LineWidth: 2},
		{Object: lib3d.MakeBody(boxMesh, lib3d.Vec3{X: 0, Y: 0, Z: 100}), Color: cyan, LineWidth: 2},
		{Object: lib3d.MakeBody(boxMesh, lib3d.Vec3{X: -150, Y: 0, Z: -100}), Color: red, LineWidth: 2},
		{Object: lib3d.MakeBody(boxMesh, lib3d.Vec3{X: -200, Y: 0, Z: 30}, lib3d.Vec3{X: math.Pi / 2}), Color: cyan, LineWidth: 2},
		{Object: lib3d.MakeBody(pyrMesh, lib3d.Vec3{X: 100, Y: 0, Z: -100}), Color: white, LineWidth: 1}, // Default: weiss, 1
	}

	/* ----
	Render-Loop - draw wird als Funktion an StartAnimation uebergeben.
	---- */
	draw := func() {
		timeAccum += 0.02
		frameCount++
		now := time.Now()
		if now.Sub(fpsLast).Seconds() >= 2.0 {
			fmt.Printf("FPS: %.1f\n", float64(frameCount)/now.Sub(fpsLast).Seconds())
			frameCount = 0
			fpsLast = now
		}

		camAngle := timeAccum * 0.15

		scene.Cam.Pos = lib3d.Vec3{
			X: math.Sin(camAngle) * camRadius,
			Y: camHeight,
			Z: math.Cos(camAngle) * camRadius,
		}

		lib3d.BeginScene(scene)

		// Alle Bodies zeichnen (Mesh einmal erzeugt, Modell-Matrix explizit,
		// Linienbreite kommt aus dem ShapeStyle).
		for _, b := range sceneBodies {
			lib3d.DrawMesh(b.Object.Geometry, bodyStyle(b.Color, b.LineWidth), b.Object.ModelMatrix())
		}

		// Welt-Faces aller Bodies (pro Frame neu, wie in der Referenz).
		faces := make([]lib3d.Face, 0)
		for _, b := range sceneBodies {
			faces = append(faces, b.Object.WorldFaces()...)
		}

		// Faces debuggen.
		for _, f := range faces {
			fmt.Printf("Face: A=(%.2f, %.2f, %.2f) B=(%.2f, %.2f, %.2f) C=(%.2f, %.2f, %.2f) Normal=(%.2f, %.2f, %.2f)\n",
				f.A.X, f.A.Y, f.A.Z,
				f.B.X, f.B.Y, f.B.Z,
				f.C.X, f.C.Y, f.C.Z,
				f.Normal.X, f.Normal.Y, f.Normal.Z)
		}

		// Lichtkegel rotieren.
		coneRotY += 0.01
		coneRot := lib3d.RotateMatrix(0, coneRotY, 0)

		// --- Rays ---
		apex := lib3d.Vec3{}
		const coneLen = 600.0
		coneAngle := math.Pi / 60
		coneR := coneLen * math.Sin(coneAngle)
		coneZ := coneLen * math.Cos(coneAngle)

		for i := 0; i < coneLines; i++ {
			a := (2.0 * math.Pi * float64(i)) / coneLines
			end := lib3d.Vec3{
				X: apex.X + math.Cos(a)*coneR,
				Y: apex.Y + math.Sin(a)*coneR,
				Z: apex.Z + coneZ,
			}
			rotatedEnd := lib3d.RotateAround(end, apex, coneRot)

			hit := lib3d.RaycastSegment(apex, rotatedEnd, faces)
			endpoint := rotatedEnd
			if hit.IsHit {
				endpoint = hit.Point
			}

			lib3d.DrawLine(apex.X, apex.Y, apex.Z, endpoint.X, endpoint.Y, endpoint.Z,
				lib3d.PrimitiveStyle{Stroke: lib3d.Hex("#ff8800"), LineWidth: 1})
			lib3d.DrawPoint(endpoint.X, endpoint.Y, endpoint.Z,
				lib3d.PrimitiveStyle{Stroke: lib3d.Hex("#ff0000"), PointSize: 5})
		}
	}

	/* Start */
	if err := lib3d.Init(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	lib3d.StartAnimation(draw)
}
