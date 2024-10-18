package renatsio

import (
	"fmt"
	"math"
)

func imul(a, b int) int {
	const mask = 0xFFFFFFFF
	result := int64(a) * int64(b)
	result = result & mask
	if result > math.MaxInt32 {
		result = result - (mask + 1)
	}
	return int(result)
}
func hashCode(s string) int {
	h := 0
	for i := 0; i < len(s); i++ {
		h = imul(31, h) + int(s[i])
	}
	return int(math.Abs(float64(h)))
}

func calcSubject(hostname string) string {
	TOTAL_SPLITTING_TOPICS := 10
	aggregateIdNumber := hashCode(hostname)
	var splittingTopicNumber string
	if TOTAL_SPLITTING_TOPICS > 0 {
		splittingTopicNumber = fmt.Sprintf("%d", aggregateIdNumber%TOTAL_SPLITTING_TOPICS)
	} else {
		splittingTopicNumber = "0"
	}
	return splittingTopicNumber
}
