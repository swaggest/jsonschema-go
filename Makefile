gen:
	cd resources/schema/ && json-cli gen-go draft-07.json --output ../../draft-07/entities.go --package-name jsonschema --root-name Schema
	gofmt -w ./draft-07/entities.go
