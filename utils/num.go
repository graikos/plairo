package utils

func Max(nums ...int) (max int) {
	if len(nums) == 0 {
		panic("empty slice given to Max function")
	}
	for i, num := range nums {
		if i == 0 || num > max {
			max = num
		}
	}
	return
}
