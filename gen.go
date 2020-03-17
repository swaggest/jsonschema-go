package jsonschema

import (
	"encoding"
	"encoding/json"
	"fmt"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/swaggest/jsonschema-go/refl"
)

var (
	typeOfJSONRawMsg      = reflect.TypeOf(json.RawMessage{})
	typeOfTime            = reflect.TypeOf(time.Time{})
	typeOfTextUnmarshaler = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
	typeOfEmptyInterface  = reflect.TypeOf((*interface{})(nil)).Elem()
)

type Described interface {
	Describe() string
}

type Titled interface {
	Title() string
}

type Ref struct {
	Path string
	Name string
}

func (r Ref) Schema() Schema {
	s := r.Path + r.Name

	return Schema{
		Ref: &s,
	}
}

type Generator struct {
	DefaultOptions []func(*ParseContext)
	typesMap       map[refl.TypeString]interface{}
}

func (g *Generator) AddTypeMapping(src, dst interface{}) {
	if g.typesMap == nil {
		g.typesMap = map[refl.TypeString]interface{}{}
	}

	g.typesMap[refl.GoType(refl.DeepIndirect(reflect.TypeOf(src)))] = dst
}

func checkSchemaSetup(v reflect.Value, s *Schema) (bool, error) {
	if preparer, ok := v.Interface().(Preparer); ok {
		err := preparer.PrepareJSONSchema(s)
		return false, err
	}

	return false, nil
}

func (g *Generator) Parse(i interface{}, options ...func(*ParseContext)) (Schema, error) {
	pc := ParseContext{}
	pc.DefinitionsPrefix = "#/definitions/"
	pc.PropertyNameTag = "json"
	pc.Path = []string{"#"}
	pc.typeCycles = make(map[refl.TypeString]bool)

	HijackType(checkSchemaSetup)(&pc)

	for _, option := range g.DefaultOptions {
		option(&pc)
	}

	for _, option := range options {
		option(&pc)
	}

	schema, err := g.parse(i, &pc)
	if err == nil && len(pc.definitions) > 0 {
		schema.Definitions = make(map[string]SchemaOrBool, len(pc.definitions))

		for typeString, def := range pc.definitions {
			def := def
			ref := pc.definitionRefs[typeString]
			schema.Definitions[ref.Name] = def.ToSchemaOrBool()
		}
	}

	return schema, err
}

func (g *Generator) parse(i interface{}, pc *ParseContext) (schema Schema, err error) {
	var (
		typeString refl.TypeString
		defName    string
		t          = reflect.TypeOf(i)
		v          = reflect.ValueOf(i)
	)

	defer func() {
		pc.Path = pc.Path[:len(pc.Path)-1]

		if t == nil {
			return
		}

		if err != nil {
			return
		}

		if schema.Ref != nil {
			return
		}

		if pc.InlineRefs {
			return
		}

		if pc.InlineRoot && len(pc.Path) == 0 {
			return
		}

		if defName == "" {
			return
		}

		if pc.definitions == nil {
			pc.definitions = make(map[refl.TypeString]Schema, 1)
			pc.definitionRefs = make(map[refl.TypeString]Ref, 1)
		}

		pc.definitions[typeString] = schema
		ref := Ref{Path: pc.DefinitionsPrefix, Name: defName}
		pc.definitionRefs[typeString] = ref

		schema = ref.Schema()
	}()

	if t == nil || t == typeOfEmptyInterface {
		return schema, nil
	}

	if t.Kind() == reflect.Ptr {
		schema.AddType(Null)
	}

	t = refl.DeepIndirect(t)
	typeString = refl.GoType(t)
	pkgPath := t.PkgPath()

	if pkgPath != "" && pkgPath != "time" && pkgPath != "encoding/json" {
		defName = toCamel(path.Base(t.PkgPath())) + strings.Title(t.Name())
	}

	if t == nil || t == typeOfEmptyInterface {
		return schema, nil
	}

	if mappedTo, found := g.typesMap[typeString]; found {
		t = refl.DeepIndirect(reflect.TypeOf(mappedTo))
		v = reflect.ValueOf(mappedTo)
	}

	// Shortcut on embedded map or slice.
	if et := refl.FindEmbeddedSliceOrMap(i); et != nil {
		t = et
	}

	if t == typeOfTime {
		schema.AddType(String)
		schema.WithFormat("date-time")

		return
	}

	if t.Implements(typeOfTextUnmarshaler) {
		schema.AddType(String)
		return
	}

	if pc.HijackType != nil {
		var ret bool

		ret, err = pc.HijackType(v, &schema)
		if err != nil || ret {
			return schema, err
		}
	}

	if ref, ok := pc.definitionRefs[typeString]; ok {
		return ref.Schema(), nil
	}

	if pc.typeCycles[typeString] {
		return
	}

	if t.PkgPath() != "" {
		pc.typeCycles[typeString] = true
	}

	if vd, ok := v.Interface().(Described); ok {
		schema.WithDescription(vd.Describe())
	}

	if vt, ok := v.Interface().(Titled); ok {
		schema.WithTitle(vt.Title())
	}

	err = g.kindSwitch(t, v, &schema, pc)

	return schema, err
}

func (g *Generator) kindSwitch(t reflect.Type, v reflect.Value, schema *Schema, pc *ParseContext) error {
	switch t.Kind() {
	case reflect.Struct:
		switch {
		case reflect.PtrTo(t).Implements(typeOfTextUnmarshaler):
			schema.AddType(String)
		default:
			schema.AddType(Object)

			err := g.walkProperties(v, schema, pc)
			if err != nil {
				return err
			}
		}

	case reflect.Slice, reflect.Array:
		if t == typeOfJSONRawMsg {
			break
		}

		elemType := refl.DeepIndirect(t.Elem())

		pc.Path = append(pc.Path, "[]")
		itemValue := reflect.Zero(elemType).Interface()

		if itemValue == nil && elemType != typeOfEmptyInterface {
			itemValue = reflect.New(elemType).Interface()
		}

		itemsSchema, err := g.parse(itemValue, pc)
		if err != nil {
			return err
		}

		schema.AddType(Array)
		schema.WithItems(*(&Items{}).WithSchemaOrBool(itemsSchema.ToSchemaOrBool()))

	case reflect.Map:
		elemType := refl.DeepIndirect(t.Elem())

		pc.Path = append(pc.Path, "{}")
		itemValue := reflect.Zero(elemType).Interface()

		if itemValue == nil && elemType != typeOfEmptyInterface {
			itemValue = reflect.New(elemType).Interface()
		}

		additionalPropertiesSchema, err := g.parse(itemValue, pc)
		if err != nil {
			return err
		}

		schema.AddType(Object)
		schema.WithAdditionalProperties(additionalPropertiesSchema.ToSchemaOrBool())

	case reflect.Bool:
		schema.AddType(Boolean)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema.AddType(Integer)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema.AddType(Integer)
		schema.WithMinimum(0)
	case reflect.Float32, reflect.Float64:
		schema.AddType(Number)
	case reflect.String:
		schema.AddType(String)
	case reflect.Interface:
		return fmt.Errorf("non-empty interface is not supported: %s", t.String())
	default:
		return fmt.Errorf("type is not supported: %s", t.String())
	}

	return nil
}

func (g *Generator) walkProperties(v reflect.Value, parent *Schema, pc *ParseContext) error {
	t := v.Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()

		if v.IsZero() {
			v = reflect.Zero(t)
		} else {
			v = v.Elem()
		}
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		var tag = field.Tag.Get(pc.PropertyNameTag)

		// Skip explicitly discarded field.
		if tag == "-" {
			continue
		}

		if tag == "" && field.Anonymous {
			err := g.walkProperties(v.Field(i), parent, pc)
			if err != nil {
				return err
			}

			continue
		}

		// Skip the field if tag is not set.
		if tag == "" {
			continue
		}

		propName := strings.Split(tag, ",")[0]

		pc.WalkedProperties = append(pc.WalkedProperties, propName)

		required := false

		err := refl.ReadBoolTag(field.Tag, "required", &required)
		if err != nil {
			return err
		}

		if required {
			parent.Required = append(parent.Required, propName)
		}

		fieldVal := v.Field(i).Interface()

		ft := t.Field(i).Type

		if fieldVal == nil && ft != typeOfEmptyInterface {
			fieldVal = reflect.Zero(ft).Interface()
			if fieldVal == nil {
				fieldVal = reflect.New(ft).Interface()
			}
		}

		pc.Path = append(pc.Path, propName)
		propertySchema, err := g.parse(fieldVal, pc)

		if err != nil {
			return err
		}

		err = refl.PopulateFieldsFromTags(&propertySchema, field.Tag)

		if err != nil {
			return err
		}

		enum := enum{}
		enum.loadFromField(field, fieldVal)

		if len(enum.items) > 0 {
			propertySchema.Enum = enum.items
			if len(enum.names) > 0 {
				if propertySchema.ExtraProperties == nil {
					propertySchema.ExtraProperties = make(map[string]interface{}, 1)
				}

				propertySchema.ExtraProperties[XEnumNames] = enum.names
			}
		}

		if parent.Properties == nil {
			parent.Properties = make(map[string]SchemaOrBool, 1)
		}

		parent.Properties[propName] = SchemaOrBool{
			TypeObject: &propertySchema,
		}
	}

	return nil
}

// enum can be use for sending enum data that need validate.
type enum struct {
	items []interface{}
	names []string
}

// loadFromField loads enum from field tag: json array or comma-separated string.
func (enum *enum) loadFromField(field reflect.StructField, fieldVal interface{}) {
	if e, isEnumer := fieldVal.(NamedEnum); isEnumer {
		enum.items, enum.names = e.NamedEnum()
	}

	if e, isEnumer := fieldVal.(Enum); isEnumer {
		enum.items = e.Enum()
	}

	if enumTag := field.Tag.Get("enum"); enumTag != "" {
		var e []interface{}
		err := json.Unmarshal([]byte(enumTag), &e)

		if err != nil {
			es := strings.Split(enumTag, ",")
			e = make([]interface{}, len(es))

			for i, s := range es {
				e[i] = s
			}
		}

		enum.items = e
	}
}
