package intutils

func ZeroThen(i int, then int) int {
	if i == 0 {
		return then
	}
	return i
}
