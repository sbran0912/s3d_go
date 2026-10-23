package lib3d

import (
	"math"
	"math/rand/v2"
)

// ====================================================================
// HILFS-MATHEMATIK  (Port von lib-3d.ts)
// ====================================================================

// Project projiziert einen 3D-Punkt (im Kamerakoordinatensystem) auf den
// Bildschirm. Fuer 2D-Overlays gedacht.
func Project(fov float64, v Vec3) Vec2 {
	s := fov / (fov + v.Z)
	return Vec2{X: v.X * s, Y: v.Y * s, S: s}
}

// Constrain begrenzt value auf den Bereich [lo, hi].
func Constrain(value, lo, hi float64) float64 {
	return math.Min(hi, math.Max(lo, value))
}

// Random liefert eine ganzzahlige Zufallszahl in [n1, n2).
func Random(n1, n2 int) int {
	return int(math.Floor(RandomFloat(0.0, 1.0)*float64(n2-n1) + float64(n1)))
}

// RandomFloat liefert eine Gleitkomma-Zufallszahl in [lo, hi).
func RandomFloat(lo, hi float64) float64 {
	return rand.Float64()*(hi-lo) + lo
}
