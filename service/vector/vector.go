package vector

import (
	"math"
	"strconv"
	"strings"
)

type Vector struct {
	vector []float64
}

// ~int | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64
type Number interface {
	~int | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~float32 | ~float64
}

// Return new vector with coordinates from arrays of any numeric type
func New[T Number](vector []T) *Vector {
	if len(vector) == 0 {
		return NewZeroVector(10)
	}
	result := make([]float64, len(vector))
	for i, v := range vector {
		result[i] = float64(v)
	}
	return &Vector{vector: result}
}

// Return new vector with coordinates of 0
func NewZeroVector(capacity int) *Vector {
	return &Vector{vector: make([]float64, capacity)}
}

// Reutrn new vector with coordinates of end vector minus start vector
func NewRadiusVector[T Number](start, end []T) *Vector {
	maxLen := int(math.Max(float64(len(start)), float64(len(end))))
	result := make([]float64, maxLen)

	for i := 0; i < maxLen; i++ {
		var startVal, endVal T
		if i < len(start) {
			startVal = start[i]
		}
		if i < len(end) {
			endVal = end[i]
		}
		result[i] = float64(endVal - startVal)
	}
	return &Vector{vector: result}
}

// Set all vector coordinates to 0
func (v *Vector) Clear() {
	v.vector = make([]float64, 0, v.Capacity())
}

// Check if all vercor coordinates are 0
func (v *Vector) IsPoint() bool {
	return v.Module() == 0
}

// Get length of intermal []float64
func (v *Vector) Capacity() int {
	if v.vector == nil {
		return 0
	}
	return len(v.vector)
}

// Get internal []float64
func (v *Vector) GetArray() []float64 {
	return v.vector
}

// Return square root of sum of squares of all coordinates.
// Equivalent of vector length
func (v *Vector) Module() float64 {
	if v.Capacity() == 0 {
		return 0
	}
	var result float64
	for _, d := range v.vector {
		result += d * d
	}
	return math.Sqrt(result)
}

func (v *Vector) Normalize() *Vector {
	module := v.Module()
	if module == 0 {
		return v // Нормализация невозможна для нулевого вектора
	}
	for i := 0; i < v.Capacity(); i++ {
		v.vector[i] /= module
	}
	return v
}

// Add another vector to this vector
func (v *Vector) Add(other *Vector) *Vector {
	if v.Capacity() == other.Capacity() {
		for i := 0; i < v.Capacity(); i++ {
			v.vector[i] += other.vector[i]
		}
	}
	return v
}

// Subtract another vector from this vector
func (v *Vector) Subtract(other ...*Vector) *Vector {
	for _, d := range other {
		for i := 0; i < v.Capacity(); i++ {
			v.vector[i] -= d.vector[i]
		}
	}
	return v
}

// Multiply all coordinates by a constant
func (v *Vector) Multiply(c float64) *Vector {
	for i := 0; i < v.Capacity(); i++ {
		v.vector[i] *= c
	}
	return v
}

// Divide all coordinates by a constant
func (v *Vector) Divide(c float64) *Vector {
	for i := 0; i < v.Capacity(); i++ {
		v.vector[i] /= c
	}
	return v
}

// Return scalar product of this vector and another vector
func (v *Vector) Scalar(other *Vector) float64 {
	var result float64 = 0
	for i := 0; i < int(math.Min(float64(v.Capacity()), float64(other.Capacity()))); i++ {
		result += v.vector[i] * other.vector[i]
	}
	return result
}

// Check if this vector is equal to another vector (all coordinates are equal)
func (v *Vector) Equals(other *Vector) bool {
	if v.Capacity() != other.Capacity() {
		return false
	}
	for i := 0; i < v.Capacity(); i++ {
		if v.vector[i] != other.vector[i] {
			return false
		}
	}
	return true
}

// Return middle of this vector
func (v *Vector) Middle() []float64 {
	result := make([]float64, v.Capacity())
	for i := 0; i < v.Capacity(); i++ {
		result[i] = v.vector[i] / 2
	}
	return result
}

// Return cosine distance between this vector and another vector.
// Equal cosine of angle between this vector and another vector.
//
// 1 means that the vectors lie on the same straight line and point in the same direction.
// -1 means that the vectors lie on the same straight line and point in the opposite direction.
// 0 means that the vectors are perpendicular.
func (v *Vector) CosDistance(other *Vector) float64 {
	up := v.Scalar(other)
	module1 := v.Module()
	module2 := other.Module()

	if module1 == 0 || module2 == 0 {
		return 0 // Косинусное расстояние не определено для нулевых векторов
	}

	return up / (module1 * module2)
}

func (v *Vector) ToPqString() string {
	if len(v.vector) == 0 {
		return "[]"
	}

	strVec := "["
	strElements := make([]string, len(v.vector))
	for i, val := range v.vector {
		strElements[i] = strconv.FormatFloat(val, 'f', -1, 64)
	}
	strVec += strings.Join(strElements, ",")
	strVec += "]"
	return strVec
}

func (v *Vector) MinkowskiDistance(other *Vector, p float64) float64 {
	sum := 0.0
	if p <= 1 {
		return -1
	}
	if v.Capacity() != other.Capacity() {
		return -1
	}

	// ChebyshevDistance
	if p == math.Inf(1) {
		max := math.Inf(-1)
		for i := 0; i < v.Capacity(); i++ {
			s := math.Abs(v.vector[i] - other.vector[i])
			if s > max {
				max = s
			}
		}
		return max
	}

	for i := 0; i < v.Capacity(); i++ {
		sum += math.Pow(math.Abs(v.vector[i]-other.vector[i]), p)
	}
	return math.Pow(sum, 1/p)
}

func (v *Vector) ManhattanDistance(other *Vector) float64 {
	return v.MinkowskiDistance(other, 1)
}

func (v *Vector) EuclideanDistance(other *Vector) float64 {
	return v.MinkowskiDistance(other, 2)
}

func (v *Vector) ChebyshevDistance(other *Vector) float64 {
	return v.MinkowskiDistance(other, math.Inf(1))
}
