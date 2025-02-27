package components

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/sahilm/fuzzy"
)

func CompositeContentReflect(item interface{}) string {
	v := reflect.ValueOf(item)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	var parts []string
	for i := 0; i < v.NumField(); i++ {
		parts = append(parts, fmt.Sprintf("%v", v.Field(i).Interface()))
	}
	return strings.Join(parts, " ")
}

func FuzzyFilter(query string, items []interface{}) []int {
	if query == "" {
		indices := make([]int, len(items))
		for i := range items {
			indices[i] = i
		}
		return indices
	}
	composites := make([]string, len(items))
	for i, item := range items {
		composites[i] = strings.ToLower(CompositeContentReflect(item))
	}
	matches := fuzzy.Find(strings.ToLower(query), composites)
	indices := make([]int, len(matches))
	for i, m := range matches {
		indices[i] = m.Index
	}
	return indices
}
