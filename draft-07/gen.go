package jsonschema

import (
	"encoding/json"
	"fmt"
	"path"
	"reflect"
	"strings"

	"github.com/swaggest/jsonschema-go/refl"
)

type Ref string

func (r Ref) Schema() CoreSchemaMetaSchema {
	s := string(r)
	return CoreSchemaMetaSchema{
		Ref: &s,
	}
}

type Generator struct {
	typesMap        map[refl.TypeString]interface{}
	definitions     map[refl.TypeString]CoreSchemaMetaSchema // list of all definition objects
	definitionRefs  map[refl.TypeString]Ref
	definitionAlloc map[string]refl.TypeString // index of allocated TypeNames
	propertyNameTag string
	reflectGoTypes  bool
}

func (g *Generator) getMappedType(t reflect.Type) (dst interface{}, found bool) {
	goTypeName := refl.GoType(refl.DeepIndirect(t))
	dst, found = g.typesMap[goTypeName]
	return
}

// reflectTypeReliableName returns real name of given reflect.Type
func (g *Generator) reflectTypeReliableName(t reflect.Type) string {
	if t.Name() != "" {
		// todo consider optionally processing package
		// return path.Base(t.PkgPath()) + t.Name()
		return t.Name()
	}
	return fmt.Sprintf("anon_%08x", reflect.Indirect(reflect.ValueOf(t)).FieldByName("hash").Uint())
}

func (g *Generator) getDefinition(t reflect.Type) (typeDef CoreSchemaMetaSchema, found bool) {
	typeDef, found = g.definitions[refl.GoType(t)]
	if !found && t.Kind() == reflect.Ptr {
		typeDef, found = g.definitions[refl.GoType(t.Elem())]
	}
	return
}

func (g *Generator) makeNameForType(t reflect.Type, baseTypeName string) string {
	goTypeName := refl.GoType(t)
	baseTypeName = strings.Title(baseTypeName)

	if g.definitionAlloc == nil {
		g.definitionAlloc = make(map[string]refl.TypeString, 1)
	}

	for typeName, allocatedGoTypeName := range g.definitionAlloc {
		if goTypeName == allocatedGoTypeName {
			return typeName
		}
	}

	pkgPath := t.PkgPath()

	if pkgPath != "" {
		pref := strings.Title(path.Base(pkgPath))
		baseTypeName = pref + baseTypeName
		pkgPath = path.Dir(pkgPath)
	}

	allocatedType, isAllocated := g.definitionAlloc[baseTypeName]
	if isAllocated && allocatedType != goTypeName {
		typeIndex := 2
		pref := strings.Title(path.Base(pkgPath))
		for {
			typeName := ""
			if pkgPath != "" {
				typeName = pref + baseTypeName
			} else {
				typeName = fmt.Sprintf("%sType%d", baseTypeName, typeIndex)
				typeIndex++
			}
			allocatedType, isAllocated := g.definitionAlloc[typeName]

			if !isAllocated || allocatedType == goTypeName {
				baseTypeName = typeName
				break
			}
			typeIndex++
			pref = strings.Title(path.Base(pkgPath)) + pref
			pkgPath = path.Dir(pkgPath)
		}
	}
	g.definitionAlloc[baseTypeName] = goTypeName
	return baseTypeName
}

func (g *Generator) Parse(i interface{}) (schema CoreSchemaMetaSchema, err error) {
	defer func() {
		if err != nil {
			return
		}
		if customizer, ok := i.(Customizer); ok {
			err = customizer.CustomizeJSONSchema(&schema)
		}
	}()

	var (
		t = reflect.TypeOf(i)
		v = reflect.ValueOf(i)
	)

	if mappedTo, ok := g.getMappedType(t); ok {
		t = reflect.TypeOf(mappedTo)
		v = reflect.ValueOf(mappedTo)
	}

	// Shortcut on embedded map or slice.
	if et := refl.FindEmbeddedSliceOrMap(i); et != nil {
		t = et
	}

	t = refl.DeepIndirect(t)

	typeString := refl.GoType(t)

	if ref, ok := g.definitionRefs[typeString]; ok {
		return ref.Schema(), nil
	}

	switch t.Kind() {
	case reflect.Struct:
		schema.WithType(Object.Type())
		err = g.walkProperties(v, &schema)
		if err != nil {
			return schema, err
		}

	case reflect.Slice, reflect.Array:
		elemType := refl.DeepIndirect(t.Elem())

		itemsSchema, err := g.Parse(reflect.Zero(elemType).Interface())
		if err != nil {
			return schema, err
		}

		schema.WithType(Array.Type())
		schema.WithItems(*(&Items{}).WithSchema(itemsSchema.ToSchema()))

	case reflect.Map:
		elemType := refl.DeepIndirect(t.Elem())

		additionalPropertiesSchema, err := g.Parse(reflect.Zero(elemType))
		if err != nil {
			return schema, err
		}

		schema.WithType(Object.Type())
		schema.WithAdditionalProperties(additionalPropertiesSchema.ToSchema())

	case reflect.Bool:
		schema.WithType(Boolean.Type())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema.WithType(Integer.Type())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema.WithType(Integer.Type())
		schema.WithMinimum(0)
	case reflect.Float32, reflect.Float64:
		schema.WithType(Number.Type())
	case reflect.String:
		schema.WithType(String.Type())
	case reflect.Interface:
		return schema, fmt.Errorf("non-empty interface is not supported: %s", typeString)
	default:
		return schema, fmt.Errorf("type is not supported: %s", typeString)
	}

	return schema, nil
}

func (g *Generator) walkProperties(v reflect.Value, parent *CoreSchemaMetaSchema) error {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	propertyNameTag := g.propertyNameTag
	if propertyNameTag == "" {
		propertyNameTag = "json"
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		var tag = field.Tag.Get(propertyNameTag)

		// Skip explicitly discarded field.
		if tag == "-" {
			continue
		}

		if tag == "" && field.Anonymous {
			err := g.walkProperties(v.Field(i), parent)
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

		propertySchema, err := g.Parse(fieldVal)
		if err != nil {
			return err
		}

		// Read tags.
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
