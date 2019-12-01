package refl

import (
	"errors"
	"reflect"
	"strconv"
)

// HasTaggedFields checks if the structure has fields with tag name
func HasTaggedFields(i interface{}, tagname string) bool {
	if i == nil {
		return false
	}
	t := reflect.TypeOf(i)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get(tagname)
		if tag != "" && tag != "-" {
			return true
		}
		if field.Anonymous {
			if tag != "-" && HasTaggedFields(reflect.New(field.Type).Interface(), tagname) {
				return true
			}
		}

	}
	return false
}

func ReadBoolTag(tag reflect.StructTag, name string, holder *bool) error {
	value, ok := tag.Lookup(name)
	if ok {
		v, err := strconv.ParseBool(value)
		if err != nil {
			return errors.New("failed to parse bool value " + value + " in tag " + name + ": " + err.Error())
		}
		*holder = v
	}
	return nil
}

func ReadBoolPtrTag(tag reflect.StructTag, name string, holder **bool) error {
	value, ok := tag.Lookup(name)
	if ok {
		v, err := strconv.ParseBool(value)
		if err != nil {
			return errors.New("failed to parse bool value " + value + " in tag " + name + ": " + err.Error())
		}
		*holder = &v
	}
	return nil
}

func ReadStringPtrTag(tag reflect.StructTag, name string, holder **string) error {
	value, ok := tag.Lookup(name)
	if ok {
		if *holder != nil && **holder != "" && value == "-" {
			*holder = nil
			return nil
		}
		*holder = &value
	}
	return nil
}

func ReadIntTag(tag reflect.StructTag, name string, holder *int64) error {
	value, ok := tag.Lookup(name)
	if ok {
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return errors.New("failed to parse float value " + value + " in tag " + name + ": " + err.Error())
		}
		*holder = v
	}
	return nil
}

func ReadIntPtrTag(tag reflect.StructTag, name string, holder **int64) error {
	value, ok := tag.Lookup(name)
	if ok {
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return errors.New("failed to parse int value " + value + " in tag " + name + ": " + err.Error())
		}
		*holder = &v
	}
	return nil
}

func ReadFloatTag(tag reflect.StructTag, name string, holder *float64) error {
	value, ok := tag.Lookup(name)
	if ok {
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return errors.New("failed to parse float value " + value + " in tag " + name + ": " + err.Error())
		}
		*holder = v
	}
	return nil
}

func ReadFloatPtrTag(tag reflect.StructTag, name string, holder **float64) error {
	value, ok := tag.Lookup(name)
	if ok {
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return errors.New("failed to parse float value " + value + " in tag " + name + ": " + err.Error())
		}
		*holder = &v
	}
	return nil
}

// JoinErrors joins non-nil errors.
func JoinErrors(errs ...error) error {
	join := ""
	for _, err := range errs {
		if err != nil {
			join += ", " + err.Error()
		}
	}
	if join != "" {
		return errors.New(join[2:])
	}
	return nil
}
