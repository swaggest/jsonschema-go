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

func (r Ref) Schema() CoreSchemaMetaSchema {
	s := r.Path + r.Name
	return CoreSchemaMetaSchema{
		Ref: &s,
	}
}

type Generator struct {
	typesMap       map[refl.TypeString]interface{}
	reflectGoTypes bool
}

func (g *Generator) AddTypeMapping(src, dst interface{}) {
	if g.typesMap == nil {
		g.typesMap = map[refl.TypeString]interface{}{}
	}
	g.typesMap[refl.GoType(refl.DeepIndirect(reflect.TypeOf(src)))] = dst
}

func (g *Generator) getMappedType(t reflect.Type) (dst interface{}, found bool) {
	goTypeName := refl.GoType(refl.DeepIndirect(t))
	dst, found = g.typesMap[goTypeName]
	return
}

func (g *Generator) Parse(i interface{}, options ...func(*ParseContext)) (CoreSchemaMetaSchema, error) {
	pc := ParseContext{}
	pc.DefinitionsPrefix = "#/definitions/"
	pc.PropertyNameTag = "json"
	pc.Path = []string{"#"}
	pc.typeCycles = make(map[refl.TypeString]bool)

	for _, option := range options {
		option(&pc)
	}

	schema, err := g.parse(i, &pc)
	if err == nil && len(pc.definitions) > 0 {
		schema.Definitions = make(map[string]Schema, len(pc.definitions))
		for typeString, def := range pc.definitions {
			def := def
			ref := pc.definitionRefs[typeString]
			schema.Definitions[ref.Name] = def.ToSchema()
		}
	}
	return schema, err
}

func (g *Generator) parse(i interface{}, pc *ParseContext) (schema CoreSchemaMetaSchema, err error) {
	var (
		typeString refl.TypeString
		t          = reflect.TypeOf(i)
		v          = reflect.ValueOf(i)
	)

	if t == nil {
		return CoreSchemaMetaSchema{}, nil
	}

	defer func() {
		pc.Path = pc.Path[:len(pc.Path)-1]

		if schema.Ref != nil {
			return
		}

		if err != nil {
			return
		}
		if customizer, ok := i.(Customizer); ok {
			err = customizer.CustomizeJSONSchema(&schema)
		}

		if pc.InlineRefs {
			return
		}

		pkgPath := t.PkgPath()
		if pkgPath == "" || pkgPath == "time" || pkgPath == "encoding/json" {
			return
		}

		if pc.definitions == nil {
			pc.definitions = make(map[refl.TypeString]CoreSchemaMetaSchema, 1)
			pc.definitionRefs = make(map[refl.TypeString]Ref, 1)
		}

		//defName := string(typeString)
		defName := toCamel(path.Base(pkgPath)) + strings.Title(t.Name())

		pc.definitions[typeString] = schema
		ref := Ref{Path: pc.DefinitionsPrefix, Name: defName}
		pc.definitionRefs[typeString] = ref

		schema = ref.Schema()

		//println(typeString, t.PkgPath())
	}()

	if mappedTo, ok := g.getMappedType(t); ok {
		t = reflect.TypeOf(mappedTo)
		v = reflect.ValueOf(mappedTo)
	}

	// Shortcut on embedded map or slice.
	if et := refl.FindEmbeddedSliceOrMap(i); et != nil {
		t = et
	}

	if t.Kind() == reflect.Ptr {
		schema.AddType(Null)
	}

	t = refl.DeepIndirect(t)
	typeString = refl.GoType(t)

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
		ret, err = pc.HijackType(t, &schema)
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

	switch t.Kind() {
	case reflect.Struct:
		switch true {
		case reflect.PtrTo(t).Implements(typeOfTextUnmarshaler):
			schema.AddType(String)
		default:
			schema.AddType(Object)
			err = g.walkProperties(v, &schema, pc)
			if err != nil {
				return schema, err
			}
		}

	case reflect.Slice, reflect.Array:
		if t == typeOfJSONRawMsg {
			break
		}

		elemType := refl.DeepIndirect(t.Elem())

		pc.Path = append(pc.Path, "[]")
		itemsSchema, err := g.parse(reflect.Zero(elemType).Interface(), pc)
		if err != nil {
			return schema, err
		}

		schema.AddType(Array)
		schema.WithItems(*(&Items{}).WithSchema(itemsSchema.ToSchema()))

	case reflect.Map:
		elemType := refl.DeepIndirect(t.Elem())

		pc.Path = append(pc.Path, "{}")
		additionalPropertiesSchema, err := g.parse(reflect.Zero(elemType).Interface(), pc)
		if err != nil {
			return schema, err
		}

		schema.AddType(Object)
		schema.WithAdditionalProperties(additionalPropertiesSchema.ToSchema())

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
		return schema, fmt.Errorf("non-empty interface is not supported: %s", typeString)
	default:
		return schema, fmt.Errorf("type is not supported: %s", typeString)
	}

	return schema, nil
}

func (g *Generator) walkProperties(v reflect.Value, parent *CoreSchemaMetaSchema, pc *ParseContext) error {
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

		required := false
		err := refl.ReadBoolTag(field.Tag, "required", &required)
		if err != nil {
			return err
		}
		if required {
			parent.Required = append(parent.Required, propName)
		}

		fieldVal := v.Field(i).Interface()

		if fieldVal == nil {
			fieldVal = reflect.New(t.Field(i).Type).Interface()
		}

		pc.Path = append(pc.Path, propName)
		propertySchema, err := g.parse(fieldVal, pc)
		if err != nil {
			return err
		}

		// Read tags.
		// TODO get rid of these handcrafted readers in favor of reflection walker.
		err = refl.JoinErrors(
			refl.ReadStringPtrTag(field.Tag, "title", &propertySchema.Title),
			refl.ReadStringPtrTag(field.Tag, "description", &propertySchema.Description),
			refl.ReadStringPtrTag(field.Tag, "format", &propertySchema.Format),
			refl.ReadStringPtrTag(field.Tag, "pattern", &propertySchema.Pattern),
			refl.ReadStringPtrTag(field.Tag, "contentMediaType", &propertySchema.ContentMediaType),
			refl.ReadStringPtrTag(field.Tag, "contentEncoding", &propertySchema.ContentEncoding),

			refl.ReadIntPtrTag(field.Tag, "maxLength", &propertySchema.MaxLength),
			refl.ReadIntTag(field.Tag, "minLength", &propertySchema.MinLength),
			refl.ReadIntPtrTag(field.Tag, "maxItems", &propertySchema.MaxItems),
			refl.ReadIntTag(field.Tag, "minItems", &propertySchema.MinItems),
			refl.ReadIntPtrTag(field.Tag, "maxProperties", &propertySchema.MaxProperties),
			refl.ReadIntTag(field.Tag, "minProperties", &propertySchema.MinProperties),

			refl.ReadFloatPtrTag(field.Tag, "multipleOf", &propertySchema.MultipleOf),
			refl.ReadFloatPtrTag(field.Tag, "maximum", &propertySchema.Maximum),
			refl.ReadFloatPtrTag(field.Tag, "minimum", &propertySchema.Minimum),

			refl.ReadFloatPtrTag(field.Tag, "exclusiveMaximum", &propertySchema.ExclusiveMaximum),
			refl.ReadFloatPtrTag(field.Tag, "exclusiveMinimum", &propertySchema.ExclusiveMinimum),
			refl.ReadBoolPtrTag(field.Tag, "uniqueItems", &propertySchema.UniqueItems),
			refl.ReadBoolPtrTag(field.Tag, "readOnly", &propertySchema.ReadOnly),
		)
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
			parent.Properties = make(map[string]Schema, 1)
		}
		parent.Properties[propName] = Schema{
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
