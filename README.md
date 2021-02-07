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
    Amount float64 `json:"amount" minimum:"10.5" example:"20.6" required:"true"`
    Abc    string  `json:"abc" pattern:"[abc]"`
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
//  "required": [
//   "amount"
//  ],
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
