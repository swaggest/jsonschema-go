gen:
	cd resources/schema/ && json-cli gen-go draft-07.json --output ../../draft-07/entities.go --package-name jsonschema --with-zero-values --fluent-setters --enable-default-additional-properties --with-tests --root-name SchemaOrBool \
		--renames CoreSchemaMetaSchema:Schema SimpleTypes:SimpleType SimpleTypeArray:Array SimpleTypeBoolean:Boolean SimpleTypeInteger:Integer SimpleTypeNull:Null SimpleTypeNumber:Number SimpleTypeObject:Object SimpleTypeString:String
	gofmt -w ./draft-07/entities.go ./draft-07/entities_test.go

gen-oas3:
	cd resources/schema/ && json-cli gen-go openapi3.json --output ../../openapi3/entities.go --package-name openapi3 --with-zero-values --fluent-setters --root-name Spec
	gofmt -w ./openapi3/entities.go
