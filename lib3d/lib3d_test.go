package lib3d

import (
	"math"
	"testing"
)

// ====================================================================
// Unit-Test der Bibliothek (Port von tests/lib3d_test.cpp)
// ====================================================================

func nearly(a, b, eps float64) bool { return math.Abs(a-b) < eps }

func TestVec3CrossAndDot(t *testing.T) {
	x := Vec3{X: 1, Y: 0, Z: 0}
	y := Vec3{X: 0, Y: 1, Z: 0}
	if !x.Cross(y).Equals(Vec3{X: 0, Y: 0, Z: 1}, 1e-12) {
		t.Error("cross = (0,0,1)")
	}
	if x.Dot(y) != 0 {
		t.Error("dot = 0")
	}
}

func TestVec3Normalize(t *testing.T) {
	got := Vec3{X: 3, Y: 0, Z: 4}.Normalize()
	if !(nearly(got.Length(), 1, 1e-12) && got.Equals(Vec3{X: 0.6, Y: 0, Z: 0.8}, 1e-12)) {
		t.Error("normalize = (0.6,0,0.8)")
	}
	zero := Vec3{}.Normalize()
	if !zero.Equals(Vec3{}, 1e-12) {
		t.Error("zero vector normalize = zero")
	}
}

func TestLookAtCameraMapsToOrigin(t *testing.T) {
	cam := Vec3{X: 0, Y: 0, Z: -100}
	view := LookAtMatrix(cam, Vec3{X: 0, Y: 0, Z: 0}, Vec3{X: 0, Y: 1, Z: 0})
	got := cam.Transform(view)
	if !got.Equals(Vec3{}, 1e-9) {
		t.Error("Kamera -> Ursprung")
	}
	ahead := Vec3{X: 0, Y: 0, Z: -90}.Transform(view)
	if !nearly(ahead.Z, 10, 1e-9) {
		t.Error("Punkt vor Kamera z = 10")
	}
}

func TestPerspectiveClipspace(t *testing.T) {
	proj := PerspectiveMatrix(math.Pi/4, 1, 0.1, 1000)
	near := 0.1
	v := Vec3{X: 0, Y: 0, Z: near}.Transform(proj)
	if !nearly(v.Z/near, -1, 1e-9) {
		t.Error("near-Ebene z_clip/w = -1")
	}
}

func TestRotateMatrixY(t *testing.T) {
	// +X um -90 Grad um Y drehen -> +Z (Richtung der Szenen-Rotation).
	rot := RotateMatrix(0, -math.Pi/2, 0)
	got := Vec3{X: 1, Y: 0, Z: 0}.Transform(rot)
	if !got.Equals(Vec3{X: 0, Y: 0, Z: 1}, 1e-9) {
		t.Error("RotateY(-90) x -> z")
	}
}

func TestMeshGeometry(t *testing.T) {
	cube := MakeCubeMesh(2, 2, 2)
	if len(cube.Triangles) != 6*2*3*3 {
		t.Error("Wuerfel: 6 Flaechen = 36 Dreiecke")
	}
	if len(cube.Lines) != 12*2*3 {
		t.Error("Wuerfel: 12 Kanten")
	}

	pyr := MakePyramidMesh(2, 3)
	if len(pyr.Triangles) != (4*3+2*3)*3 {
		t.Error("Pyramide: 4 Seiten + Basis")
	}
	if len(pyr.Lines) != 8*2*3 {
		t.Error("Pyramide: 8 Kanten")
	}

	grid := MakeGridMesh(6, 3)
	if len(grid.Triangles) != 0 {
		t.Error("Gitter: keine Dreiecke")
	}
	if len(grid.Lines) != 2*(3+1)*2*3 {
		t.Error("Gitter: 2*(cells+1) Linien")
	}

	gridMin := MakeGridMesh(6, 0)
	if len(gridMin.Lines) != 2*2*2*3 {
		t.Error("Gitter: cells < 1 -> 1 Zelle")
	}
}

func TestScaleMesh(t *testing.T) {
	cube := MakeCubeMesh(2, 2, 2)
	scaled := ScaleMesh(cube, 2.0)

	if len(scaled.Triangles) != len(cube.Triangles) || len(scaled.Lines) != len(cube.Lines) {
		t.Error("ScaleMesh: Anzahl der Vertices bleibt gleich")
	}
	// Erste Wuerfel-Ecke ist (-1,-1,-1) -> skaliert (-2,-2,-2).
	if scaled.Triangles[0] != cube.Triangles[0]*2 || scaled.Triangles[1] != cube.Triangles[1]*2 {
		t.Errorf("ScaleMesh: Triangle-Koordinate nicht skaliert (%v -> %v)",
			cube.Triangles[0], scaled.Triangles[0])
	}
	if scaled.Lines[0] != cube.Lines[0]*2 {
		t.Errorf("ScaleMesh: Line-Koordinate nicht skaliert (%v -> %v)",
			cube.Lines[0], scaled.Lines[0])
	}
	// Eingabe-Mesh bleibt unveraendert.
	if cube.Triangles[0] != -1 {
		t.Errorf("ScaleMesh: Eingabe-Mesh wurde veraendert (%v)", cube.Triangles[0])
	}
}

func TestBatchMerging(t *testing.T) {
	// Ohne Init sammelt submit nur (kein GL-Zugriff).
	batchVerts = batchVerts[:0]
	batchCmds = batchCmds[:0]

	// Sortierung fasst auch nicht-benachbarte Befehle zusammen.
	SetBatchSort(true)
	StrokeColor(ParseColorRGB(200, 200, 200))
	StrokeWidth(1)
	Line(0, 0, 0, 1, 1, 1)
	StrokeWidth(9)
	Line(0, 0, 0, 1, 1, 1)
	StrokeWidth(1)
	Line(0, 0, 0, 1, 1, 1)
	StrokeWidth(9)
	Line(0, 0, 0, 1, 1, 1)
	if got := GetPendingDrawCalls(); got != 4 {
		t.Errorf("w1/w9 im Wechsel -> 4 Draw-Calls, got %d", got)
	}
	CompactBatch()
	if got := GetPendingDrawCalls(); got != 2 {
		t.Errorf("sortiert -> 2 Draw-Calls, got %d", got)
	}
	SetBatchSort(false)

	// 50 identische Linien -> ein einziger Draw-Call.
	StrokeColor(ParseColorRGB(123, 45, 67))
	StrokeWidth(1)
	b1 := GetPendingDrawCalls()
	for i := 0; i < 50; i++ {
		Line(0, 0, 0, 1, 1, 1)
	}
	if got := GetPendingDrawCalls() - b1; got != 1 {
		t.Errorf("50 gleiche Linien -> 1 Draw-Call, got %d", got)
	}

	// Unterschiedliche Farben (flat) werden trotzdem gebuendelt, weil die
	// Farbe pro Vertex uebertragen wird -> keine neuen Draw-Calls.
	b2 := GetPendingDrawCalls()
	DrawLine(0, 0, 0, 1, 0, 0, PrimitiveStyle{Stroke: Hex("#00ff00")})
	DrawLine(0, 0, 0, 1, 0, 0, PrimitiveStyle{Stroke: Hex("#0000ff")})
	if got := GetPendingDrawCalls() - b2; got != 0 {
		t.Errorf("Farbe (flat) bleibt ein Draw-Call, got %d", got)
	}

	// Eine echte Zustandsaenderung (Linienbreite) trennt die Draw-Calls.
	b2b := GetPendingDrawCalls()
	DrawLine(0, 0, 0, 1, 0, 0, PrimitiveStyle{Stroke: Hex("#00ff00"), LineWidth: 5})
	if got := GetPendingDrawCalls() - b2b; got != 1 {
		t.Errorf("Linienbreite trennt Draw-Calls, got %d", got)
	}

	// Ein Wuerfel -> 1 Fill-Call + 1 Line-Call.
	cube := ShapeStyle{
		Stroke:    Hex("#111111"),
		Fill:      Hex("#dddddd"),
		LineWidth: 2,
		Mode:      StyleBoth,
	}
	b3 := GetPendingDrawCalls()
	DrawMesh(MakeCubeMesh(10, 10, 10), cube)
	if got := GetPendingDrawCalls() - b3; got != 2 {
		t.Errorf("Wuerfel -> 1 Fill + 1 Line, got %d", got)
	}
}

func TestInstancing(t *testing.T) {
	// 10 Instanzen eines Wuerfels -> 1 Fill- + 1 Line-Instancing-Draw-Call.
	style := ShapeStyle{
		Stroke:    Hex("#000000"),
		Fill:      Hex("#ffffff"),
		LineWidth: 1,
		Mode:      StyleBoth,
	}
	instances := make([]Instance, 0, 10)
	for i := 0; i < 10; i++ {
		instances = append(instances, Instance{
			Model: TranslateMatrix(float64(i)*5.0, 0, 0),
			Color: ColorState{R: 1, G: 0, B: 0, A: 1},
		})
	}
	b := GetPendingInstancedDrawCalls()
	DrawMeshInstanced(MakeCubeMesh(2, 2, 2), style, instances)
	if got := GetPendingInstancedDrawCalls() - b; got != 2 {
		t.Errorf("10 Instanzen -> 1 Fill + 1 Line (instanced), got %d", got)
	}
}

func TestParseColor(t *testing.T) {
	hex := ParseHex("#ff0000")
	if !(nearly(hex.R, 1, 1e-12) && hex.G == 0 && hex.B == 0 && hex.A == 1) {
		t.Error("hex rot")
	}
	gray := ParseGray(128)
	if !(nearly(gray.R, 128.0/255, 1e-12) && nearly(gray.G, 128.0/255, 1e-12)) {
		t.Error("grau")
	}
	rgba := ParseColorRGBA(51, 255, 51, 64)
	if !(nearly(rgba.G, 1, 1e-12) && nearly(rgba.A, 64.0/255, 1e-12)) {
		t.Error("rgba")
	}
}

// Smoke-Test: DrawMesh sammelt nur in den Batch (kein GL-Zugriff) und darf
// ohne initialisierten Kontext nicht abstuerzen.
func TestMeshShapes(t *testing.T) {
	style := ShapeStyle{
		Stroke:    Hex("#000000"),
		Fill:      Hex("#ffffff"),
		LineWidth: 1,
		Mode:      StyleBoth,
	}
	DrawMesh(MakeCubeMesh(10, 10, 10), style)
	DrawMesh(MakePyramidMesh(10, 15), style)
	DrawMesh(MakeSphereMesh(5, 12, 6), style)
}
