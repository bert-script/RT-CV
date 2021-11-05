package testingdb

import (
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type filter struct {
	filters reflect.Value
	empty   bool
}

func newFilter(filters bson.M) *filter {
	return &filter{
		filters: reflect.ValueOf(filters),
		empty:   len(filters) == 0,
	}
}

func (f *filter) matches(value interface{}) bool {
	if f.empty {
		return true
	}

	valueReflection := reflect.ValueOf(value)
	for valueReflection.Kind() == reflect.Ptr {
		valueReflection = valueReflection.Elem()
	}

	return filterMatchesValue(f.filters, valueReflection)
}

var timeType = reflect.TypeOf(time.Time{})

func filterMatchesValue(filterMap reflect.Value, value reflect.Value) bool {
	for value.Kind() == reflect.Interface && !value.IsNil() {
		value = value.Elem()
	}

	valueFieldsMap, valueIsStruct := mapStruct(value.Type())

	iter := filterMap.MapRange()

filtersLoop:
	for iter.Next() {
		// FIXME we assume the key is a string, maybe we should support also other values
		key := iter.Key().String()
		filter := iter.Value()
		if filter.Kind() == reflect.Interface {
			filter = filter.Elem()
		}

		for {
			if filter.Kind() != reflect.Ptr || filter.IsNil() {
				break
			}
			filter = filter.Elem()
		}

		if strings.HasPrefix(key, "$") {
			for value.Kind() == reflect.Ptr && !value.IsNil() {
				value = value.Elem()
			}

			switch key {
			case "$gt":
				if filter.Type().ConvertibleTo(timeType) {
					if !value.Type().ConvertibleTo(timeType) {
						return false
					}
					if value.Convert(timeType).Interface().(time.Time).Before(filter.Convert(timeType).Interface().(time.Time)) {
						return false
					}
				} else if !compareNumbers(numComparisonGreater, value, filter) {
					return false
				}
			case "$gte":
				if filter.Type().ConvertibleTo(timeType) {
					if !value.Type().ConvertibleTo(timeType) {
						return false
					}
					if value.Convert(timeType).Interface().(time.Time).Before(filter.Convert(timeType).Interface().(time.Time)) {
						return false
					}
				} else if !compareNumbers(numComparisonGreaterOrEqual, value, filter) {
					return false
				}
			case "$lt":
				if filter.Type().ConvertibleTo(timeType) {
					if !value.Type().ConvertibleTo(timeType) {
						return false
					}
					if value.Convert(timeType).Interface().(time.Time).After(filter.Convert(timeType).Interface().(time.Time)) {
						return false
					}
				} else if !compareNumbers(numComparisonLess, value, filter) {
					return false
				}
			case "$lte":
				if filter.Type().ConvertibleTo(timeType) {
					if !value.Type().ConvertibleTo(timeType) {
						return false
					}
					if value.Convert(timeType).Interface().(time.Time).After(filter.Convert(timeType).Interface().(time.Time)) {
						return false
					}
				} else if !compareNumbers(numComparisonLessOrEqual, value, filter) {
					return false
				}
			case "$eq":
				if !filterCompare(filter, value) {
					return false
				}
			case "$not", "$ne":
				if filterCompare(filter, value) {
					return false
				}
			case "$or":
				if filter.Kind() != reflect.Slice && filter.Kind() != reflect.Array {
					return false
				}

				foundOk := false
				for i := 0; i < filter.Len(); i++ {
					orEntry := filter.Index(i)
					if orEntry.Kind() != reflect.Map || orEntry.IsNil() {
						// TODO maybe we should also support structs here
						continue
					}
					assertMapHasStringKeys(orEntry.Type())
					if filterMatchesValue(orEntry, value) {
						foundOk = true
						break
					}
				}
				if !foundOk {
					return false
				}
			case "$and":
				if filter.Kind() != reflect.Slice && filter.Kind() != reflect.Array {
					return false
				}

				for i := 0; i < filter.Len(); i++ {
					andEntry := filter.Index(i)
					if andEntry.Kind() != reflect.Map || andEntry.IsNil() {
						// TODO maybe we should also support structs here
						continue
					}
					assertMapHasStringKeys(andEntry.Type())
					if !filterMatchesValue(andEntry, value) {
						return false
					}
				}
			case "$size":
				var expectedSize int
				switch filter.Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					expectedSize = int(filter.Int())
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					expectedSize = int(filter.Uint())
				default:
					panic("$size should have a number as argument")
				}

				if value.Kind() != reflect.Slice && value.Kind() != reflect.Array {
					return false
				}
				if value.IsNil() {
					return false
				}
				if value.Len() != expectedSize {
					return false
				}
			case "$type":
				var expectedType string
				intToTypeName := map[int64]string{
					1:  "double",
					2:  "string",
					3:  "object",
					4:  "array",
					8:  "bool",
					10: "null",
					16: "int",
					19: "decimal",
				}

				switch filter.Kind() {
				case reflect.String:
					expectedType = filter.String()
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					expectedType = intToTypeName[filter.Int()]
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					expectedType = intToTypeName[int64(filter.Uint())]
				default:
					panic("$type should have a string or number as argument")
				}

				valKind := value.Kind()
				switch expectedType {
				case "decimal", "double":
					if valKind != reflect.Float64 && valKind != reflect.Float32 {
						return false
					}
				case "int":
					switch valKind {
					case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
					default:
						return false
					}
				case "string":
					if valKind != reflect.String {
						return false
					}
				case "object":
					if valKind == reflect.Map {
						if value.IsNil() {
							return false
						}
					} else if valKind != reflect.Struct {
						return false
					}
				case "array":
					if valKind != reflect.Slice && valKind != reflect.Array {
						return false
					}
					if value.IsNil() {
						return false
					}
				case "bool":
					if valKind != reflect.Bool {
						return false
					}
				case "null":
					switch valKind {
					case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
						if !value.IsNil() {
							return false
						}
					default:
						return false
					}
				default:
					return false
				}
			default:
				// For docs see:
				// https://docs.mongodb.com/manual/reference/operator/query/
				panic("FIXME unimplemented custom MongoDB filter " + key)
			}
			continue
		}

		if !valueIsStruct {
			return false
		}

		field, fieldFound := valueFieldsMap[key]
		if !fieldFound {
			return false
		}

		tempValueCopy := value
		for _, goPathPart := range field.GoPathToField {
			tempValueCopy = tempValueCopy.FieldByName(goPathPart)
		}
		valueField := tempValueCopy.FieldByName(field.GoFieldName)

		if !filter.IsValid() {
			// filter is probably a nil interface{}
			// note that isNil panics if the value is a nil interface without a type
			// so we check here for: interface{}(nil)
			// and not: interface{}([]string(nil))
			switch valueField.Kind() {
			case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
				if valueField.IsNil() {
					continue filtersLoop
				}
			}
			return false
		}

		if !filterCompare(filter, valueField) {
			return false
		}
	}

	return true
}

func filterCompare(filter, value reflect.Value) bool {
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return false
		}
		value = value.Elem()
	}

	switch filter.Kind() {
	case reflect.String:
		if value.Kind() != reflect.String || value.String() != filter.String() {
			return false
		}
	case reflect.Bool:
		if value.Kind() != reflect.Bool || value.Bool() != filter.Bool() {
			return false
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fallthrough
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if !compareNumbers(numComparisonEqual, filter, value) {
			return false
		}
	case reflect.Map:
		assertMapHasStringKeys(filter.Type())
		return filterMatchesValue(filter, value)
	default:
		filterValue := filter.Interface()
		if filterObjectID, ok := filterValue.(primitive.ObjectID); ok {
			goFieldValue, ok := value.Interface().(primitive.ObjectID)
			if !ok {
				return false
			}
			if goFieldValue != filterObjectID {
				return false
			}
		} else {
			panic(fmt.Sprintf("Unimplemented value filter type: %T, value: %#v", filterValue, filterValue))
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

func mapStruct(entry reflect.Type) (structEntries map[string]structField, isStruct bool) {
	if entry.Kind() != reflect.Struct {
		return nil, false
	}

	res := map[string]structField{}
	for i := 0; i < entry.NumField(); i++ {
		mapStructField(entry.Field(i), func(field structField) {
			res[field.DbFieldName] = field
		})
	}
	return res, true
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

type numComparison uint8

const (
	numComparisonEqual numComparison = iota
	numComparisonGreater
	numComparisonGreaterOrEqual
	numComparisonLess
	numComparisonLessOrEqual
)

func compareNumbers(kind numComparison, a, b reflect.Value) bool {
	compareUInts := func(a, b uint64) bool {
		switch kind {
		case numComparisonEqual:
			return a == b
		case numComparisonGreater:
			return a > b
		case numComparisonGreaterOrEqual:
			return a >= b
		case numComparisonLess:
			return a < b
		case numComparisonLessOrEqual:
			return a <= b
		}
		return false
	}

	switch a.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch b.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			aInt := a.Int()
			bInt := b.Int()

			switch kind {
			case numComparisonEqual:
				return aInt == bInt
			case numComparisonGreater:
				return aInt > bInt
			case numComparisonGreaterOrEqual:
				return aInt >= bInt
			case numComparisonLess:
				return aInt < bInt
			case numComparisonLessOrEqual:
				return aInt <= bInt
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			aInt := a.Int()
			if aInt < 0 {
				if kind == numComparisonLess {
					return true
				}
				return false
			}
			return compareUInts(uint64(aInt), b.Uint())
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch b.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			bInt := b.Int()
			if bInt < 0 {
				if kind == numComparisonGreater {
					return true
				}
				return false
			}
			return compareUInts(a.Uint(), uint64(bInt))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return compareUInts(a.Uint(), b.Uint())
		}
	}

	return false
}

func assertMapHasStringKeys(m reflect.Type) {
	if m.Key().Kind() != reflect.String {
		panic("TODO support filter type map with non string key")
	}
}
