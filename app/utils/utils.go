package utils

import (
	"encoding/json"
	"reflect"
	"vsphere-facade/app/logging"
)

func SliceContain(src, elem interface{}) bool {
	switch reflect.TypeOf(src).Kind() {
	case reflect.Array, reflect.Slice:
	default:
		return false
	}
	values := reflect.ValueOf(src)
	if !values.IsValid() || values.IsZero() {
		return false
	}
	for i := 0; i < values.Len(); i++ {
		if reflect.DeepEqual(values.Index(i).Interface(), elem) {
			return true
		}
	}
	return false
}

func SliceSplit(src interface{}, len int) []interface{} {
	var ret []interface{}
	switch reflect.TypeOf(src).Kind() {
	case reflect.Array, reflect.Slice:
	default:
		return ret
	}
	values := reflect.ValueOf(src)
	for s := 0; s < values.Len(); {
		e := s + len
		if e > values.Len() {
			e = values.Len()
		}

		xx := values.Slice(s, e)
		ret = append(ret, xx.Interface())
		s = e
	}
	return ret
}

func ToJson(data interface{}) string {
	if data == nil {
		return ""
	}

	b, err := json.Marshal(data)
	if err != nil {
		logging.L().Errorf("[%v] to JSON失败: %v", data, err)
		return ""
	}
	return string(b)
}

func NilNext(t interface{}, others ...interface{}) interface{} {
	if IsNil(t) {
		for _, o := range others {
			if !IsNil(o) {
				return o
			}
		}
	} else {
		return t
	}
	return nil
}

func IsNil(i interface{}) bool {
	vi := reflect.ValueOf(i)
	if vi.Kind() == reflect.Ptr {
		return vi.IsNil()
	}
	return false
}

func IsAllNil(l ...interface{}) bool {
	for _, i := range l {
		if !IsNil(i) {
			return false
		}
	}
	return true
}

func HasNil(l ...interface{}) bool {
	for _, i := range l {
		if IsNil(i) {
			return true
		}
	}
	return false
}

func NoNil(l ...interface{}) bool {
	for _, i := range l {
		if IsNil(i) {
			return false
		}
	}
	return true
}

func HasNoneNil(l ...interface{}) bool {
	for _, i := range l {
		if !IsNil(i) {
			return true
		}
	}
	return false
}

func IsEmptyOrNil(i interface{}) bool {
	switch reflect.TypeOf(i).Kind() {
	case reflect.Array, reflect.Slice, reflect.String, reflect.Map, reflect.Chan:
		value := reflect.ValueOf(i)
		return value.Len() == 0
	default:
		value := reflect.ValueOf(i)
		return value.IsZero()
	}
}
