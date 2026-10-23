package main

import (
	"math"

	"s3dgo/lib3d"
)

// ====================================================================
// VEHICLE – Fahrzeug mit Steering-Behavior (Seek) und DNA
//
// Ein Vehicle ist ein Body (Mesh + Position/Rotation) mit zusaetzlichem
// Physik-Zustand (Geschwindigkeit/Beschleunigung), einer Gesundheit und einer
// DNA. Die DNA steuert, wie stark Gift bzw. gutes Futter angesteuert wird und
// in welchem Radius es ueberhaupt wahrgenommen wird.
// ====================================================================

// Vehicle ist ein Fahrzeug.
type Vehicle struct {
	Body    lib3d.Body
	Vel     lib3d.Vec3
	Accel   lib3d.Vec3
	Heading lib3d.Vec3
	Health  float64
	DNA     [4]float64
}

// NewVehicle erzeugt ein Fahrzeug aus einem Body mit zufaelliger DNA.
func NewVehicle(body lib3d.Body) Vehicle {
	return Vehicle{
		Body:   body,
		Health: 1,
		DNA: [4]float64{
			lib3d.RandomFloat(-1.0, 1.0),  // DNA[0]: Kraft Richtung Gift
			lib3d.RandomFloat(-1.0, 1.0),  // DNA[1]: Kraft Richtung gutes Futter
			lib3d.RandomFloat(20.0, 60.0), // DNA[2]: Radius fuer Gift
			lib3d.RandomFloat(20.0, 60.0), // DNA[3]: Radius fuer gutes Futter
		},
	}
}

// AlignToVelocity passt die Richtung des Bodys an die Geschwindigkeit an.
func (v *Vehicle) AlignToVelocity() {
	mag := v.Vel.Length()
	if mag < 0.0001 {
		return
	}

	magXZ := math.Sqrt(v.Vel.X*v.Vel.X + v.Vel.Z*v.Vel.Z)

	// vel.Y/mag muss in [-1,1] liegen (Schutz vor NaN durch Float-Rundung).
	cosY := lib3d.Constrain(v.Vel.Y/mag, -1.0, 1.0)
	rot := lib3d.Vec3{X: math.Acos(cosY)}
	if magXZ >= 0.0001 {
		rot.Y = math.Atan2(v.Vel.X/magXZ, v.Vel.Z/magXZ)
	}
	v.Body.Rot = rot

	// Heading als normalisierte Richtung ableiten.
	v.Heading = v.Vel.Normalize()
}

// ApplyForce wendet eine Kraft an (akkumuliert in Accel).
func (v *Vehicle) ApplyForce(force lib3d.Vec3) {
	v.Accel = v.Accel.Add(force)
}

// Update ist das Physik-Update: Geschwindigkeit aus Beschleunigung, Position
// aus Geschwindigkeit. Die Geschwindigkeit wird auf [0.5, 2.0] begrenzt.
func (v *Vehicle) Update() {
	v.Vel = v.Vel.Add(v.Accel)
	speed := v.Vel.Length()
	v.Vel = v.Vel.Normalize().Scale(lib3d.Constrain(speed, 0.5, 2.0))
	v.Accel = lib3d.Vec3{}
	v.Body.Pos = v.Body.Pos.Add(v.Vel)
}

// seekForce bestimmt, wie stark das Seek-Steering pro Frame in die
// Beschleunigung eingeht. Kleiner = ruhigeres, traegeres Anfliegen; die
// Geschwindigkeit selbst bleibt davon unberuehrt.
const seekForce = 0.1

// steerLimit ist die Obergrenze der Steuerkraft (Laenge des Steering-Vektors).
// Kleiner = sanftere, weniger scharfe Korrekturen bei grossem Kursfehler,
// waehrend kleine Korrekturen unveraendert bleiben.
const steerLimit = 1.0

// Seek ist das Seek-Verhalten. Die gewuenschte Richtung wird je nach Futter-Art
// mit dem passenden DNA-Wert skaliert (DNA[0] fuer Gift, DNA[1] fuer gutes
// Futter); steerLimit begrenzt die Steuerkraft, seekForce daempft sie.
func (v *Vehicle) Seek(target lib3d.Vec3, isBadFood bool) {
	desired := target.Sub(v.Body.Pos).Limit(3)
	if isBadFood {
		desired = desired.Scale(v.DNA[0])
	} else {
		desired = desired.Scale(v.DNA[1])
	}
	steer := desired.Sub(v.Vel).Limit(steerLimit)
	v.ApplyForce(steer.Scale(seekForce))
}

// EatFood sucht das naechste Futter im DNA-Radius und frisst es, wenn es nah
// genug ist. Gutes Futter -> Health +0.1, Gift -> Health -0.1. Ansonsten wird
// es angesteuert (Seek). Rueckgabe: ggf. gekuerztes Food-Array.
func (v *Vehicle) EatFood(food Food, isBadFood bool) Food {
	filter := v.DNA[3]
	if isBadFood {
		filter = v.DNA[2]
	}
	mindist := math.Inf(1)
	idx := -1

	for i := range food {
		distance := v.Body.Pos.DistanceTo(food[i])
		if distance < filter && distance < mindist {
			mindist = distance
			idx = i
		}
	}

	if idx > -1 {
		if v.Body.Pos.DistanceTo(food[idx]) < 3 {
			// Food aufessen: aus dem Array entfernen genuegt – es gibt keinen
			// GPU-Buffer/Refcount mehr freizugeben (nur CPU-Geometrie).
			food = append(food[:idx], food[idx+1:]...)
			if isBadFood {
				v.Health += -0.1
			} else {
				v.Health += 0.1
			}
		} else {
			v.Seek(food[idx], isBadFood)
		}
	}
	return food
}

// Boundary kehrt die Geschwindigkeit an den Raendern um (Bounce).
func (v *Vehicle) Boundary() {
	const (
		minX, maxX = -130.0, 130.0
		minY, maxY = -130.0, 130.0
		minZ, maxZ = -130.0, 130.0
	)
	if v.Body.Pos.X < minX || v.Body.Pos.X > maxX {
		v.Vel.X *= -1
	}
	if v.Body.Pos.Y < minY || v.Body.Pos.Y > maxY {
		v.Vel.Y *= -1
	}
	if v.Body.Pos.Z < minZ || v.Body.Pos.Z > maxZ {
		v.Vel.Z *= -1
	}
}

// IsDead gibt true zurueck, wenn das Vehicle tot ist (Health < 0).
func (v *Vehicle) IsDead() bool {
	return v.Health < 0
}

// ====================================================================
// FUTTER
// ====================================================================

// Food ist eine Menge von Futterpositionen. Das gemeinsame Mesh wird nicht
// hier verwaltet, daher reichen die Positionen.
type Food []lib3d.Vec3

// NewFood erzeugt count Futterpositionen an Zufallspositionen.
func NewFood(count int) Food {
	food := make(Food, 0, count)
	for i := 0; i < count; i++ {
		food = append(food, randomFoodPos())
	}
	return food
}

// Respawn fuellt das Futter-Array auf: sind weniger als min vorhanden, kommen
// count neue dazu.
func (f Food) Respawn(min, count int) Food {
	if len(f) < min {
		for i := 0; i < count; i++ {
			f = append(f, randomFoodPos())
		}
	}
	return f
}

// ====================================================================
// HILFSFUNKTIONEN
// ====================================================================

// centerMeshY verschiebt ein Mesh entlang Y. MakePyramidMesh legt die Basis bei
// y=0 (Spitze bei y=height); fuer ein um die Position zentriertes Fahrzeug wird
// das Mesh um -height/2 verschoben.
func centerMeshY(mesh lib3d.Mesh, dy float64) lib3d.Mesh {
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

// randomFoodPos liefert eine zufaellige Futterposition in [-100, 100).
func randomFoodPos() lib3d.Vec3 {
	return lib3d.Vec3{
		X: float64(lib3d.Random(-100, 100)),
		Y: float64(lib3d.Random(-100, 100)),
		Z: float64(lib3d.Random(-100, 100)),
	}
}
