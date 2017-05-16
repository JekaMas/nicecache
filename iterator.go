package nicecache

import "errors"

// chunks returns equals chunks
func chunks(sliceLen, chunkSize int) ([][2]int, error) {
	ranges := [][2]int{}

	if sliceLen < 0 {
		return nil, errors.New("sliceLen should be non-negative")
	}

	if chunkSize <= 0 {
		return nil, errors.New("sliceLen should be positive")
	}

	var prevI int
	for i := 0; i < sliceLen; i++ {
		if sliceLen != 1 && i != sliceLen-1 && ((i+1)%chunkSize != 0 || i == 0) {
			continue
		}

		ranges = append(ranges, [2]int{prevI, i + 1})

		prevI = i + 1
	}

	return ranges, nil
}
