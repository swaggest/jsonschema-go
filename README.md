# JSON Schema structures for Go

<img align="right" width="100px" src="https://avatars0.githubusercontent.com/u/13019229?s=200&v=4">

[![Build Status](https://github.com/swaggest/jsonschema-go/workflows/test-unit/badge.svg)](https://github.com/swaggest/jsonschema-go/actions?query=branch%3Amaster+workflow%3Atest-unit)
[![Coverage Status](https://codecov.io/gh/swaggest/jsonschema-go/branch/master/graph/badge.svg)](https://codecov.io/gh/swaggest/jsonschema-go)
[![GoDevDoc](https://img.shields.io/badge/dev-doc-00ADD8?logo=go)](https://pkg.go.dev/github.com/swaggest/jsonschema-go)
[![time tracker](https://wakatime.com/badge/github/swaggest/jsonschema-go.svg)](https://wakatime.com/badge/github/swaggest/jsonschema-go)
![Code lines](https://sloc.xyz/github/swaggest/jsonschema-go/?category=code)
![Comments](https://sloc.xyz/github/swaggest/jsonschema-go/?category=comments)

This library provides Go structures to marshal/unmarshal and reflect [JSON Schema](https://json-schema.org/) documents.

## Reflector

[Documentation](https://pkg.go.dev/github.com/swaggest/jsonschema-go#Reflector.Reflect).

```go
type MyStruct struct {
    Amount float64  `json:"amount" minimum:"10.5" example:"20.6" required:"true"`
    Abc    string   `json:"abc" pattern:"[abc]"`
    _      struct{} `additionalProperties:"false"`                   // Tags of unnamed field are applied to parent schema.
    _      struct{} `title:"My Struct" description:"Holds my data."` // Multiple unnamed fields can be used.
}

reflector := jsonschema.Reflector{}

schema, err := reflector.Reflect(MyStruct{})
if err != nil {
    log.Fatal(err)
}

j, err := json.MarshalIndent(schema, "", " ")
if err != nil {
    log.Fatal(err)
}

fmt.Println(string(j))

// Output:
// {
//  "title": "My Struct",
//  "description": "Holds my data.",
//  "required": [
//   "amount"
//  ],
//  "additionalProperties": false,
//  "properties": {
//   "abc": {
//    "pattern": "[abc]",
//    "type": "string"
//   },
//   "amount": {
//    "examples": [
//     20.6
//    ],
//    "minimum": 10.5,
//    "type": "number"
//   }
//  },
//  "type": "object"
// }
```

## Customization

By default, JSON Schema is generated from Go struct field types and tags.
It works well for the majority of cases, but if it does not there are rich customization options.

### Field tags

```go
type MyObj struct {
   BoundedNumber int `query:"boundedNumber" minimum:"-100" maximum:"100"`
   SpecialString string `json:"specialString" pattern:"^[a-z]{4}$" minLength:"4" maxLength:"4"`
}
```

Note: field tags are only applied to inline schemas, if you use named type then referenced schema
will be created and tags will be ignored. This happens because referenced schema can be used in
multiple fields with conflicting tags, therefore customization of referenced schema has to done on
the type itself via `RawExposer`, `Exposer` or `Preparer`.

Each tag value has to be put in double quotes (`"123"`).

These tags can be used:
* [`title`](https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.6.1), string
* [`description`](https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.6.1), string
* [`default`](https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.6.2), can be scalar or JSON value
* [`example`](https://json-schema.org/draft/2020-12/json-schema-validation.html#name-examples), a scalar value that matches type of parent property, for an array it is applied to items
* [`examples`](https://json-schema.org/draft/2020-12/json-schema-validation.html#name-examples), a JSON array value
* [`const`](https://json-schema.org/draft/2020-12/json-schema-validation.html#rfc.section.6.1.3), can be scalar or JSON value
* [`pattern`](https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.2.3), string
* [`format`](https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.7), string
* [`multipleOf`](https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.1.1), float > 0
* [`maximum`](https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.1.2), float
* [`minimum`](https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.1.3), float
* [`maxLength`](https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.2.1), integer
* [`minLength`](https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.2.2), integer
* [`maxItems`](https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.3.2), integer
* [`minItems`](https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.3.3), integer
* [`maxProperties`](https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.4.1), integer
* [`minProperties`](https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.4.2), integer
* [`exclusiveMaximum`](https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.1.2), boolean
* [`exclusiveMinimum`](https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.1.3), boolean
* [`uniqueItems`](https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.3.4), boolean
* [`enum`](https://json-schema.org/draft-04/json-schema-validation.html#rfc.section.5.5.1), tag value must be a JSON or comma-separated list of strings
* `required`, boolean, marks property as required
* `nullable`, boolean, overrides nullability of the property

Unnamed fields can be used to configure parent schema:

```go
type MyObj struct {
   BoundedNumber int `query:"boundedNumber" minimum:"-100" maximum:"100"`
   SpecialString string `json:"specialString" pattern:"^[a-z]{4}$" minLength:"4" maxLength:"4"`
   _             struct{} `additionalProperties:"false" description:"MyObj is my object."`
}
```

In case of a structure with multiple name tags, you can enable filtering of unnamed fields with
ReflectContext.UnnamedFieldWithTag option and add matching name tags to structure (e.g. query:"_").

```go
type MyObj struct {
   BoundedNumber int `query:"boundedNumber" minimum:"-100" maximum:"100"`
   SpecialString string `json:"specialString" pattern:"^[a-z]{4}$" minLength:"4" maxLength:"4"`
   // These parent schema tags would only be applied to `query` schema reflection (not for `json`).
   _ struct{} `query:"_" additionalProperties:"false" description:"MyObj is my object."`
}
```

### Implementing interfaces on a type

There are a few interfaces that can be implemented on a type to customize JSON Schema generation.

* [`Preparer`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#Preparer) allows to change generated JSON Schema.
* [`Exposer`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#Exposer) overrides generated JSON Schema.
* [`RawExposer`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#RawExposer) overrides generated JSON Schema.
* [`Described`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#Described) exposes description.
* [`Titled`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#Titled) exposes title.
* [`Enum`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#Enum) exposes enum values.
* [`NamedEnum`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#NamedEnum) exposes enum values with names.
* [`SchemaInliner`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#SchemaInliner) inlines schema without creating a definition.
* [`IgnoreTypeName`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#IgnoreTypeName), when implemented on a mapped type forces the use of original type for definition name.
* [`EmbedReferencer`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#EmbedReferencer), when implemented on an embedded field type, makes an `allOf` reference to that type definition.

And a few interfaces to expose subschemas (`anyOf`, `allOf`, `oneOf`, `not` and `if`, `then`, `else`).
* [`AnyOfExposer`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#AnyOfExposer) exposes `anyOf` subschemas.
* [`AllOfExposer`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#AllOfExposer) exposes `allOf` subschemas.
* [`OneOfExposer`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#OneOfExposer) exposes `oneOf` subschemas.
* [`NotExposer`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#NotExposer) exposes `not` subschema.
* [`IfExposer`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#IfExposer) exposes `if` subschema.
* [`ThenExposer`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#ThenExposer) exposes `then` subschema.
* [`ElseExposer`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#ElseExposer) exposes `else` subschema.

There are also helper functions 
[`jsonschema.AllOf`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#AllOf), 
[`jsonschema.AnyOf`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#AnyOf), 
[`jsonschema.OneOf`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#OneOf) 
to create exposer instance from multiple values.



### Configuring the reflector

Additional centralized configuration is available with 
[`jsonschema.ReflectContext`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#ReflectContext) and 
[`Reflect`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#Reflector.Reflect) options.

* [`CollectDefinitions`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#CollectDefinitions) disables definitions storage in schema and calls user function instead.
* [`DefinitionsPrefix`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#DefinitionsPrefix) sets path prefix for definitions.
* [`PropertyNameTag`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#PropertyNameTag) allows using field tags other than `json`.
* [`InterceptSchema`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#InterceptSchema) called for every type during schema reflection.
* [`InterceptProp`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#InterceptProp) called for every property during schema reflection.
* [`InlineRefs`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#InlineRefs) tries to inline all references (instead of creating definitions).
* [`RootNullable`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#RootNullable) enables nullability of root schema.
* [`RootRef`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#RootRef) converts root schema to definition reference.
* [`StripDefinitionNamePrefix`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#StripDefinitionNamePrefix) strips prefix from definition name.
* [`PropertyNameMapping`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#PropertyNameMapping) explicit name mapping instead field tags.
* [`ProcessWithoutTags`](https://pkg.go.dev/github.com/swaggest/jsonschema-go#ProcessWithoutTags) enables processing fields without any tags specified.

### Virtual structure

Sometimes it is impossible to define a static Go `struct`, for example when fields are only known at runtime.
Yet, you may need to include such fields in JSON schema reflection pipeline.

For any reflected value, standalone or nested, you can define a virtual structure that would be treated as a native Go struct.

```go
s := jsonschema.Struct{}
s.SetTitle("Test title")
s.SetDescription("Test description")
s.DefName = "TestStruct"
s.Nullable = true

s.Fields = append(s.Fields, jsonschema.Field{
    Name:  "Foo",
    Value: "abc",
    Tag:   `json:"foo" minLength:"3"`,
})

r := jsonschema.Reflector{}
schema, _ := r.Reflect(s)
j, _ := assertjson.MarshalIndentCompact(schema, "", " ", 80)

fmt.Println("Standalone:", string(j))

type MyStruct struct {
    jsonschema.Struct // Can be embedded.

    Bar int `json:"bar"`

    Nested jsonschema.Struct `json:"nested"` // Can be nested.
}

ms := MyStruct{}
ms.Nested = s
ms.Struct = s

schema, _ = r.Reflect(ms)
j, _ = assertjson.MarshalIndentCompact(schema, "", " ", 80)

fmt.Println("Nested:", string(j))

// Output:
// Standalone: {
//  "title":"Test title","description":"Test description",
//  "properties":{"foo":{"minLength":3,"type":"string"}},"type":"object"
// }
// Nested: {
//  "definitions":{
//   "TestStruct":{
//    "title":"Test title","description":"Test description",
//    "properties":{"foo":{"minLength":3,"type":"string"}},"type":"object"
//   }
//  },
//  "properties":{
//   "bar":{"type":"integer"},"foo":{"minLength":3,"type":"string"},
//   "nested":{"$ref":"#/definitions/TestStruct"}
//  },
//  "type":"object"
// }
```

### Custom Tags For Schema Definitions

If you're using additional libraries for validation, like for example 
[`go-playground/validator`](https://github.com/go-playground/validator), you may want to infer validation rules into 
documented JSON schema.

```go
type My struct {
    Foo *string `json:"foo" validate:"required"`
}
```

Normally, `validate:"required"` is not recognized, and you'd need to add `required:"true"` to have the rule exported to 
JSON schema.

However, it is possible to extend reflection with custom processing with `InterceptProp` option.

```go
s, err := r.Reflect(My{}, jsonschema.InterceptProp(func(params jsonschema.InterceptPropParams) error {
    if !params.Processed {
        return nil
    }

    if v, ok := params.Field.Tag.Lookup("validate"); ok {
        if strings.Contains(v, "required") {
            params.ParentSchema.Required = append(params.ParentSchema.Required, params.Name)
        }
    }

    return nil
}))
```