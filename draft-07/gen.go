package jsonschema

import (
	"encoding/json"
	"fmt"
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

func (g *Generator) Parse(i interface{}) (CoreSchemaMetaSchema, error) {
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
	schema := CoreSchemaMetaSchema{}

	floatZero := 0.0

	switch t.Kind() {
	case reflect.Struct:
		schema.Type = &Type{
			SimpleTypes: SimpleTypesObject.Ptr(),
		}
		err := g.walkProperties(v, &schema)
		if err != nil {
			return schema, err
		}

	case reflect.Slice, reflect.Array:
		elemType := refl.DeepIndirect(t.Elem())

		itemsSchema, err := g.Parse(reflect.Zero(elemType))
		if err != nil {
			return schema, err
		}

		schema.Type = &Type{
			SimpleTypes: SimpleTypesArray.Ptr(),
		}
		schema.Items = &Items{
			Schema: &Schema{
				TypeObject: &itemsSchema,
			},
		}

	case reflect.Map:
		elemType := refl.DeepIndirect(t.Elem())

		additionalPropertiesSchema, err := g.Parse(reflect.Zero(elemType))
		if err != nil {
			return schema, err
		}

		schema.Type = &Type{
			SimpleTypes: SimpleTypesObject.Ptr(),
		}
		schema.AdditionalProperties = &Schema{
			TypeObject: &additionalPropertiesSchema,
		}

	case reflect.Bool:
		schema.Type = &Type{
			SimpleTypes: SimpleTypesBoolean.Ptr(),
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		schema.Type = &Type{
			SimpleTypes: SimpleTypesInteger.Ptr(),
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		schema.Type = &Type{
			SimpleTypes: SimpleTypesInteger.Ptr(),
		}
		schema.Minimum = &floatZero
	case reflect.Float32, reflect.Float64:
		schema.Type = &Type{
			SimpleTypes: SimpleTypesNumber.Ptr(),
		}
	case reflect.String:
		schema.Type = &Type{
			SimpleTypes: SimpleTypesString.Ptr(),
		}
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
			// Where are the extra props?!
			//if len(enum.names) > 0 {
			//	propertySchema.ExtraProperties = ""
			//}
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
	type namedEnum interface {
		// NamedEnum return the const-name pair slice
		NamedEnum() ([]interface{}, []string)
	}

	type plainEnum interface {
		Enum() []interface{}
	}

	if e, isEnumer := fieldVal.(namedEnum); isEnumer {
		enum.items, enum.names = e.NamedEnum()
	}

	if e, isEnumer := fieldVal.(plainEnum); isEnumer {
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
