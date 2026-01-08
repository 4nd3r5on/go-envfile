package envfile

import (
	"fmt"
	"reflect"

	"github.com/4nd3r5on/go-envfile/common"
	"github.com/4nd3r5on/go-envfile/updater"
)

// UpdatesFromStruct converts a struct into a slice of updater.Update entries.
// It uses struct tags to determine environment variable names and sections:
// - envNameTag (default "env"): specifies the environment variable name
// - sectionTag (default "section"): specifies the configuration section
// If the env tag is absent, field names are converted to UPPER_SNAKE_CASE.
// Fields tagged with "-" are skipped.
func UpdatesFromStruct(
	data any,
	envNameTag, sectionTag string,
) []updater.Update {
	envNameTag = defaultString(envNameTag, "env")
	sectionTag = defaultString(sectionTag, "section")

	v := unwrapToStruct(data)
	if v == nil {
		return nil
	}

	var updates []updater.Update
	walkStruct(*v, func(field reflectField) {
		if update := fieldToUpdate(field, envNameTag, sectionTag); update != nil {
			updates = append(updates, *update)
		}
	})

	return updates
}

type reflectField struct {
	value reflect.Value
	typ   reflect.StructField
}

// unwrapToStruct dereferences pointers and validates the result is a struct
func unwrapToStruct(data any) *reflect.Value {
	v := reflect.ValueOf(data)
	if !v.IsValid() {
		return nil
	}

	// Dereference all pointer layers
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil
	}

	return &v
}

// walkStruct recursively traverses struct fields, calling the visitor function
func walkStruct(v reflect.Value, visitor func(reflectField)) {
	t := v.Type()

	for i := range v.NumField() {
		fieldVal := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields
		if fieldType.PkgPath != "" {
			continue
		}

		// Recurse into embedded structs
		if fieldType.Anonymous && fieldVal.Kind() == reflect.Struct {
			walkStruct(fieldVal, visitor)
			continue
		}

		visitor(reflectField{value: fieldVal, typ: fieldType})
	}
}

// fieldToUpdate converts a struct field to an Update, or returns nil if skipped
func fieldToUpdate(field reflectField, envNameTag, sectionTag string) *updater.Update {
	// Determine environment variable name
	key := field.typ.Tag.Get(envNameTag)
	if key == "-" {
		return nil
	}
	if key == "" {
		key = common.ToUpperSnake(field.typ.Name)
	}

	section := field.typ.Tag.Get(sectionTag)

	return &updater.Update{
		Key:     key,
		Value:   fmt.Sprint(field.value.Interface()),
		Section: section,
	}
}

// defaultString returns the value if non-empty, otherwise returns the default
func defaultString(value, defaultVal string) string {
	if value == "" {
		return defaultVal
	}
	return value
}
