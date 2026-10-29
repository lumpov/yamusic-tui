package tracker

import "math"

func compressedByteOffset(totalBytes int64, durationMs, positionMs float64) int64 {
	if durationMs <= 0 || totalBytes <= 0 {
		return 0
	}
	offset := int64(math.Round((float64(totalBytes) / durationMs) * positionMs))
	offset += offset % 4
	return offset
}

func clampByteOffset(offset, totalBytes int64) int64 {
	if offset < 0 {
		return 0
	}
	if totalBytes >= 0 && offset > totalBytes {
		return totalBytes
	}
	return offset
}

func clampPlayableByteOffset(offset, totalBytes, stepBytes int64) int64 {
	if stepBytes < 0 {
		stepBytes = -stepBytes
	}
	max := totalBytes - stepBytes
	if max < 0 {
		max = 0
	}
	if offset < 0 {
		return 0
	}
	if offset > max {
		return max
	}
	return offset
}
