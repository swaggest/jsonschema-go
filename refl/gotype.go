package refl

import (
	"reflect"
	"strings"
)

type TypeString string

// GoType returns string representation of type name including import path
func GoType(t reflect.Type) TypeString {
	s := t.Name()
	pkgPath := t.PkgPath()
	if pkgPath != "" {
		pos := strings.Index(pkgPath, "/vendor/")
		if pos != -1 {
			pkgPath = pkgPath[pos+8:]
		}
		s = pkgPath + "." + s
	}

	ts := t.String()
	typeRef := s

	pos := strings.LastIndex(typeRef, "/")
	if pos != -1 {
		typeRef = typeRef[pos+1:]
	}

	if typeRef != ts {
		s = s + "::" + t.String()
	}

	switch t.Kind() {
	case reflect.Slice:
		return "[]" + GoType(t.Elem())
	case reflect.Ptr:
		return "*" + GoType(t.Elem())
	case reflect.Map:
		return "map[" + GoType(t.Key()) + "]" + GoType(t.Elem())
	}

	return TypeString(s)
}
