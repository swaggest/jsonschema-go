# JSON Schema structures for Go

[![Build Status](https://travis-ci.org/swaggest/jsonschema-go.svg?branch=master)](https://travis-ci.org/swaggest/jsonschema-go)
[![Coverage Status](https://codecov.io/gh/swaggest/jsonschema-go/branch/master/graph/badge.svg)](https://codecov.io/gh/swaggest/jsonschema-go)
[![GoDevDoc](https://img.shields.io/badge/dev-doc-00ADD8?logo=go)](https://pkg.go.dev/github.com/swaggest/jsonschema-go)
![Code lines](https://sloc.xyz/github/swaggest/jsonschema-go/?category=code)
![Comments](https://sloc.xyz/github/swaggest/jsonschema-go/?category=comments)

This library provides Go structures to marshal/unmarshal and reflect [JSON Schema](https://json-schema.org/) documents.

## Reflector

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