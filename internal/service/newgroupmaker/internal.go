package newgroupmaker

import "math"

func (group *Group) CalculateDynamicThresholdLogarithmicish(newsCount int64) float64 {
	const minThreshold = 0.8
	const initialThreshold = 0.85
	const decayRate = 0.1
	return minThreshold + (initialThreshold-minThreshold)*math.Exp(-decayRate*(float64(newsCount)-1))
}
