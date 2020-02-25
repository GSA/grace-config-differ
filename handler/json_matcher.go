package main

import (
	"encoding/json"
	"log"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

// sliceSorter ... private function for sorting slices for order independent
//  json matching
func sliceSorter(x, y interface{}) bool {
	s1, err := json.Marshal(x)
	if err != nil {
		log.Printf("error marshaling to json: %v", err)
		return false
	}

	s2, err := json.Marshal(y)
	if err != nil {
		log.Printf("error marshaling to json: %v", err)
		return false
	}

	return string(s1) < string(s2)
}

// matchJSON ... compares and matches two json strings without regard for the
//  order of contained slices/arrays.  Returns true if equal false if different
func matchJSON(j1, j2 string) bool {
	var m1, m2 map[string]interface{}

	err := json.Unmarshal([]byte(j1), &m1)
	if err != nil {
		log.Printf("error unmarshaling json: %v", err)
		return false
	}

	err = json.Unmarshal([]byte(j2), &m2)
	if err != nil {
		log.Printf("error unmarshaling json: %v", err)
		return false
	}

	return cmp.Equal(m1, m2, cmpopts.SortSlices(sliceSorter))
}
