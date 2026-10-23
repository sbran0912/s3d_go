package lib3d

import "math"

// ====================================================================
// BODIES - Flaechen/Normalen und Schnittabfragen (Raycasting)
//
// Teil des lib3d-Pakets. Ermittelt aus der Dreiecksliste eines Meshes die
// Flaechen inklusive Normalen (Faces) und bietet Strecken-/Strahl-Schnitttests
// darauf an. Das Modul zeichnet NICHT; das Rendern bleibt bei lib3d (DrawMesh).
// ====================================================================

// Face ist ein Dreieck eines Meshes samt (normierter) Flaechennormale.
type Face struct {
	A      Vec3
	B      Vec3
	C      Vec3
	Normal Vec3
}

// BuildFaces ermittelt aus der Dreiecksliste (9 Floats je Dreieck, xyz-Tripel)
// eines Meshes die Faces inklusive Normalen. Windungsreihenfolge CCW -> die
// Normale zeigt nach aussen.
func BuildFaces(mesh Mesh) []Face {
	t := mesh.Triangles
	faces := make([]Face, 0, len(t)/9)
	for i := 0; i+8 < len(t); i += 9 {
		var f Face
		f.A = Vec3{X: float64(t[i+0]), Y: float64(t[i+1]), Z: float64(t[i+2])}
		f.B = Vec3{X: float64(t[i+3]), Y: float64(t[i+4]), Z: float64(t[i+5])}
		f.C = Vec3{X: float64(t[i+6]), Y: float64(t[i+7]), Z: float64(t[i+8])}
		f.Normal = f.B.Sub(f.A).Cross(f.C.Sub(f.A)).Normalize()
		faces = append(faces, f)
	}
	return faces
}

// TransformFace bringt ein Face mit einer Modell-Matrix in den Weltraum.
func TransformFace(face Face, model Matrix4x4) Face {
	return Face{
		A:      face.A.Transform(model),
		B:      face.B.Transform(model),
		C:      face.C.Transform(model),
		Normal: face.Normal.TransformDir(model).Normalize(),
	}
}

// TransformFaces bringt Faces mit einer Modell-Matrix in den Weltraum.
func TransformFaces(faces []Face, model Matrix4x4) []Face {
	out := make([]Face, 0, len(faces))
	for _, f := range faces {
		out = append(out, TransformFace(f, model))
	}
	return out
}

// RayHit beschreibt den naechsten Treffer einer Strecke.
type RayHit struct {
	IsHit  bool
	T      float64 // Parameter entlang der Strecke, 0..1
	Point  Vec3    // Trefferpunkt
	Normal Vec3    // Face-Normale am Treffer
}

// rayTriangle ist der Moeller-Trumbore-Schnitt einer Strecke [p1,p2] mit einem
// Dreieck. Liefert true und t (Parameter 0..1) bei einem Treffer.
func rayTriangle(p1, dir, a, b, c Vec3) (float64, bool) {
	e1 := b.Sub(a)
	e2 := c.Sub(a)
	p := dir.Cross(e2)
	det := e1.Dot(p)
	if math.Abs(det) < 1e-12 {
		return 0, false // Strecke parallel zur Face
	}
	inv := 1.0 / det
	tvec := p1.Sub(a)
	u := tvec.Dot(p) * inv
	if u < 0.0 || u > 1.0 {
		return 0, false
	}
	q := tvec.Cross(e1)
	v := dir.Dot(q) * inv
	if v < 0.0 || u+v > 1.0 {
		return 0, false
	}
	t := e2.Dot(q) * inv
	return t, t >= 0.0 && t <= 1.0
}

// RaycastSegment liefert den naechsten Schnitt der Strecke [p1,p2] mit den
// Faces. Ohne Treffer ist IsHit == false.
func RaycastSegment(p1, p2 Vec3, faces []Face) RayHit {
	var best RayHit
	dir := p2.Sub(p1)
	if dir.SquaredLength() <= 0 {
		return best
	}

	bestT := 1.0
	for _, f := range faces {
		t, ok := rayTriangle(p1, dir, f.A, f.B, f.C)
		if !ok {
			continue
		}
		if t > bestT {
			continue
		}
		bestT = t
		best.IsHit = true
		best.T = t
		best.Point = p1.Add(dir.Scale(t))
		best.Normal = f.Normal
	}
	return best
}

// ClosestFaceHit beschreibt den naechsten Punkt auf einem Face zu einem
// Abfragepunkt. Distance ist der ungerichtete Abstand zu Point.
type ClosestFaceHit struct {
	Point          Vec3    // naechster Punkt auf dem Face
	Normal         Vec3    // Flaechennormale (Aussenseite)
	Distance       float64 // Abstand |Point - p| (unsigned)
	SignedDistance float64 // >0 ausserhalb, <0 innerhalb (bzgl. Normal)
}

// closestPointOnTriangle ermittelt den naechsten Punkt auf dem Dreieck (a,b,c)
// zu p (Ericson, Real-Time Collision Detection, 5.1.5).
func closestPointOnTriangle(p, a, b, c Vec3) Vec3 {
	ab := b.Sub(a)
	ac := c.Sub(a)
	ap := p.Sub(a)
	d1 := ab.Dot(ap)
	d2 := ac.Dot(ap)
	if d1 <= 0 && d2 <= 0 {
		return a // Ecke A
	}

	bp := p.Sub(b)
	d3 := ab.Dot(bp)
	d4 := ac.Dot(bp)
	if d3 >= 0 && d4 <= d3 {
		return b // Ecke B
	}

	vc := d1*d4 - d3*d2
	if vc <= 0 && d1 >= 0 && d3 <= 0 {
		return a.Add(ab.Scale(d1 / (d1 - d3))) // Kante AB
	}

	cp := p.Sub(c)
	d5 := ab.Dot(cp)
	d6 := ac.Dot(cp)
	if d6 >= 0 && d5 <= d6 {
		return c // Ecke C
	}

	vb := d5*d2 - d1*d6
	if vb <= 0 && d2 >= 0 && d6 <= 0 {
		return a.Add(ac.Scale(d2 / (d2 - d6))) // Kante AC
	}

	va := d3*d6 - d5*d4
	if va <= 0 && d4-d3 >= 0 && d5-d6 >= 0 {
		w := (d4 - d3) / ((d4 - d3) + (d5 - d6))
		return b.Add(c.Sub(b).Scale(w)) // Kante BC
	}

	// Inneres: baryzentrische Koordinaten (u, v, w), u = 1 - v - w.
	denom := 1.0 / (va + vb + vc)
	v := vb * denom
	w := vc * denom
	return a.Add(ab.Scale(v)).Add(ac.Scale(w))
}

// ClosestPointToFace liefert den Punkt auf dem Face (Dreieck A,B,C), der einem
// Abfragepunkt p am naechsten liegt, samt Normale und Abstand.
func ClosestPointToFace(p Vec3, face Face) ClosestFaceHit {
	var hit ClosestFaceHit
	hit.Point = closestPointOnTriangle(p, face.A, face.B, face.C)
	hit.Normal = face.Normal
	hit.Distance = p.DistanceTo(hit.Point)
	hit.SignedDistance = p.Sub(hit.Point).Dot(face.Normal)
	return hit
}

// Body ist ein platzierbarer, kollisionsfaehiger Koerper: Mesh + lokale Faces
// + Transform. Darstellungsdaten (Farbe, Linienbreite) gehoeren nicht
// hierher, sondern in den ShapeStyle beim Zeichnen (DrawMesh).
type Body struct {
	Geometry Mesh   // Zeichengeometrie (lokal)
	Faces    []Face // lokale Faces (inkl. Normalen)
	Pos      Vec3
	// Rot sind Euler-Winkel in Radiant (X/Y/Z-Achse); RotateMatrix wendet
	// sie in der Reihenfolge Rz*Ry*Rx an. Vec3 ist hier reiner
	// Datentraeger - Length/Normalize/Cross sind auf Winkeln sinnlos.
	Rot Vec3
}

// ModelMatrix ist die Modellmatrix (Translation * Rotation).
func (b Body) ModelMatrix() Matrix4x4 {
	t := TranslateMatrix(b.Pos.X, b.Pos.Y, b.Pos.Z)
	if b.Rot == (Vec3{}) {
		return t
	}
	return MultMatrix(t, RotateMatrix(b.Rot.X, b.Rot.Y, b.Rot.Z))
}

// WorldFaces liefert die Faces in Weltkoordinaten.
func (b Body) WorldFaces() []Face {
	return TransformFaces(b.Faces, b.ModelMatrix())
}

// MakeBody erzeugt einen Body aus einem Mesh. Die Faces werden automatisch
// aus der Geometrie ermittelt (Single Source of Truth). Farbe und
// Linienbreite kommen beim Zeichnen ueber den ShapeStyle (DrawMesh).
// Die optionale Rotation ist ein Vec3 aus Euler-Winkeln in Radiant.
func MakeBody(geometry Mesh, pos Vec3, rot ...Vec3) Body {
	var r Vec3
	if len(rot) > 0 {
		r = rot[0]
	}

	b := Body{
		Geometry: geometry,
		Pos:      pos,
		Rot:      r,
	}
	b.Faces = BuildFaces(b.Geometry)
	return b
}
