package testingdb

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/script-development/RT-CV/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type filter struct {
	filters bson.M
	empty   bool
}

func newFilter(filters bson.M) *filter {
	res := &filter{
		filters: filters,
	}
	if len(res.filters) == 0 {
		res.empty = true

		// if filters is nil
		res.filters = bson.M{}
	}

	return res
}

func (f *filter) matches(e db.Entry) bool {
	if f.empty {
		return true
	}

	eRefl := reflect.ValueOf(e).Elem()
	eFieldsMap := mapStruct(eRefl.Type())

	for key, value := range f.filters {
		if strings.HasPrefix("$", key) {
			panic("FIXME implement custom filter MongoDB filter properties")
		}

		field, fieldFound := eFieldsMap[key]
		if !fieldFound {
			return false
		}

		eReflPath := eRefl
		for _, pathPart := range field.GoPathToField {
			eReflPath = eReflPath.FieldByName(pathPart)
		}
		entryField := eRefl.FieldByName(field.GoFieldName)

		if entryField.Kind() == reflect.Ptr {
			if entryField.IsNil() {
				return false
			}
			entryField = entryField.Elem()
		}

		reflectionValue := reflect.ValueOf(value)
		switch reflectionValue.Kind() {
		case reflect.String:
			if entryField.Kind() != reflect.String || entryField.String() != reflectionValue.String() {
				return false
			}
		case reflect.Bool:
			if entryField.Kind() != reflect.Bool || entryField.Bool() != reflectionValue.Bool() {
				return false
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if !compareInt64ToReflect(reflectionValue.Int(), entryField) {
				return false
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if !compareUint64ToReflect(reflectionValue.Uint(), entryField) {
				return false
			}
		default:
			valueObjectID, ok := value.(primitive.ObjectID)
			if ok {
				goFieldValue, ok := entryField.Interface().(primitive.ObjectID)
				if !ok {
					return false
				}
				if goFieldValue != valueObjectID {
					return false
				}
			} else {
				panic(fmt.Sprintf("Unimplemented value filter type: %T, key: %v, value: %#v", value, key, value))
			}
		}
	}

	return true
}

type structField struct {
	// incase of a inline field we need to resolve the field within another struct
	GoPathToField []string

	GoFieldName string
	DbFieldName string
}

func mapStruct(entry reflect.Type) map[string]structField {
	if entry.Kind() != reflect.Struct {
		panic("expected struct but got " + entry.Kind().String())
	}

	res := map[string]structField{}
	for i := 0; i < entry.NumField(); i++ {
		mapStructField(entry.Field(i), func(field structField) {
			res[field.DbFieldName] = field
		})
	}
	return res
}

func mapStructField(field reflect.StructField, add func(structField)) {
	bsonTag := field.Tag.Get("bson")
	if bsonTag == "" {
		bsonTag = field.Tag.Get("json")
	}

	values := strings.Split(bsonTag, ",")
	dbName := values[0]
	if dbName == "" {
		dbName = convertGoToDbName(field.Name)
	}

	isInlineField := false
	if len(values) > 1 {
		for _, entry := range values[1:] {
			if entry == "inline" && field.Type.Kind() == reflect.Struct {
				isInlineField = true
			}
		}
	}

	if isInlineField {
		for i := 0; i < field.Type.NumField(); i++ {
			mapStructField(field.Type.Field(i), func(toAdd structField) {
				toAdd.GoPathToField = append(toAdd.GoPathToField, field.Name)
				add(toAdd)
			})
		}
	} else {
		add(structField{
			GoPathToField: []string{},
			GoFieldName:   field.Name,
			DbFieldName:   dbName,
		})
	}
}

func convertGoToDbName(fieldname string) string {
	// No need to check if filename length is > 0 beaucase go field name always have a name
	return string(unicode.ToLower(rune(fieldname[0]))) + fieldname[1:]
}

func compareInt64ToReflect(value int64, reflectionValue reflect.Value) bool {
	switch reflectionValue.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflectionValue.Int() == value
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		reflectionIntValue := int64(reflectionValue.Uint())
		if reflectionIntValue < 0 {
			// The uint64 value of the reflect value was more than the highest int64 value and thus resetted itself and now it's below zero
			return false
		}
		return reflectionIntValue == value
	default:
		return false
	}
}

func compareUint64ToReflect(value uint64, reflectionValue reflect.Value) bool {
	switch reflectionValue.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intValue := reflectionValue.Int()
		if intValue < 0 {
			return false
		}
		return uint64(intValue) == value
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return reflectionValue.Uint() == value
	default:
		return false
	}
}
