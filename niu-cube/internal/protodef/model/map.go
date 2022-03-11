package model

import "fmt"

type FlattenMap map[string]interface{}

func (f FlattenMap) Filter(fields ...string) FlattenMap {
	var newMap = make(map[string]interface{})
	for _, field := range fields {
		if val, ok := f[field]; ok {
			newMap[field] = val
		}
	}
	return newMap
}

func (f FlattenMap) Merge(a map[string]interface{}) FlattenMap {
	for k, v := range a {
		f[k] = v
	}
	return f
}

func (f FlattenMap) Exclude(fields ...string) FlattenMap {
	hash := make(map[string]interface{})
	for _, v := range fields {
		hash[v] = 1
	}
	for k := range f {
		if _, ok := hash[k]; ok {
			delete(f, k)
		}
	}
	return f
}

// MakeFlattenMap parse kv pair from kvs,panic if not in pair
func MakeFlattenMap(kvs ...interface{}) FlattenMap {
	res := make(map[string]interface{})
	if len(kvs)%2 == 1 {
		panic("key/val should in pair")
	}
	for index, val := range kvs {
		// skip value
		if index%2 == 1 {
			continue
		}
		// res[key]=value
		key, ok := val.(string)
		if !ok {
			panic(fmt.Sprintf("error cover %v to string", key))
		}
		res[key] = kvs[index+1]
	}
	return res
}
