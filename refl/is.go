package refl

import "reflect"

// IsSliceOrMap checks if variable is a slice/array/map or a pointer to it.
func IsSliceOrMap(i interface{}) bool {
	if i == nil {
		return false
	}
	t := DeepIndirect(reflect.TypeOf(i))
	return t.Kind() == reflect.Slice || t.Kind() == reflect.Map || t.Kind() == reflect.Array
}

// IsStruct checks if variable is a struct or a pointer to a struct.
func IsStruct(i interface{}) bool {
	if i == nil {
		return false
	}
	t := reflect.TypeOf(i)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Kind() == reflect.Struct
}

// FindEmbeddedSliceOrMap checks if variable has a slice/array/map or a pointer to it embedded.
func FindEmbeddedSliceOrMap(i interface{}) reflect.Type {
	if i == nil {
		return nil
	}
	t := DeepIndirect(reflect.TypeOf(i))
	if t.Kind() != reflect.Struct {
		return nil
	}

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.Anonymous {
			v := reflect.Zero(f.Type).Interface()
			if IsSliceOrMap(v) {
				return f.Type
			}
			if t := FindEmbeddedSliceOrMap(v); t != nil {
				return t
			}
		}
	}
	return nil
}
