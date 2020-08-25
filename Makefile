GOLANGCI_LINT_VERSION := "v1.30.0"

gen:
	cd resources/schema/ && json-cli gen-go draft-07.json --output ../../entities.go --package-name jsonschema --with-zero-values --fluent-setters --enable-default-additional-properties --with-tests --root-name SchemaOrBool \
		--renames CoreSchemaMetaSchema:Schema SimpleTypes:SimpleType SimpleTypeArray:Array SimpleTypeBoolean:Boolean SimpleTypeInteger:Integer SimpleTypeNull:Null SimpleTypeNumber:Number SimpleTypeObject:Object SimpleTypeString:String
	gofmt -w ./entities.go ./entities_test.go

lint:
	@test -s $(GOPATH)/bin/golangci-lint-$(GOLANGCI_LINT_VERSION) || (curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b /tmp $(GOLANGCI_LINT_VERSION) && mv /tmp/golangci-lint $(GOPATH)/bin/golangci-lint-$(GOLANGCI_LINT_VERSION))
	@$(GOPATH)/bin/golangci-lint-$(GOLANGCI_LINT_VERSION) run ./... --fix
