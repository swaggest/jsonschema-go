package jsonschema

import (
	"encoding"
	"encoding/json"
	"fmt"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/swaggest/refl"
)

var (
	typeOfJSONRawMsg      = reflect.TypeOf(json.RawMessage{})
	typeOfTime            = reflect.TypeOf(time.Time{})
	typeOfTextUnmarshaler = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
	typeOfEmptyInterface  = reflect.TypeOf((*interface{})(nil)).Elem()
)

// IgnoreTypeName is a marker interface to ignore type name of mapped value and use original.
type IgnoreTypeName interface {
	IgnoreTypeName()
}

// IgnoreTypeName instructs reflector to keep original type name during mapping.
func (s Schema) IgnoreTypeName() {}

// Described exposes description.
type Described interface {
	Description() string
}

// Titled exposes title.
type Titled interface {
	Title() string
}

// Ref is a definition reference.
type Ref struct {
	Path string
	Name string
}

// Schema creates schema instance from reference.
func (r Ref) Schema() Schema {
	s := r.Path + r.Name

	return Schema{
		Ref: &s,
	}
}

// Reflector creates JSON Schemas from Go values.
type Reflector struct {
	DefaultOptions []func(*ReflectContext)
	typesMap       map[refl.TypeString]interface{}
}

// AddTypeMapping creates substitution link between types of src and dst when reflecting JSON Schema.
func (r *Reflector) AddTypeMapping(src, dst interface{}) {
	if r.typesMap == nil {
		r.typesMap = map[refl.TypeString]interface{}{}
	}

	r.typesMap[refl.GoType(refl.DeepIndirect(reflect.TypeOf(src)))] = dst
}

func checkSchemaSetup(v reflect.Value, s *Schema) (bool, error) {
	if preparer, ok := v.Interface().(Preparer); ok {
		err := preparer.PrepareJSONSchema(s)
		return false, err
	}

	if exposer, ok := v.Interface().(Exposer); ok {
		schema, err := exposer.JSONSchema()
		if err != nil {
			return true, err
		}

		*s = schema

		return true, nil
	}

	if exposer, ok := v.Interface().(RawExposer); ok {
		schemaBytes, err := exposer.JSONSchemaBytes()
		if err != nil {
			return true, err
		}

		err = json.Unmarshal(schemaBytes, s)
		if err != nil {
			return true, err
		}

		return true, nil
	}

	return false, nil
}

// Reflect walks Go value and builds its JSON Schema based on types and field tags.
func (r *Reflector) Reflect(i interface{}, options ...func(*ReflectContext)) (Schema, error) {
	rc := ReflectContext{}
	rc.DefinitionsPrefix = "#/definitions/"
	rc.PropertyNameTag = "json"
	rc.Path = []string{"#"}
	rc.typeCycles = make(map[refl.TypeString]bool)

	InterceptType(checkSchemaSetup)(&rc)

	for _, option := range r.DefaultOptions {
		option(&rc)
	}

	for _, option := range options {
		option(&rc)
	}

	schema, err := r.reflect(i, &rc)
	if err == nil && len(rc.definitions) > 0 {
		schema.Definitions = make(map[string]SchemaOrBool, len(rc.definitions))

		for typeString, def := range rc.definitions {
			def := def
			ref := rc.definitionRefs[typeString]

			if rc.CollectDefinitions != nil {
				rc.CollectDefinitions(ref.Name, def)
			} else {
				schema.Definitions[ref.Name] = def.ToSchemaOrBool()
			}
		}
	}

	return schema, err
}

func removeNull(t *Type) {
	if t.SimpleTypes != nil && *t.SimpleTypes == Null {
		t.SimpleTypes = nil
	} else if len(t.SliceOfSimpleTypeValues) > 0 {
		for i, ti := range t.SliceOfSimpleTypeValues {
			if ti == Null {
				// Remove Null from slice.
				t.SliceOfSimpleTypeValues = append(t.SliceOfSimpleTypeValues[:i],
					t.SliceOfSimpleTypeValues[i+1:]...)
			}
		}

		if len(t.SliceOfSimpleTypeValues) == 1 {
			t.SimpleTypes = &t.SliceOfSimpleTypeValues[0]
			t.SliceOfSimpleTypeValues = nil
		}
	}
}

func (r *Reflector) reflectDefer(defName string, typeString refl.TypeString, rc *ReflectContext, schema Schema) Schema {
	if rc.RootNullable && len(rc.Path) == 0 {
		schema.AddType(Null)
	}

	if schema.Ref != nil {
		return schema
	}

	if rc.InlineRefs {
		return schema
	}

	if !rc.RootRef && len(rc.Path) == 0 {
		return schema
	}

	if defName == "" {
		return schema
	}

	if !rc.RootRef && defName == rc.rootDefName {
		ref := Ref{Path: "#"}
		return ref.Schema()
	}

	if rc.definitions == nil {
		rc.definitions = make(map[refl.TypeString]Schema, 1)
		rc.definitionRefs = make(map[refl.TypeString]Ref, 1)
	}

	rc.definitions[typeString] = schema
	ref := Ref{Path: rc.DefinitionsPrefix, Name: defName}
	rc.definitionRefs[typeString] = ref

	return ref.Schema()
}

func (r *Reflector) reflect(i interface{}, rc *ReflectContext) (schema Schema, err error) {
	var (
		typeString refl.TypeString
		defName    string
		t          = reflect.TypeOf(i)
		v          = reflect.ValueOf(i)
	)

	defer func() {
		rc.Path = rc.Path[:len(rc.Path)-1]

		if t == nil {
			return
		}

		if err != nil {
			return
		}

		schema = r.reflectDefer(defName, typeString, rc, schema)
	}()

	if t == nil || t == typeOfEmptyInterface {
		return schema, nil
	}

	if t.Kind() == reflect.Ptr && t.Elem() != typeOfJSONRawMsg {
		schema.AddType(Null)
	}

	t = refl.DeepIndirect(t)

	if t == nil || t == typeOfEmptyInterface {
		schema.Type = nil

		return schema, nil
	}

	typeString = refl.GoType(t)
	pkgPath := t.PkgPath()

	if pkgPath != "" && pkgPath != "time" && pkgPath != "encoding/json" {
		defName = toCamel(path.Base(t.PkgPath())) + strings.Title(t.Name())
	}

	rebuildDefName := false
	if mappedTo, found := r.typesMap[refl.GoType(t)]; found {
		rebuildDefName = true
		t = refl.DeepIndirect(reflect.TypeOf(mappedTo))
		v = reflect.ValueOf(mappedTo)

		if _, ok := mappedTo.(IgnoreTypeName); ok {
			rebuildDefName = false
		}
	}

	if rebuildDefName {
		typeString = refl.GoType(t)
		pkgPath = t.PkgPath()
		defName = ""

		if pkgPath != "" && pkgPath != "time" && pkgPath != "encoding/json" {
			defName = toCamel(path.Base(t.PkgPath())) + strings.Title(t.Name())
		}
	}

	if len(rc.Path) == 1 {
		rc.rootDefName = defName
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

	if rc.InterceptType != nil {
		var ret bool

		ret, err = rc.InterceptType(v, &schema)
		if err != nil || ret {
			return schema, err
		}
	}

	if ref, ok := rc.definitionRefs[typeString]; ok {
		return ref.Schema(), nil
	}

	if rc.typeCycles[typeString] {
		return
	}

	if t.PkgPath() != "" && len(rc.Path) > 1 {
		rc.typeCycles[typeString] = true
	}

	if vd, ok := v.Interface().(Described); ok {
		schema.WithDescription(vd.Description())
	}

	if vt, ok := v.Interface().(Titled); ok {
		schema.WithTitle(vt.Title())
	}

	err = r.kindSwitch(t, v, &schema, rc)

	return schema, err
}

func (r *Reflector) kindSwitch(t reflect.Type, v reflect.Value, schema *Schema, rc *ReflectContext) error {
	switch t.Kind() {
	case reflect.Struct:
		switch {
		case reflect.PtrTo(t).Implements(typeOfTextUnmarshaler):
			schema.AddType(String)
		default:
			schema.AddType(Object)
			removeNull(schema.Type)

			err := r.walkProperties(v, schema, rc)
			if err != nil {
				return err
			}
		}

	case reflect.Slice, reflect.Array:
		if t == typeOfJSONRawMsg {
			break
		}

		elemType := refl.DeepIndirect(t.Elem())

		rc.Path = append(rc.Path, "[]")
		itemValue := reflect.Zero(elemType).Interface()

		if itemValue == nil && elemType != typeOfEmptyInterface {
			itemValue = reflect.New(elemType).Interface()
		}

		itemsSchema, err := r.reflect(itemValue, rc)
		if err != nil {
			return err
		}

		schema.AddType(Array)
		schema.WithItems(*(&Items{}).WithSchemaOrBool(itemsSchema.ToSchemaOrBool()))

	case reflect.Map:
		elemType := refl.DeepIndirect(t.Elem())

		rc.Path = append(rc.Path, "{}")
		itemValue := reflect.Zero(elemType).Interface()

		if itemValue == nil && elemType != typeOfEmptyInterface {
			itemValue = reflect.New(elemType).Interface()
		}

		additionalPropertiesSchema, err := r.reflect(itemValue, rc)
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
		if t.NumMethod() > 0 {
			return fmt.Errorf("non-empty interface is not supported: %s", t.String())
		}

		schema.Type = nil
	default:
		return fmt.Errorf("type is not supported: %s", t.String())
	}

	return nil
}

func (r *Reflector) walkProperties(v reflect.Value, parent *Schema, rc *ReflectContext) error {
	t := v.Type()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()

		if refl.IsZero(v) {
			v = reflect.Zero(t)
		} else {
			v = v.Elem()
		}
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		tag := field.Tag.Get(rc.PropertyNameTag)

		// Skip explicitly discarded field.
		if tag == "-" {
			continue
		}

		if tag == "" && field.Anonymous {
			err := r.walkProperties(v.Field(i), parent, rc)
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
		omitEmpty := strings.Contains(tag, ",omitempty")
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

		rc.Path = append(rc.Path, propName)

		propertySchema, err := r.reflect(fieldVal, rc)
		if err != nil {
			return err
		}

		if !omitEmpty {
			checkNullability(&propertySchema, rc, ft)
		}

		err = refl.PopulateFieldsFromTags(&propertySchema, field.Tag)

		if err != nil {
			return err
		}

		err = reflectExample(&propertySchema, field)
		if err != nil {
			return err
		}

		reflectEnum(&propertySchema, field, fieldVal)

		if parent.Properties == nil {
			parent.Properties = make(map[string]SchemaOrBool, 1)
		}

		parent.Properties[propName] = SchemaOrBool{
			TypeObject: &propertySchema,
		}

		if rc.InterceptProperty != nil {
			err = rc.InterceptProperty(propName, field, &propertySchema)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func checkNullability(propertySchema *Schema, rc *ReflectContext, ft reflect.Type) {
	if propertySchema.HasType(Array) || (propertySchema.HasType(Object) && len(propertySchema.Properties) == 0) {
		propertySchema.AddType(Null)
	}

	if propertySchema.Ref != nil && ft.Kind() != reflect.Struct {
		def := rc.getDefinition(*propertySchema.Ref)

		if (def.HasType(Array) || def.HasType(Object)) && !def.HasType(Null) {
			if rc.EnvelopNullability {
				refSchema := *propertySchema
				propertySchema.Ref = nil
				propertySchema.AnyOf = []SchemaOrBool{
					Null.ToSchemaOrBool(),
					refSchema.ToSchemaOrBool(),
				}
			} else {
				def.AddType(Null)
			}
		}
	}
}

func reflectExample(propertySchema *Schema, field reflect.StructField) error {
	var err error

	if propertySchema.Type == nil || propertySchema.Type.SimpleTypes == nil {
		return nil
	}

	t := *propertySchema.Type.SimpleTypes
	switch t {
	case String:
		var example *string

		refl.ReadStringPtrTag(field.Tag, "example", &example)

		if example != nil {
			propertySchema.WithExamples(*example)
		}
	case Integer:
		var example *int64

		err = refl.ReadIntPtrTag(field.Tag, "example", &example)
		if err != nil {
			return err
		}

		if example != nil {
			propertySchema.WithExamples(*example)
		}
	case Number:
		var example *float64

		err = refl.ReadFloatPtrTag(field.Tag, "example", &example)
		if err != nil {
			return err
		}

		if example != nil {
			propertySchema.WithExamples(*example)
		}
	case Boolean:
		var example *bool

		err = refl.ReadBoolPtrTag(field.Tag, "example", &example)
		if err != nil {
			return err
		}

		if example != nil {
			propertySchema.WithExamples(*example)
		}
	}

	return nil
}

func reflectEnum(propertySchema *Schema, field reflect.StructField, fieldVal interface{}) {
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
