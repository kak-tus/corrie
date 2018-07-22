// Copyright (c) 2018, Eugene Ponizovsky, <ponizovsky@gmail.com>. All rights
// reserved. Use of this source code is governed by a MIT License that can
// be found in the LICENSE file.

// Package merger performs recursive merge of maps or structures into new one.
// Non-zero values from the right side has higher precedence. Slices do not
// merging, because main use case of this package is merging configuration
// parameters, and in this case merging of slices is unacceptable. Slices from
// the right side has higher precedence.
package merger

import "reflect"

// Merge method performs recursive merge of two maps or structures into new one.
func Merge(left, right interface{}) interface{} {
	result := merge(
		reflect.ValueOf(left),
		reflect.ValueOf(right),
	)

	if !result.IsValid() {
		return nil
	}

	return result.Interface()
}

func merge(left, right reflect.Value) reflect.Value {
	left = stripValue(left)
	right = stripValue(right)
	leftKind := left.Kind()
	rightKind := right.Kind()

	if !left.IsValid() {
		return right
	}
	if !right.IsValid() {
		return left
	}

	if leftKind == reflect.Ptr &&
		rightKind == reflect.Ptr {

		left := left.Elem()
		leftKind := left.Kind()

		right := right.Elem()
		rightKind := right.Kind()

		if leftKind == reflect.Map &&
			rightKind == reflect.Map {

			return mergeMap(left, right).Addr()
		}

		if leftKind == reflect.Struct &&
			rightKind == reflect.Struct {

			return mergeStruct(left, right).Addr()
		}
	}

	if leftKind == reflect.Map &&
		rightKind == reflect.Map {

		return mergeMap(left, right)
	}

	if leftKind == reflect.Struct &&
		rightKind == reflect.Struct {

		return mergeStruct(left, right)
	}

	if isZero(right) {
		return left
	}

	return right
}

func mergeMap(left, right reflect.Value) reflect.Value {
	rightType := right.Type()
	result := reflect.MakeMap(rightType)

	for _, key := range left.MapKeys() {
		result.SetMapIndex(key, left.MapIndex(key))
	}

	for _, key := range right.MapKeys() {
		value := merge(result.MapIndex(key), right.MapIndex(key))
		result.SetMapIndex(key, value)
	}

	return result
}

func mergeStruct(left, right reflect.Value) reflect.Value {
	leftType := left.Type()
	rightType := right.Type()

	if leftType != rightType {
		return right
	}

	result := reflect.New(rightType).Elem()

	for i := 0; i < rightType.NumField(); i++ {
		leftField := left.Field(i)
		rightField := right.Field(i)
		resField := result.Field(i)

		if resField.Kind() == reflect.Interface &&
			isZero(leftField) && isZero(rightField) {

			continue
		}

		if resField.CanSet() {
			resField.Set(merge(leftField, rightField))
		}
	}

	return result
}

func stripValue(value reflect.Value) reflect.Value {
	valueKind := value.Kind()

	if valueKind == reflect.Interface {
		return value.Elem()
	}

	return value
}

func isZero(value reflect.Value) bool {
	zero := reflect.Zero(value.Type())
	return reflect.DeepEqual(zero.Interface(), value.Interface())
}
