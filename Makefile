gen:
	cd resources/schema/ && json-cli gen-go draft-07.json --output ../../entities.go --package-name jsonschema --with-zero-values --fluent-setters --enable-default-additional-properties --with-tests --root-name SchemaOrBool \
		--renames CoreSchemaMetaSchema:Schema SimpleTypes:SimpleType SimpleTypeArray:Array SimpleTypeBoolean:Boolean SimpleTypeInteger:Integer SimpleTypeNull:Null SimpleTypeNumber:Number SimpleTypeObject:Object SimpleTypeString:String
	gofmt -w ./entities.go ./entities_test.go

lint:
	golangci-lint run --enable-all --disable gochecknoglobals,funlen,gomnd,gocognit ./...
