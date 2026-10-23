package lib3d

import (
	"math"
	"testing"
)

// TestClosestPointToFace (Port von tests/lib3d_test.cpp).
func TestClosestPointToFace(t *testing.T) {
	f := Face{
		A:      Vec3{X: 0, Y: 0, Z: 0},
		B:      Vec3{X: 10, Y: 0, Z: 0},
		C:      Vec3{X: 0, Y: 0, Z: 10},
		Normal: Vec3{X: 0, Y: 1, Z: 0},
	}

	// Innerer Punkt: Projektion senkrecht nach unten in die Flaechenebene.
	inside := ClosestPointToFace(Vec3{X: 2, Y: 5, Z: 2}, f)
	if !inside.Point.Equals(Vec3{X: 2, Y: 0, Z: 2}, 1e-9) {
		t.Error("innen -> Punkt in der Ebene")
	}
	if math.Abs(inside.Distance-5) > 1e-9 {
		t.Error("innen -> Abstand 5")
	}
	if math.Abs(inside.SignedDistance-5) > 1e-9 {
		t.Error("aussen -> SignedDistance +5")
	}
	if !inside.Normal.Equals(Vec3{X: 0, Y: 1, Z: 0}, 1e-12) {
		t.Error("Normale uebernommen")
	}

	// Unterhalb der Ebene -> negativer gerichteter Abstand.
	below := ClosestPointToFace(Vec3{X: 2, Y: -5, Z: 2}, f)
	if !below.Point.Equals(Vec3{X: 2, Y: 0, Z: 2}, 1e-9) {
		t.Error("unterhalb -> Punkt in der Ebene")
	}
	if math.Abs(below.SignedDistance-(-5)) > 1e-9 {
		t.Error("innen -> SignedDistance -5")
	}

	// Neben der Kante BC (x+z > 10): naechster Punkt auf der Kante.
	edge := ClosestPointToFace(Vec3{X: 6, Y: 0, Z: 6}, f)
	if !edge.Point.Equals(Vec3{X: 5, Y: 0, Z: 5}, 1e-9) {
		t.Error("Kante BC -> Mittelpunkt")
	}
	if math.Abs(edge.Distance-math.Sqrt2) > 1e-9 {
		t.Error("Kante BC -> Abstand sqrt(2)")
	}

	// Neben der Ecke A: naechster Punkt ist die Ecke A.
	corner := ClosestPointToFace(Vec3{X: -4, Y: 0, Z: -4}, f)
	if !corner.Point.Equals(Vec3{X: 0, Y: 0, Z: 0}, 1e-9) {
		t.Error("Ecke A")
	}
	if math.Abs(corner.Distance-math.Sqrt(32)) > 1e-9 {
		t.Error("Ecke A -> Abstand")
	}

	// Senkrecht neben der Kante AC (x < 0).
	ac := ClosestPointToFace(Vec3{X: -3, Y: 0, Z: 5}, f)
	if !ac.Point.Equals(Vec3{X: 0, Y: 0, Z: 5}, 1e-9) {
		t.Error("Kante AC")
	}
	if math.Abs(ac.Distance-3) > 1e-9 {
		t.Error("Kante AC -> Abstand 3")
	}
}

// TestBodyRotVec3 prueft, dass MakeBody die Rotation als Euler-Vec3 uebernimmt
// und dass die Null-Rotation nur die Translation in die Modellmatrix bringt.
func TestBodyRotVec3(t *testing.T) {
	mesh := MakeCubeMesh(2, 2, 2)
	corner := Vec3{X: 1, Y: 0, Z: 0}

	// Ohne Rotationsargument: Rot bleibt der Nullvektor, nur Translation.
	plain := MakeBody(mesh, Vec3{X: 10, Y: 0, Z: 0})
	if plain.Rot != (Vec3{}) {
		t.Error("Default-Rotation ist der Nullvektor")
	}
	if got := corner.Transform(plain.ModelMatrix()); !got.Equals(Vec3{X: 11, Y: 0, Z: 0}, 1e-9) {
		t.Errorf("ohne Rotation nur Translation, got %v", got)
	}

	// Rotation als Vec3: -90 Grad um Y dreht +X nach +Z.
	rotated := MakeBody(mesh, Vec3{X: 10, Y: 0, Z: 0}, Vec3{Y: -math.Pi / 2})
	if !rotated.Rot.Equals(Vec3{Y: -math.Pi / 2}, 1e-12) {
		t.Error("Rotation wird uebernommen")
	}
	if got := corner.Transform(rotated.ModelMatrix()); !got.Equals(Vec3{X: 10, Y: 0, Z: 1}, 1e-9) {
		t.Errorf("RotY(-90) x -> z, got %v", got)
	}
}
