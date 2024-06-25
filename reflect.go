package jsonschema

import (
	"context"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/swaggest/refl"
)

var (
	typeOfJSONRawMsg      = reflect.TypeOf(json.RawMessage{})
	typeOfByteSlice       = reflect.TypeOf([]byte{})
	typeOfTime            = reflect.TypeOf(time.Time{})
	typeOfDate            = reflect.TypeOf(Date{})
	typeOfTextUnmarshaler = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
	typeOfTextMarshaler   = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
	typeOfJSONMarshaler   = reflect.TypeOf((*json.Marshaler)(nil)).Elem()
	typeOfEmptyInterface  = reflect.TypeOf((*interface{})(nil)).Elem()
	typeOfSchemaInliner   = reflect.TypeOf((*SchemaInliner)(nil)).Elem()
	typeOfEmbedReferencer = reflect.TypeOf((*EmbedReferencer)(nil)).Elem()
)

const (
	// ErrSkipProperty indicates that property should not be added to object.
	ErrSkipProperty = sentinelError("property skipped")
)

type sentinelError string

func (e sentinelError) Error() string {
	return string(e)
}

// IgnoreTypeName is a marker interface to ignore type name of mapped value and use original.
type IgnoreTypeName interface {
	IgnoreTypeName()
}

// SchemaInliner is a marker interface to inline schema without creating a definition.
type SchemaInliner interface {
	InlineJSONSchema()
}

// EmbedReferencer is a marker interface to enable reference to embedded struct type.
type EmbedReferencer interface {
	ReferEmbedded()
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

var defNameEscaper = strings.NewReplacer(
	"~", "~0",
	"/", "~1",
	"%", "%25",
)

// Schema creates schema instance from reference.
func (r Ref) Schema() Schema {
	s := r.Path + defNameEscaper.Replace(r.Name)

	return Schema{
		Ref: &s,
	}
}

// Reflector creates JSON Schemas from Go values.
type Reflector struct {
	DefaultOptions   []func(*ReflectContext)
	typesMap         map[reflect.Type]interface{}
	inlineDefinition map[refl.TypeString]bool
	defNameTypes     map[string]reflect.Type
}

// AddTypeMapping creates substitution link between types of src and dst when reflecting JSON Schema.
//
// A configured Schema instance can also be used as dst.
func (r *Reflector) AddTypeMapping(src, dst interface{}) {
	if r.typesMap == nil {
		r.typesMap = map[reflect.Type]interface{}{}
	}

	r.typesMap[refl.DeepIndirect(reflect.TypeOf(src))] = dst
}

// InlineDefinition enables schema inlining for a type of given sample.
//
// Inlined schema is used instead of a reference to a shared definition.
func (r *Reflector) InlineDefinition(sample interface{}) {
	if r.inlineDefinition == nil {
		r.inlineDefinition = map[refl.TypeString]bool{}
	}

	r.inlineDefinition[refl.GoType(refl.DeepIndirect(reflect.TypeOf(sample)))] = true
}

// InterceptDefName allows modifying reflected definition names.
//
// Deprecated: add jsonschema.InterceptDefName to DefaultOptions.
func (r *Reflector) InterceptDefName(f func(t reflect.Type, defaultDefName string) string) {
	r.DefaultOptions = append(r.DefaultOptions, InterceptDefName(f))
}

func checkSchemaSetup(params InterceptSchemaParams) (bool, error) {
	v := params.Value
	s := params.Schema

	reflectEnum(s, "", v.Interface())

	var e Exposer

	if exposer, ok := safeInterface(v).(Exposer); ok {
		e = exposer
	} else if exposer, ok := ptrTo(v).(Exposer); ok {
		e = exposer
	}

	if e != nil {
		schema, err := e.JSONSchema()
		if err != nil {
			return true, err
		}

		*s = schema

		return true, nil
	}

	var re RawExposer

	// Checking if RawExposer is defined on a current value.
	if exposer, ok := safeInterface(v).(RawExposer); ok {
		re = exposer
	} else if exposer, ok := ptrTo(v).(RawExposer); ok { // Checking if RawExposer is defined on a pointer to current value.
		re = exposer
	}

	if re != nil {
		schemaBytes, err := re.JSONSchemaBytes()
		if err != nil {
			return true, err
		}

		var rs Schema

		err = json.Unmarshal(schemaBytes, &rs)
		if err != nil {
			return true, err
		}

		*s = rs

		return true, nil
	}

	return false, nil
}

// Reflect walks Go value and builds its JSON Schema based on types and field tags.
//
// Values can be populated from field tags of original field:
//
//	type MyObj struct {
//	   BoundedNumber int `query:"boundedNumber" minimum:"-100" maximum:"100"`
//	   SpecialString string `json:"specialString" pattern:"^[a-z]{4}$" minLength:"4" maxLength:"4"`
//	}
//
// Note: field tags are only applied to inline schemas, if you use named type then referenced schema
// will be created and tags will be ignored. This happens because referenced schema can be used in
// multiple fields with conflicting tags, therefore customization of referenced schema has to done on
// the type itself via RawExposer, Exposer or Preparer.
//
// These tags can be used:
//   - `title`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.6.1
//   - `description`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.6.1
//   - `default`, can be scalar or JSON value,
//     https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.6.2
//   - `const`, can be scalar or JSON value,
//     https://json-schema.org/draft/2020-12/json-schema-validation.html#rfc.section.6.1.3
//   - `pattern`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.2.3
//   - `format`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.7
//   - `multipleOf`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.1.1
//   - `maximum`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.1.2
//   - `minimum`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.1.3
//   - `maxLength`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.2.1
//   - `minLength`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.2.2
//   - `maxItems`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.3.2
//   - `minItems`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.3.3
//   - `maxProperties`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.4.1
//   - `minProperties`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.4.2
//   - `exclusiveMaximum`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.1.2
//   - `exclusiveMinimum`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.1.3
//   - `uniqueItems`, https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.3.4
//   - `enum`, tag value must be a JSON or comma-separated list of strings,
//     https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.5.1
//   - `required`, boolean, marks property as required
//   - `nullable`, boolean, overrides nullability of a property
//
// Unnamed fields can be used to configure parent schema:
//
//	type MyObj struct {
//	   BoundedNumber int `query:"boundedNumber" minimum:"-100" maximum:"100"`
//	   SpecialString string `json:"specialString" pattern:"^[a-z]{4}$" minLength:"4" maxLength:"4"`
//	   _             struct{} `additionalProperties:"false" description:"MyObj is my object."`
//	}
//
// In case of a structure with multiple name tags, you can enable filtering of unnamed fields with
// ReflectContext.UnnamedFieldWithTag option and add matching name tags to structure (e.g. query:"_").
//
//	type MyObj struct {
//	   BoundedNumber int `query:"boundedNumber" minimum:"-100" maximum:"100"`
//	   SpecialString string `json:"specialString" pattern:"^[a-z]{4}$" minLength:"4" maxLength:"4"`
//	   // These parent schema tags would only be applied to `query` schema reflection (not for `json`).
//	   _ struct{} `query:"_" additionalProperties:"false" description:"MyObj is my object."`
//	}
//
// Additionally there are structure can implement any of special interfaces for fine-grained Schema control:
// RawExposer, Exposer, Preparer.
//
// These interfaces allow exposing particular schema keywords:
// Titled, Described, Enum, NamedEnum.
//
// Available options:
//
//		CollectDefinitions
//		DefinitionsPrefix
//		PropertyNameTag
//		InterceptNullability
//		InterceptType
//		InterceptProperty
//	 	InterceptDefName
//		InlineRefs
//		RootNullable
//		RootRef
//		StripDefinitionNamePrefix
//		PropertyNameMapping
//		ProcessWithoutTags
//		SkipEmbeddedMapsSlices
//		SkipUnsupportedProperties
//
// Fields from embedded structures are processed as if they were defined in the root structure.
// Alternatively, if embedded structure has a field tag `refer:"true"` or implements EmbedReferencer,
// its reference will be added to `allOf` of the parent schema.
func (r *Reflector) Reflect(i interface{}, options ...func(rc *ReflectContext)) (Schema, error) {
	rc := ReflectContext{}
	rc.Context = context.Background()
	rc.DefinitionsPrefix = "#/definitions/"
	rc.PropertyNameTag = "json"
	rc.Path = []string{"#"}
	rc.typeCycles = make(map[refl.TypeString]*Schema)

	InterceptSchema(checkSchemaSetup)(&rc)

	for _, option := range r.DefaultOptions {
		option(&rc)
	}

	for _, option := range options {
		option(&rc)
	}

	rc.deprecatedFallback()

	schema, err := r.reflect(i, &rc, false, nil)
	if err == nil && len(rc.definitions) > 0 {
		schema.Definitions = make(map[string]SchemaOrBool, len(rc.definitions))

		for typeString, def := range rc.definitions {
			def := def
			ref := rc.definitionRefs[typeString]

			if rc.CollectDefinitions != nil {
				rc.CollectDefinitions(ref.Name, *def)
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

func (r *Reflector) reflectDefer(defName string, typeString refl.TypeString, rc *ReflectContext, schema Schema, keepType bool) Schema {
	if rc.RootNullable && len(rc.Path) == 0 {
		schema.AddType(Null)
	}

	if schema.Ref != nil {
		return schema
	}

	if rc.InlineRefs {
		return schema
	}

	if r.inlineDefinition[typeString] {
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

	// Inlining trivial scalar schemas.
	if schema.IsTrivial() && schema.Type != nil && !schema.HasType(Object) && !schema.HasType(Array) {
		return schema
	}

	if rc.definitions == nil {
		rc.definitions = make(map[refl.TypeString]*Schema, 1)
		rc.definitionRefs = make(map[refl.TypeString]Ref, 1)
	}

	rc.definitions[typeString] = &schema
	ref := Ref{Path: rc.DefinitionsPrefix, Name: defName}
	rc.definitionRefs[typeString] = ref

	s := ref.Schema()

	if keepType {
		s.Type = schema.Type
	}

	s.ReflectType = schema.ReflectType

	return s
}

func (r *Reflector) checkTitle(v reflect.Value, s *Struct, schema *Schema) {
	if vd, ok := safeInterface(v).(Described); ok {
		schema.WithDescription(vd.Description())
	} else if vd, ok := ptrTo(v).(Described); ok {
		schema.WithDescription(vd.Description())
	}

	if s != nil && s.Description != nil {
		schema.WithDescription(*s.Description)
	}

	if vt, ok := safeInterface(v).(Titled); ok {
		schema.WithTitle(vt.Title())
	} else if vt, ok := ptrTo(v).(Titled); ok {
		schema.WithTitle(vt.Title())
	}

	if s != nil && s.Title != nil {
		schema.WithTitle(*s.Title)
	}
}

func (r *Reflector) reflect(i interface{}, rc *ReflectContext, keepType bool, parent *Schema) (schema Schema, err error) {
	var (
		t          = reflect.TypeOf(i)
		v          = reflect.ValueOf(i)
		s          *Struct
		typeString refl.TypeString
		defName    string
	)

	if st, ok := i.(withStruct); ok {
		s = st.structPtr()
	}

	defer func() {
		rc.Path = rc.Path[:len(rc.Path)-1]

		if t == nil {
			return
		}

		if err != nil {
			return
		}

		schema = r.reflectDefer(defName, typeString, rc, schema, keepType)
	}()

	if t == nil || t == typeOfEmptyInterface {
		return schema, nil
	}

	schema.ReflectType = t
	schema.Parent = parent

	if (t.Kind() == reflect.Ptr && t.Elem() != typeOfJSONRawMsg) || (s != nil && s.Nullable) {
		schema.AddType(Null)
	}

	t = refl.DeepIndirect(t)

	if t == nil || t == typeOfEmptyInterface {
		schema.Type = nil

		return schema, nil
	}

	typeString = refl.GoType(t)
	defName = r.defName(rc, t)

	if s != nil {
		defName, typeString = s.names()
	}

	if mappedTo, found := r.typesMap[t]; found && s == nil {
		t = refl.DeepIndirect(reflect.TypeOf(mappedTo))
		v = reflect.ValueOf(mappedTo)

		if _, ok := mappedTo.(IgnoreTypeName); !ok {
			typeString = refl.GoType(t)
			defName = r.defName(rc, t)
		}
	}

	if len(rc.Path) == 1 {
		rc.rootDefName = defName
	}

	// Shortcut on embedded map or slice.
	if !rc.SkipEmbeddedMapsSlices {
		if et := refl.FindEmbeddedSliceOrMap(i); et != nil {
			t = et
		}
	}

	sp := &schema

	if rc.interceptSchema != nil {
		if ret, err := rc.interceptSchema(InterceptSchemaParams{
			Context:   rc,
			Value:     v,
			Schema:    sp,
			Processed: false,
		}); err != nil || ret {
			return schema, err
		}
	}

	if r.isWellKnownType(t, sp) {
		return schema, nil
	}

	isTextMarshaler := checkTextMarshaler(t, &schema)

	if ref, ok := rc.definitionRefs[typeString]; ok && defName != "" {
		return ref.Schema(), nil
	}

	if rc.typeCycles[typeString] != nil && !rc.InlineRefs {
		return *rc.typeCycles[typeString], nil
	}

	if t.PkgPath() != "" && len(rc.Path) > 1 && defName != "" && !r.inlineDefinition[typeString] {
		rc.typeCycles[typeString] = sp
	}

	r.checkTitle(v, s, sp)

	if err := r.applySubSchemas(v, rc, sp); err != nil {
		return schema, err
	}

	if !isTextMarshaler {
		if err = r.kindSwitch(t, v, sp, rc); err != nil {
			return schema, err
		}
	}

	if rc.interceptSchema != nil {
		if ret, err := rc.interceptSchema(InterceptSchemaParams{
			Context:   rc,
			Value:     v,
			Schema:    sp,
			Processed: true,
		}); err != nil || ret {
			return schema, err
		}
	}

	if preparer, ok := safeInterface(v).(Preparer); ok {
		err := preparer.PrepareJSONSchema(sp)

		return schema, err
	} else if preparer, ok := ptrTo(v).(Preparer); ok {
		err := preparer.PrepareJSONSchema(sp)

		return schema, err
	}

	return schema, nil
}

func checkTextMarshaler(t reflect.Type, schema *Schema) bool {
	if (t.Implements(typeOfTextUnmarshaler) || reflect.PtrTo(t).Implements(typeOfTextUnmarshaler)) &&
		(t.Implements(typeOfTextMarshaler) || reflect.PtrTo(t).Implements(typeOfTextMarshaler)) {
		if !t.Implements(typeOfJSONMarshaler) && !reflect.PtrTo(t).Implements(typeOfJSONMarshaler) {
			schema.TypeEns().WithSimpleTypes(String)
			schema.Type.SliceOfSimpleTypeValues = nil

			return true
		}
	}

	return false
}

func safeInterface(v reflect.Value) interface{} {
	if !v.IsValid() {
		return nil
	}

	if v.Kind() == reflect.Ptr && !v.Elem().IsValid() {
		v = reflect.New(v.Type().Elem())
	}

	return v.Interface()
}

func ptrTo(v reflect.Value) interface{} {
	if !v.IsValid() {
		return nil
	}

	rd := reflect.New(v.Type())
	rd.Elem().Set(v)

	return rd.Interface()
}

func (r *Reflector) applySubSchemas(v reflect.Value, rc *ReflectContext, schema *Schema) error {
	vi := safeInterface(v)
	vp := ptrTo(v)

	var oe OneOfExposer
	if e, ok := vi.(OneOfExposer); ok {
		oe = e
	} else if e, ok := vp.(OneOfExposer); ok {
		oe = e
	}

	if oe != nil {
		var schemas []SchemaOrBool

		for _, item := range oe.JSONSchemaOneOf() {
			rc.Path = append(rc.Path, "oneOf")

			s, err := r.reflect(item, rc, false, schema)
			if err != nil {
				return fmt.Errorf("failed to reflect 'oneOf' values of %T: %w", oe, err)
			}

			schemas = append(schemas, s.ToSchemaOrBool())
		}

		schema.OneOf = schemas
	}

	var ane AnyOfExposer
	if e, ok := vi.(AnyOfExposer); ok {
		ane = e
	} else if e, ok := vp.(AnyOfExposer); ok {
		ane = e
	}

	if ane != nil {
		var schemas []SchemaOrBool

		for _, item := range ane.JSONSchemaAnyOf() {
			rc.Path = append(rc.Path, "anyOf")

			s, err := r.reflect(item, rc, false, schema)
			if err != nil {
				return fmt.Errorf("failed to reflect 'anyOf' values of %T: %w", ane, err)
			}

			schemas = append(schemas, s.ToSchemaOrBool())
		}

		schema.AnyOf = schemas
	}

	var ale AllOfExposer
	if e, ok := vi.(AllOfExposer); ok {
		ale = e
	} else if e, ok := vp.(AllOfExposer); ok {
		ale = e
	}

	if ale != nil {
		var schemas []SchemaOrBool

		for _, item := range ale.JSONSchemaAllOf() {
			rc.Path = append(rc.Path, "allOf")

			s, err := r.reflect(item, rc, false, schema)
			if err != nil {
				return fmt.Errorf("failed to reflect 'allOf' values of %T: %w", ale, err)
			}

			schemas = append(schemas, s.ToSchemaOrBool())
		}

		schema.AllOf = schemas
	}

	var ne NotExposer
	if e, ok := vi.(NotExposer); ok {
		ne = e
	} else if e, ok := vp.(NotExposer); ok {
		ne = e
	}

	if ne != nil {
		rc.Path = append(rc.Path, "not")

		s, err := r.reflect(ne.JSONSchemaNot(), rc, false, schema)
		if err != nil {
			return fmt.Errorf("failed to reflect 'not' value of %T: %w", ne, err)
		}

		schema.WithNot(s.ToSchemaOrBool())
	}

	var ie IfExposer
	if e, ok := vi.(IfExposer); ok {
		ie = e
	} else if e, ok := vp.(IfExposer); ok {
		ie = e
	}

	if ie != nil {
		rc.Path = append(rc.Path, "if")

		s, err := r.reflect(ie.JSONSchemaIf(), rc, false, schema)
		if err != nil {
			return fmt.Errorf("failed to reflect 'if' value of %T: %w", ie, err)
		}

		schema.WithIf(s.ToSchemaOrBool())
	}

	var te ThenExposer
	if e, ok := vi.(ThenExposer); ok {
		te = e
	} else if e, ok := vp.(ThenExposer); ok {
		te = e
	}

	if te != nil {
		rc.Path = append(rc.Path, "if")

		s, err := r.reflect(te.JSONSchemaThen(), rc, false, schema)
		if err != nil {
			return fmt.Errorf("failed to reflect 'then' value of %T: %w", te, err)
		}

		schema.WithThen(s.ToSchemaOrBool())
	}

	var ee ElseExposer
	if e, ok := vi.(ElseExposer); ok {
		ee = e
	} else if e, ok := vp.(ElseExposer); ok {
		ee = e
	}

	if ee != nil {
		rc.Path = append(rc.Path, "if")

		s, err := r.reflect(ee.JSONSchemaElse(), rc, false, schema)
		if err != nil {
			return fmt.Errorf("failed to reflect 'else' value of %T: %w", ee, err)
		}

		schema.WithElse(s.ToSchemaOrBool())
	}

	return nil
}

func (r *Reflector) isWellKnownType(t reflect.Type, schema *Schema) bool {
	ts := refl.GoType(t)

	switch ts {
	case "github.com/google/uuid.UUID", "github.com/gofrs/uuid.UUID", "github.com/gofrs/uuid/v5::uuid.UUID":
		schema.AddType(String)
		schema.WithFormat("uuid")
		schema.WithExamples("248df4b7-aa70-47b8-a036-33ac447e668d")

		return true
	}

	if t == typeOfByteSlice {
		schema.AddType(String)
		schema.WithFormat("base64")

		return true
	}

	if t == typeOfTime {
		schema.AddType(String)
		schema.WithFormat("date-time")

		return true
	}

	if t == typeOfDate {
		schema.AddType(String)
		schema.WithFormat("date")

		return true
	}

	return false
}

var baseNameRegex = regexp.MustCompile(`\[(.+\/)*([^\/]+)·\d+\]`)

func (r *Reflector) defName(rc *ReflectContext, t reflect.Type) string {
	if t.PkgPath() == "" || t == typeOfTime || t == typeOfJSONRawMsg || t == typeOfDate {
		return ""
	}

	if t.Implements(typeOfSchemaInliner) {
		return ""
	}

	if t.Kind() == reflect.Func {
		return ""
	}

	if r.defNameTypes == nil {
		r.defNameTypes = map[string]reflect.Type{}
	}

	var defName string

	try := 1

	for {
		tn := t.Name()
		tn = baseNameRegex.ReplaceAllString(tn, "[$2]")

		if t.PkgPath() == "main" {
			defName = toCamel(strings.Title(tn))
		} else {
			defName = toCamel(path.Base(t.PkgPath()) + strings.Title(tn))
		}

		if rc.DefName != nil {
			defName = rc.DefName(t, defName)
		}

		if try > 1 {
			defName = defName + "Type" + strconv.Itoa(try)
		}

		conflict := false

		for dn, tt := range r.defNameTypes {
			if dn == defName && tt != t {
				conflict = true

				break
			}
		}

		if !conflict {
			r.defNameTypes[defName] = t

			return defName
		}

		try++
	}
}

func (r *Reflector) kindSwitch(t reflect.Type, v reflect.Value, schema *Schema, rc *ReflectContext) error {
	//nolint:exhaustive // Covered with default case.
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

		elemType := t.Elem()

		rc.Path = append(rc.Path, "[]")
		itemValue := reflect.Zero(elemType).Interface()

		if itemValue == nil && elemType != typeOfEmptyInterface {
			itemValue = reflect.New(elemType).Interface()
		}

		for v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		if (v.Kind() == reflect.Slice || v.Kind() == reflect.Array) && v.Len() > 0 {
			itemValue = v.Index(0).Interface()
		}

		itemsSchema, err := r.reflect(itemValue, rc, false, schema)
		if err != nil {
			return err
		}

		schema.AddType(Array)
		schema.WithItems(*(&Items{}).WithSchemaOrBool(itemsSchema.ToSchemaOrBool()))

	case reflect.Map:
		elemType := t.Elem()

		rc.Path = append(rc.Path, "{}")
		itemValue := reflect.Zero(elemType).Interface()

		if itemValue == nil && elemType != typeOfEmptyInterface {
			itemValue = reflect.New(elemType).Interface()
		}

		for v.Kind() == reflect.Ptr {
			v = v.Elem()
		}

		if v.Kind() == reflect.Map {
			rng := v.MapRange()
			for rng.Next() {
				itemValue = rng.Value().Interface()

				break
			}
		}

		additionalPropertiesSchema, err := r.reflect(itemValue, rc, false, schema)
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
		schema.Type = nil
	default:
		if rc.SkipUnsupportedProperties {
			return ErrSkipProperty
		}

		return fmt.Errorf("%s: type is not supported: %s", strings.Join(rc.Path[1:], "."), t.String())
	}

	return nil
}

// MakePropertyNameMapping makes property name mapping from struct value suitable for jsonschema.PropertyNameMapping.
func MakePropertyNameMapping(v interface{}, tagName string) map[string]string {
	res := make(map[string]string)

	refl.WalkTaggedFields(reflect.ValueOf(v), func(_ reflect.Value, sf reflect.StructField, tag string) {
		res[sf.Name] = tag
	}, tagName)

	return res
}

func (r *Reflector) fieldVal(fv reflect.Value, ft reflect.Type) interface{} {
	fieldVal := fv.Interface()

	if ft != typeOfEmptyInterface {
		if ft.Kind() == reflect.Ptr && fv.IsNil() {
			fieldVal = reflect.New(ft.Elem()).Interface()
		} else if ft.Kind() == reflect.Interface && fv.IsNil() {
			fieldVal = reflect.New(ft).Interface()
		}
	}

	return fieldVal
}

func (r *Reflector) propertyTag(rc *ReflectContext, field reflect.StructField) (string, bool) {
	if rc.PropertyNameMapping != nil {
		if tag, tagFound := rc.PropertyNameMapping[field.Name]; tagFound {
			return tag, true
		}
	}

	if tag, tagFound := field.Tag.Lookup(rc.PropertyNameTag); tagFound {
		return tag, true
	}

	for _, t := range rc.PropertyNameAdditionalTags {
		if tag, tagFound := field.Tag.Lookup(t); tagFound {
			return tag, true
		}
	}

	return "", false
}

func (r *Reflector) makeFields(v reflect.Value) ([]reflect.StructField, []reflect.Value) {
	t := v.Type()
	for t.Kind() == reflect.Ptr {
		t = t.Elem()

		if refl.IsZero(v) {
			v = reflect.Zero(t)
		} else {
			v = v.Elem()
		}
	}

	var (
		fields []reflect.StructField
		values []reflect.Value
	)

	isVirtualStruct := false

	if v.CanInterface() {
		if s, ok := v.Interface().(Struct); ok {
			isVirtualStruct = true

			for _, f := range s.Fields {
				field := reflect.StructField{}
				field.Name = f.Name
				field.Tag = f.Tag
				field.Type = reflect.TypeOf(f.Value)

				fields = append(fields, field)
				values = append(values, reflect.ValueOf(f.Value))
			}
		}
	}

	if !isVirtualStruct {
		for i := 0; i < t.NumField(); i++ {
			fields = append(fields, t.Field(i))
			values = append(values, v.Field(i))
		}
	}

	return fields, values
}

func (r *Reflector) walkProperties(v reflect.Value, parent *Schema, rc *ReflectContext) error {
	fields, values := r.makeFields(v)

	for i, field := range fields {
		tag, tagFound := r.propertyTag(rc, field)

		// Skip explicitly discarded field.
		if tag == "-" {
			continue
		}

		deepIndirect := refl.DeepIndirect(field.Type)

		if tag == "" && field.Anonymous &&
			(field.Type.Kind() == reflect.Struct || deepIndirect.Kind() == reflect.Struct) {
			forceReference := (field.Type.Implements(typeOfEmbedReferencer) && field.Tag.Get("refer") == "") ||
				field.Tag.Get("refer") == "true"

			if forceReference {
				rc.Path = append(rc.Path, "")

				s, err := r.reflect(values[i].Interface(), rc, false, parent)
				if err != nil {
					return err
				}

				parent.AllOf = append(parent.AllOf, s.ToSchemaOrBool())
			} else if err := r.walkProperties(values[i], parent, rc); err != nil {
				return err
			}

			continue
		}

		// Use unnamed fields to configure parent schema.
		if field.Name == "_" && (!rc.UnnamedFieldWithTag || tagFound) {
			if err := refl.PopulateFieldsFromTags(parent, field.Tag); err != nil {
				return err
			}

			var additionalProperties *bool
			if err := refl.ReadBoolPtrTag(field.Tag, "additionalProperties", &additionalProperties); err != nil {
				return err
			}

			if additionalProperties != nil {
				parent.AdditionalProperties = &SchemaOrBool{TypeBoolean: additionalProperties}
			}

			if !rc.SkipNonConstraints {
				if err := reflectExamples(rc, parent, field); err != nil {
					return err
				}
			}

			continue
		}

		// Skip the field if tag is not set.
		if !rc.ProcessWithoutTags && !tagFound {
			continue
		}

		// Skip the field if it's non-exported.  There is field.IsExported() method, but it was introduced in go 1.17
		// and will break backward compatibility.
		if field.PkgPath != "" {
			continue
		}

		propName := strings.Split(tag, ",")[0]
		omitEmpty := strings.Contains(tag, ",omitempty")
		required := false

		var nullable *bool

		if propName == "" {
			propName = field.Name
		}

		if err := refl.ReadBoolTag(field.Tag, "required", &required); err != nil {
			return err
		}

		if err := refl.ReadBoolPtrTag(field.Tag, "nullable", &nullable); err != nil {
			return err
		}

		if required {
			parent.Required = append(parent.Required, propName)
		}

		ft := field.Type
		fieldVal := r.fieldVal(values[i], ft)

		rc.Path = append(rc.Path, propName)

		if rc.interceptProp != nil {
			if err := rc.interceptProp(InterceptPropParams{
				Context:      rc,
				Path:         rc.Path,
				Name:         propName,
				Field:        field,
				ParentSchema: parent,
			}); err != nil {
				if errors.Is(err, ErrSkipProperty) {
					rc.Path = rc.Path[:len(rc.Path)-1]

					continue
				}

				return err
			}
		}

		propertySchema, err := r.reflect(fieldVal, rc, true, parent)
		if err != nil {
			if errors.Is(err, ErrSkipProperty) {
				continue
			}

			return err
		}

		checkNullability(&propertySchema, rc, ft, omitEmpty, nullable)

		if !rc.SkipNonConstraints {
			err = checkInlineValue(&propertySchema, field, "default", propertySchema.WithDefault)
			if err != nil {
				return fmt.Errorf("%s: %w", strings.Join(append(rc.Path[1:], field.Name), "."), err)
			}
		}

		err = checkInlineValue(&propertySchema, field, "const", propertySchema.WithConst)
		if err != nil {
			return err
		}

		if err := refl.PopulateFieldsFromTags(&propertySchema, field.Tag); err != nil {
			return err
		}

		deprecated := false
		if err := refl.ReadBoolTag(field.Tag, "deprecated", &deprecated); err != nil {
			return err
		} else if deprecated {
			propertySchema.WithExtraPropertiesItem("deprecated", true)
		}

		if !rc.SkipNonConstraints {
			if err := reflectExamples(rc, &propertySchema, field); err != nil {
				return err
			}
		}

		reflectEnum(&propertySchema, field.Tag, nil)

		// Remove temporary kept type from referenced schema.
		if propertySchema.Ref != nil {
			propertySchema.Type = nil
		}

		if rc.interceptProp != nil {
			if err := rc.interceptProp(InterceptPropParams{
				Context:        rc,
				Path:           rc.Path,
				Name:           propName,
				Field:          field,
				PropertySchema: &propertySchema,
				ParentSchema:   parent,
				Processed:      true,
			}); err != nil {
				if errors.Is(err, ErrSkipProperty) {
					continue
				}

				return err
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

func checkInlineValue(propertySchema *Schema, field reflect.StructField, tag string, setter func(interface{}) *Schema) error {
	var (
		val interface{}
		t   SimpleType

		i *int64
		f *float64
		s *string
		b *bool
	)

	if propertySchema.Type != nil && propertySchema.Type.SimpleTypes != nil {
		t = *propertySchema.Type.SimpleTypes
	}

	_ = refl.ReadIntPtrTag(field.Tag, tag, &i)   //nolint:errcheck
	_ = refl.ReadFloatPtrTag(field.Tag, tag, &f) //nolint:errcheck
	_ = refl.ReadBoolPtrTag(field.Tag, tag, &b)  //nolint:errcheck
	refl.ReadStringPtrTag(field.Tag, tag, &s)

	switch {
	case propertySchema.HasType(Number) && f != nil:
		val = *f
	case propertySchema.HasType(Integer) && i != nil:
		val = *i
	case propertySchema.HasType(Boolean) && b != nil:
		val = *b
	case propertySchema.HasType(String) && s != nil:
		val = *s
	case t == Null:
		// No default for type null.
	default:
		var v string

		refl.ReadStringTag(field.Tag, tag, &v)

		if v == "" {
			break
		}

		err := json.Unmarshal([]byte(v), &val)
		if err == nil {
			break
		}

		if strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]") &&
			propertySchema.Items != nil &&
			propertySchema.Items.SchemaOrBool != nil &&
			propertySchema.Items.SchemaOrBool.TypeObject.HasType(String) {
			val = strings.Split(v[1:len(v)-1], ",")

			break
		}

		return fmt.Errorf("parsing %s as JSON: %w", tag, err)
	}

	if val != nil {
		setter(val)
	}

	return nil
}

// checkNullability checks Go semantic conditions and adds null type to schemas when appropriate.
//
// Presence of `omitempty` field tag disables nullability for the reason that marshaled value
// would be absent instead of having `null`.
//
// Shared definitions (used by $ref) are not nullable by default, so that they can be set to nullable
// where necessary with `"anyOf":[{"type":"null"},{"$ref":"..."}]` (see ReflectContext.EnvelopNullability).
//
// Nullability cases include:
//   - Array, slice accepts `null` as a value.
//   - Object without properties, it is a map, and it accepts `null` as a value.
//   - Pointer type.
func checkNullability(propertySchema *Schema, rc *ReflectContext, ft reflect.Type, omitEmpty bool, nullable *bool) {
	in := InterceptNullabilityParams{
		Context:    rc,
		OrigSchema: *propertySchema,
		Schema:     propertySchema,
		Type:       ft,
		OmitEmpty:  omitEmpty,
	}

	defer func() {
		if rc.InterceptNullability != nil {
			rc.InterceptNullability(in)
		}
	}()

	if nullable != nil {
		if *nullable {
			propertySchema.AddType(Null)

			in.NullAdded = true
		} else if propertySchema.Ref == nil && propertySchema.HasType(Null) {
			propertySchema.RemoveType(Null)

			in.NullAdded = false
		}

		return
	}

	if omitEmpty {
		return
	}

	if propertySchema.HasType(Array) ||
		(propertySchema.HasType(Object) && len(propertySchema.Properties) == 0 && propertySchema.Ref == nil) {
		propertySchema.AddType(Null)

		in.NullAdded = true
	}

	if ft.Kind() == reflect.Ptr && propertySchema.Ref == nil && ft.Elem() != typeOfJSONRawMsg {
		propertySchema.AddType(Null)

		in.NullAdded = true
	}

	if propertySchema.Ref != nil && ft.Kind() != reflect.Struct {
		def := rc.getDefinition(*propertySchema.Ref)
		in.RefDef = def

		if (def.HasType(Array) || def.HasType(Object) || ft.Kind() == reflect.Ptr) && !def.HasType(Null) {
			if rc.EnvelopNullability {
				refSchema := *propertySchema
				propertySchema.Ref = nil
				propertySchema.AnyOf = []SchemaOrBool{
					Null.ToSchemaOrBool(),
					refSchema.ToSchemaOrBool(),
				}
			}
		}
	}
}

func reflectExamples(rc *ReflectContext, propertySchema *Schema, field reflect.StructField) error {
	if err := reflectExample(rc, propertySchema, field); err != nil {
		return err
	}

	value, ok := field.Tag.Lookup("examples")
	if !ok {
		return nil
	}

	var val []interface{}
	if err := json.Unmarshal([]byte(value), &val); err != nil {
		return fmt.Errorf("failed to parse examples in field %s: %w", field.Name, err)
	}

	propertySchema.Examples = append(propertySchema.Examples, val...)

	return nil
}

func reflectExample(rc *ReflectContext, propertySchema *Schema, field reflect.StructField) error {
	err := checkInlineValue(propertySchema, field, "example", func(i interface{}) *Schema {
		return propertySchema.WithExamples(i)
	})
	if err != nil {
		return fmt.Errorf("%s: %w", strings.Join(append(rc.Path[1:], field.Name), "."), err)
	}

	return nil
}

func reflectEnum(schema *Schema, fieldTag reflect.StructTag, fieldVal interface{}) {
	enum := enum{}
	enum.loadFromField(fieldTag, fieldVal)

	if len(enum.items) > 0 {
		schema.Enum = enum.items
		if len(enum.names) > 0 {
			if schema.ExtraProperties == nil {
				schema.ExtraProperties = make(map[string]interface{}, 1)
			}

			schema.ExtraProperties[XEnumNames] = enum.names
		}
	}
}

// enum can be use for sending enum data that need validate.
type enum struct {
	items []interface{}
	names []string
}

// loadFromField loads enum from field tag: json array or comma-separated string.
func (enum *enum) loadFromField(fieldTag reflect.StructTag, fieldVal interface{}) {
	fv := reflect.ValueOf(fieldVal)

	if e, isEnumer := safeInterface(fv).(NamedEnum); isEnumer {
		enum.items, enum.names = e.NamedEnum()
	} else if e, isEnumer := ptrTo(fv).(NamedEnum); isEnumer {
		enum.items, enum.names = e.NamedEnum()
	}

	if e, isEnumer := safeInterface(fv).(Enum); isEnumer {
		enum.items = e.Enum()
	} else if e, isEnumer := ptrTo(fv).(Enum); isEnumer {
		enum.items = e.Enum()
	}

	if enumTag := fieldTag.Get("enum"); enumTag != "" {
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

type (
	oneOf []interface{}
	allOf []interface{}
	anyOf []interface{}
)

var (
	_ Preparer = oneOf{}
	_ Preparer = anyOf{}
	_ Preparer = allOf{}
)

// OneOf exposes list of values as JSON "oneOf" schema.
func OneOf(v ...interface{}) OneOfExposer {
	return oneOf(v)
}

// PrepareJSONSchema removes unnecessary constraints.
func (oneOf) PrepareJSONSchema(schema *Schema) error {
	schema.Type = nil
	schema.Items = nil

	return nil
}

// JSONSchemaOneOf implements OneOfExposer.
func (o oneOf) JSONSchemaOneOf() []interface{} {
	return o
}

// InlineJSONSchema implements SchemaInliner.
func (o oneOf) InlineJSONSchema() {}

// AnyOf exposes list of values as JSON "anyOf" schema.
func AnyOf(v ...interface{}) AnyOfExposer {
	return anyOf(v)
}

// PrepareJSONSchema removes unnecessary constraints.
func (anyOf) PrepareJSONSchema(schema *Schema) error {
	schema.Type = nil
	schema.Items = nil

	return nil
}

// JSONSchemaAnyOf implements AnyOfExposer.
func (o anyOf) JSONSchemaAnyOf() []interface{} {
	return o
}

// InlineJSONSchema implements SchemaInliner.
func (o anyOf) InlineJSONSchema() {}

// AllOf exposes list of values as JSON "allOf" schema.
func AllOf(v ...interface{}) AllOfExposer {
	return allOf(v)
}

// PrepareJSONSchema removes unnecessary constraints.
func (allOf) PrepareJSONSchema(schema *Schema) error {
	schema.Type = nil
	schema.Items = nil

	return nil
}

// JSONSchemaAllOf implements AllOfExposer.
func (o allOf) JSONSchemaAllOf() []interface{} {
	return o
}

// InlineJSONSchema implements SchemaInliner.
func (o allOf) InlineJSONSchema() {}
