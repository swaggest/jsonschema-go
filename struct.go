package jsonschema

import (
	"reflect"
	"strconv"

	"github.com/swaggest/refl"
)

var structDefaultDefNameIndex = 0

// Field mimics Go reflect.StructField for purposes of schema reflection.
type Field struct {
	Name  string
	Value interface{}
	Tag   reflect.StructTag
}

// Struct mimics Go struct to allow schema reflection on virtual struct type.
//
// This can be handy for dynamic values that can not be represented as static Go structures.
type Struct struct {
	Title       *string
	Description *string
	Nullable    bool
	DefName     string

	Fields []Field
}

// SetTitle sets title.
func (s *Struct) SetTitle(title string) {
	s.Title = &title
}

// SetDescription sets description.
func (s *Struct) SetDescription(description string) {
	s.Description = &description
}

type withStruct interface {
	structPtr() *Struct
}

func (s Struct) structPtr() *Struct {
	return &s
}

func (s Struct) names() (string, refl.TypeString) {
	defName := s.DefName

	if defName == "" {
		structDefaultDefNameIndex++

		defName = "struct" + strconv.Itoa(structDefaultDefNameIndex)
	}

	return defName, refl.TypeString("struct." + defName)
}
