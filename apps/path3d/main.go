package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"s3dgo/lib3d"
)

// ====================================================================
// path3d - Grundgeruest fuer spaetere Pfadsuche.
//
// Ein Bodengitter aus 30x30 Feldern. Auf 40 zufaellig gewaehlten Feldern
// steht je ein Wuerfel, der genau ein Feld bedeckt (Blockade). Es werden
// ausschliesslich Meshes verwendet (DrawMesh / DrawMeshInstanced) - keine
// Bodies, keine Physik.
// ====================================================================

const (
	gridCells    = 30                 // Felder pro Achse
	cellSize     = 1.0                // Kantenlaenge eines Feldes
	gridSize     = float64(gridCells) // Gesamtgroesse des Gitters
	cubeSize     = 0.9                // Wuerfelkante (etwas kleiner als das Feld, damit klar bleibt, dass er EIN Feld bedeckt)
	blockedCount = 40                 // Anzahl zufaelliger Blockade-Wuerfel
)

func main() {
	scene := lib3d.NewScene()
	scene.Cam.Pos = lib3d.Vec3{X: 17, Y: 24, Z: 32}
	scene.Cam.Target = lib3d.Vec3{X: 0, Y: 0, Z: 0}
	scene.Cam.Up = lib3d.Vec3{X: 0, Y: 1, Z: 0}
	scene.Cam.Far = 500

	// --- Zufaellige Felder fuer die Blockade-Wuerfel auswaehlen ---
	// map[int]bool garantiert 40 verschiedene Feldindizes (0..899).
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	blocked := make(map[int]bool, blockedCount)
	for len(blocked) < blockedCount {
		blocked[rng.Intn(gridCells*gridCells)] = true
	}

	// Feldindex -> Weltkoordinaten. Das Gitter liegt zentriert um den
	// Ursprung in der XZ-Ebene, die Feldmitte bekommt die +0.5-Verschiebung.
	half := gridSize / 2
	cellCenter := func(i int) (x float64, z float64) {
		x = -half + (float64(i%gridCells)+0.5)*cellSize
		z = -half + (float64(i/gridCells)+0.5)*cellSize
		return x, z
	}

	// Meshes EINMAL erzeugen und wiederverwenden.
	cubeMesh := lib3d.MakeCubeMesh(cubeSize, cubeSize, cubeSize)
	gridMesh := lib3d.MakeGridMesh(gridSize, gridCells)

	gridStyle := lib3d.ShapeStyle{
		Stroke:    lib3d.Hex("#777774"),
		LineWidth: 1,
	}
	cubeStyle := lib3d.ShapeStyle{
		Stroke:    lib3d.Hex("#6b5200"),
		Fill:      lib3d.Hex("#ffffff"), // wird pro Instanz ueberschrieben
		LineWidth: 1,
		Mode:      lib3d.StyleBoth,
	}
	cubeCol := lib3d.ParseHex("#ffcc33")

	// Je Blockade-Feld eine Instanz: Wuerfel stehen auf dem Gitter (y=0),
	// daher liegt ihr Mittelpunkt bei cubeSize/2.
	instances := make([]lib3d.Instance, 0, len(blocked))
	for idx := range blocked {
		x, z := cellCenter(idx)
		instances = append(instances, lib3d.Instance{
			Model: lib3d.TranslateMatrix(x, cubeSize/2, z),
			Color: cubeCol,
		})
	}

	/* Loop */
	draw := func() {
		lib3d.BeginScene(scene)

		// Bodengitter (XZ-Ebene bei y=0) aus dem geteilten Mesh.
		lib3d.DrawMesh(gridMesh, gridStyle)

		// Alle Blockade-Wuerfel in EINEM Instancing-Draw-Call.
		lib3d.DrawMeshInstanced(cubeMesh, cubeStyle, instances)
	}

	/* Start */
	if err := lib3d.Init(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	lib3d.SetBatchSort(true)
	lib3d.StartAnimation(draw)
}
