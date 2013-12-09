package plist

import (
	"reflect"
	"sort"
)

type plistKind uint

const (
	Invalid plistKind = iota
	Dictionary
	Array
	String
	Integer
	Real
	Boolean
	Data
	Date
)

var plistKindNames map[plistKind]string = map[plistKind]string{
	Invalid:    "invalid",
	Dictionary: "dictionary",
	Array:      "array",
	String:     "string",
	Integer:    "integer",
	Real:       "real",
	Boolean:    "boolean",
	Data:       "data",
	Date:       "date",
}

type plistValue struct {
	kind  plistKind
	value interface{}
}

type sizedFloat struct {
	value float64
	bits  int
}

type dictionary struct {
	count  int
	m      map[string]*plistValue
	keys   sort.StringSlice
	values []*plistValue
}

func (d *dictionary) Len() int {
	return d.count
}

func (d *dictionary) Less(i, j int) bool {
	return d.keys.Less(i, j)
}

func (d *dictionary) Swap(i, j int) {
	d.keys.Swap(i, j)
	d.values[i], d.values[j] = d.values[j], d.values[i]
}

func (d *dictionary) populateArrays() {
	if d.count > 0 {
		return
	}

	l := len(d.m)
	d.count = l
	d.keys = make([]string, l)
	d.values = make([]*plistValue, l)
	i := 0
	for k, v := range d.m {
		d.keys[i] = k
		d.values[i] = v
		i++
	}
	sort.Sort(d)
}

type unknownTypeError struct {
	typ reflect.Type
}

func (u *unknownTypeError) Error() string {
	return "plist: can't marshal value of type " + u.typ.String()
}
