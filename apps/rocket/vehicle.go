package main

import (
	"math"

	"s3dgo/lib3d"
)

// ====================================================================
// SCHWEBENDE HINDERNISBOX
//
// Ein FloatingBox ist ein Body mit Grundposition, sanfter Auf-und-Ab-
// Bewegung (Sinus) und langsamer Rotation um die Y-Achse. Der Radius ist
// die maximale Entfernung eines Vertices vom Ursprung (Bounding-Sphere-
// Radius) und dient als schneller Vorab-Test.
// ====================================================================

// FloatingBox ist eine schwebende Hindernis-Box.
type FloatingBox struct {
	Body   lib3d.Body
	Base   lib3d.Vec3
	Phase  float64
	Spin   float64
	Radius float64 // Bounding-Sphere-Radius (max. Vertex-Laenge)
}

// NewFloatingBox erstellt eine schwebende Hindernis-Box aus einem Body.
// phase: Phasenversatz des Schwebens, spin: Rotationsgeschwindigkeit.
func NewFloatingBox(b lib3d.Body, phase, spin float64) FloatingBox {
	f := FloatingBox{Body: b, Base: b.Pos, Phase: phase, Spin: spin}

	maxR := 0.0
	t := b.Geometry.Triangles
	for i := 0; i+2 < len(t); i += 9 {
		x := float64(t[i])
		y := float64(t[i+1])
		z := float64(t[i+2])
		l := math.Sqrt(x*x + y*y + z*z)
		if l > maxR {
			maxR = l
		}
	}
	f.Radius = maxR
	return f
}

// Update animiert die Box (Schweben + Rotation). t ist die Zeit in Sekunden.
func (f *FloatingBox) Update(t float64) {
	f.Body.Pos = lib3d.Vec3{
		X: f.Base.X,
		Y: f.Base.Y + math.Sin(t+f.Phase)*6.0,
		Z: f.Base.Z,
	}
	f.Body.Rot.Y += f.Spin
}

// ====================================================================
// VEHICLE (BOID)
//
// Ein Vehicle ist ein Body mit Geschwindigkeit/Beschleunigung samt den
// Flocking-Regeln (Separate, Align, Cohesion) sowie Ausweichen vor den
// Hindernissen (AvoidObstacles) und Reflexion an den Weltgrenzen
// (ApplyBoundary).
// ====================================================================

// Vehicle ist ein Boid.
type Vehicle struct {
	Body    lib3d.Body
	Vel     lib3d.Vec3
	Accel   lib3d.Vec3
	Heading lib3d.Vec3
	Health  float64
	DNA     [4]float64
}

// NewVehicle erzeugt ein Fahrzeug aus einem Body.
func NewVehicle(b lib3d.Body) Vehicle {
	return Vehicle{Body: b, Health: 1.0}
}

// NewVehicleVel erzeugt ein Fahrzeug mit Anfangsgeschwindigkeit.
func NewVehicleVel(b lib3d.Body, v lib3d.Vec3) Vehicle {
	return Vehicle{Body: b, Vel: v, Health: 1.0}
}

// AlignToVelocity richtet den Body an der Geschwindigkeit aus.
func (v *Vehicle) AlignToVelocity() {
	mag := math.Sqrt(v.Vel.X*v.Vel.X + v.Vel.Y*v.Vel.Y + v.Vel.Z*v.Vel.Z)
	if mag < 0.0001 {
		return
	}

	magXZ := math.Sqrt(v.Vel.X*v.Vel.X + v.Vel.Z*v.Vel.Z)

	// Vel.Y/mag muss in [-1,1] liegen (Schutz vor NaN durch Float-Rundung).
	cosY := lib3d.Constrain(v.Vel.Y/mag, -1.0, 1.0)
	rot := lib3d.Vec3{X: math.Acos(cosY)}
	if magXZ >= 0.0001 {
		rot.Y = math.Atan2(v.Vel.X/magXZ, v.Vel.Z/magXZ)
	}
	v.Body.Rot = rot

	v.Heading = lib3d.Vec3{X: v.Vel.X / mag, Y: v.Vel.Y / mag, Z: v.Vel.Z / mag}
}

// ApplyForce addiert eine Kraft zur Beschleunigung.
func (v *Vehicle) ApplyForce(force lib3d.Vec3) {
	v.Accel = v.Accel.Add(force)
}

// Update integriert die Bewegung und begrenzt die Geschwindigkeit.
func (v *Vehicle) Update() {
	v.Vel = v.Vel.Add(v.Accel)
	speed := v.Vel.Length()
	v.Vel = v.Vel.Normalize().Scale(lib3d.Constrain(speed, 0.5, 2.0))
	v.Accel = lib3d.Vec3{}
	v.Body.Pos = v.Body.Pos.Add(v.Vel)
}

// Seek steuert das Fahrzeug in Richtung eines Ziels.
func (v *Vehicle) Seek(target lib3d.Vec3) {
	desired := target.Sub(v.Body.Pos).Limit(3.0)
	steer := desired.Sub(v.Vel).Limit(2.0)
	v.ApplyForce(steer.Scale(0.2))
}

// Align gleicht die eigene Geschwindigkeit an die der Nachbarn an
// (Flocking-Regel: Alignment).
func (v *Vehicle) Align(vehics []Vehicle) {
	const minDistance = 50.0
	var sumVel lib3d.Vec3
	count := 0

	for i := range vehics {
		other := &vehics[i]
		distance := v.Body.Pos.DistanceTo(other.Body.Pos)
		if distance > 0 && distance < minDistance {
			sumVel = sumVel.Add(other.Vel)
			count++
		}
	}

	if count > 0 {
		sumVel = sumVel.Scale(1.0 / float64(count)) // Durchschnittsgeschwindigkeit
		sumVel = sumVel.Mag(0.1)
		steer := sumVel.Sub(v.Vel).Scale(0.1)
		v.ApplyForce(steer)
	}
}

// Separate haelt Abstand zu den Nachbarn (Flocking-Regel: Separation).
func (v *Vehicle) Separate(vehics []Vehicle) {
	const minDistance = 40.0
	var diffSum lib3d.Vec3
	count := 0

	for i := range vehics {
		other := &vehics[i]
		distance := v.Body.Pos.DistanceTo(other.Body.Pos)
		if distance > 0 && distance < minDistance {
			diffSum = diffSum.Add(v.Body.Pos.Sub(other.Body.Pos).Normalize())
			count++
		}
	}

	if count > 0 {
		diffSum = diffSum.Scale(1.0 / float64(count)) // Durchschnitt
		diffSum = diffSum.Mag(0.1)
		steer := diffSum.Sub(v.Vel).Scale(0.1)
		v.ApplyForce(steer)
	}
}

// Cohesion steuert in Richtung des Schwerpunkts der Nachbarn
// (Flocking-Regel: Kohaesion).
func (v *Vehicle) Cohesion(vehics []Vehicle) {
	const minDistance = 50.0
	var sumPos lib3d.Vec3
	count := 0

	for i := range vehics {
		other := &vehics[i]
		distance := v.Body.Pos.DistanceTo(other.Body.Pos)
		if distance > 0 && distance < minDistance {
			sumPos = sumPos.Add(other.Body.Pos)
			count++
		}
	}

	if count > 0 {
		sumPos = sumPos.Scale(1.0 / float64(count)) // Schwerpunkt der Nachbarn
		desired := sumPos.Sub(v.Body.Pos)           // Richtung zur Gruppenmitte
		desired = desired.Mag(0.1)
		steer := desired.Sub(v.Vel).Scale(0.1)
		v.ApplyForce(steer)
	}
}

// AvoidObstacles laesst das Vehicle den schwebenden Boxen per Steering
// ausweichen. margin ist die letzte Absicherung, detect der
// Vorausschau-Radius.
func (v *Vehicle) AvoidObstacles(obstacles []FloatingBox, margin, detect float64) {
	for oi := range obstacles {
		o := &obstacles[oi]

		// Schneller Vorab-Test (Bounding Sphere).
		if v.Body.Pos.DistanceTo(o.Body.Pos) > o.Radius+detect {
			continue
		}

		faces := o.Body.WorldFaces()
		if len(faces) == 0 {
			continue
		}

		// Naechste Flaeche ueber den betragsmaessig kleinsten
		// vorzeichenbehafteten Abstand bestimmen.
		signedDist := 0.0
		var closest, normal lib3d.Vec3
		bestAbs := math.Inf(1)
		for _, f := range faces {
			hit := lib3d.ClosestPointToFace(v.Body.Pos, f)
			absDist := math.Abs(hit.SignedDistance)
			if absDist < bestAbs {
				bestAbs = absDist
				signedDist = hit.SignedDistance
				closest = hit.Point
				normal = hit.Normal
			}
		}

		// ---- Weiches, vorausschauendes Ausweichen ----
		away := v.Body.Pos.Sub(closest)
		if signedDist < 0.0 || away.Length() < 1e-6 {
			away = normal
		}
		away = away.Normalize()

		// near = 1 direkt an der Oberflaeche, 0 am Rand des Vorausschau-Bereichs.
		dist := lib3d.Constrain(signedDist, 0.0, detect)
		near := lib3d.Constrain(1.0-dist/detect, 0.0, 1.0)

		if near > 0.0 {
			const maxSpeed = 2.0 // Zielgeschwindigkeit
			const maxForce = 2.0 // Grenze der Steering-Kraft
			const weight = 0.02  // Einfluss (klein = sanfter)

			desired := away.Mag(maxSpeed)
			steer := desired.Sub(v.Vel).Limit(maxForce)
			v.ApplyForce(steer.Scale(weight * near))
		}

		// ---- Letzte Absicherung gegen Eindringen ----
		// Zum Testen deaktiviert (false): die Boids duerfen dann in das
		// Hindernis eintauchen. Auf true setzen, um die harte Rueckstellung
		// wieder zu aktivieren.
		const hardSafety = false
		if hardSafety && signedDist < margin {
			v.Body.Pos = v.Body.Pos.Add(normal.Scale(margin - signedDist))

			intoVel := v.Vel.Dot(normal)
			if intoVel < 0.0 {
				v.Vel = v.Vel.Sub(normal.Scale(intoVel))
			}
			intoAccel := v.Accel.Dot(normal)
			if intoAccel < 0.0 {
				v.Accel = v.Accel.Sub(normal.Scale(intoAccel))
			}
		}
	}
}

// ApplyBoundary reflektiert die Geschwindigkeit an den Weltgrenzen.
func (v *Vehicle) ApplyBoundary() {
	const minX, maxX = -400.0, 400.0
	const minY, maxY = -400.0, 400.0
	const minZ, maxZ = -400.0, 400.0

	if v.Body.Pos.X < minX || v.Body.Pos.X > maxX {
		v.Vel.X *= -1.0
	}
	if v.Body.Pos.Y < minY || v.Body.Pos.Y > maxY {
		v.Vel.Y *= -1.0
	}
	if v.Body.Pos.Z < minZ || v.Body.Pos.Z > maxZ {
		v.Vel.Z *= -1.0
	}
}
