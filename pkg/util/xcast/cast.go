package xcast

import (
	"fmt"
	"reflect"

	"github.com/spf13/cast"
)

// ToInt64Slice casts an interface to []int64.
func ToInt64Slice(i any) []int64 {
	v, _ := ToInt64SliceE(i)
	return v
}

// ToInt64SliceE casts an interface to []int64.
func ToInt64SliceE(i any) ([]int64, error) {
	if i == nil {
		return []int64{}, fmt.Errorf("unable to cast %#v to []int64", i)
	}
	if v, ok := i.([]int64); ok {
		return v, nil
	}
	kind := reflect.TypeOf(i).Kind()
	if kind != reflect.Slice && kind != reflect.Array {
		return []int64{}, fmt.Errorf("unable to cast %#v to []int64", i)
	}
	s := reflect.ValueOf(i)
	out := make([]int64, s.Len())
	for j := 0; j < s.Len(); j++ {
		val, err := cast.ToInt64E(s.Index(j).Interface())
		if err != nil {
			return []int64{}, fmt.Errorf("unable to cast %#v to []int64", i)
		}
		out[j] = val
	}
	return out, nil
}
