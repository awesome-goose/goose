package types

import "reflect"

func HasEmbed(target, embedded any) bool {
	targetType := reflect.TypeOf(target)
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	embeddedType := reflect.TypeOf(embedded)
	if embeddedType.Kind() == reflect.Ptr {
		embeddedType = embeddedType.Elem()
	}

	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
		if field.Anonymous && field.Type == embeddedType {
			return true
		}
	}

	return false
}
