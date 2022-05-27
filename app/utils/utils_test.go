package utils

import (
	"fmt"
	"testing"
)

func TestSliceSplit(t *testing.T) {
	arr := []int{1, 2, 3}
	s := SliceSplit(arr, 1)
	for _, i := range s {
		fmt.Println(i.([]int))
	}
}

func TestC(t *testing.T) {
	n := 16
	d := (30*n - 5) + (n % 4 * 10)
	fmt.Println(d)
}

func TestIsEmptyOrNil(t *testing.T) {
	var arr []string
	fmt.Println(IsEmptyOrNil(arr))
	str := ""
	fmt.Println(IsEmptyOrNil(str))
	var mp = make(map[string]interface{})
	fmt.Println(IsEmptyOrNil(mp))
}
