package profiler

import "sort"

func UpdateOneHotDistribution24(old [24]float64, index int, alpha float64) [24]float64 {
	var next [24]float64
	for i := range old {
		value := 0.0
		if i == index {
			value = 1.0
		}
		next[i] = EMA(old[i], value, alpha)
	}
	return normalize24(next)
}

func UpdateOneHotDistribution7(old [7]float64, index int, alpha float64) [7]float64 {
	var next [7]float64
	for i := range old {
		value := 0.0
		if i == index {
			value = 1.0
		}
		next[i] = EMA(old[i], value, alpha)
	}
	return normalize7(next)
}

func normalize24(values [24]float64) [24]float64 {
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	if sum == 0 {
		return values
	}
	for i := range values {
		values[i] /= sum
	}
	return values
}

func normalize7(values [7]float64) [7]float64 {
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	if sum == 0 {
		return values
	}
	for i := range values {
		values[i] /= sum
	}
	return values
}

func PeakHours(distribution [24]float64, topN int) []int {
	type pair struct {
		Hour  int
		Value float64
	}
	pairs := make([]pair, 0, len(distribution))
	for hour, value := range distribution {
		pairs = append(pairs, pair{Hour: hour, Value: value})
	}
	sort.SliceStable(pairs, func(i, j int) bool {
		if pairs[i].Value == pairs[j].Value {
			return pairs[i].Hour < pairs[j].Hour
		}
		return pairs[i].Value > pairs[j].Value
	})
	if topN > len(pairs) {
		topN = len(pairs)
	}
	out := make([]int, 0, topN)
	for i := 0; i < topN; i++ {
		out = append(out, pairs[i].Hour)
	}
	return out
}

func ActiveHoursCount(distribution [24]float64, threshold float64) int {
	count := 0
	for _, value := range distribution {
		if value > threshold {
			count++
		}
	}
	return count
}
