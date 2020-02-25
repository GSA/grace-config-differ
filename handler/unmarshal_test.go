package main

import (
	"reflect"
	"testing"
)

const (
	test1      = "test1"
	test2      = "test2"
	test3      = "test3"
	testing5   = "testing5"
	testString = "testString"
)

func TestUnmarshalSnapshot(t *testing.T) {
	tt := map[string]struct {
		json        []byte
		expected    snapshot
		expectedErr string
	}{
		"empty":   {[]byte("{}"), snapshot{}, ""},
		"invalid": {[]byte{}, snapshot{}, "unexpected end of JSON input"},
	}
	for name, tc := range tt {
		tc := tc

		t.Run(name, func(t *testing.T) {
			snapshots, err := unmarshalSnapshot(tc.json)
			if tc.expectedErr == "" && err != nil {
				t.Errorf("unmarshalSnapshot() failed. Unexpected error: %v", err)
			} else if err != nil && tc.expectedErr != err.Error() {
				t.Errorf("unmarshalSnapshot() failed. Expected error: %s. Got: %v", tc.expectedErr, err)
			}
			if !reflect.DeepEqual(snapshots, tc.expected) {
				t.Errorf("unmarshalSnapshot() failed. Expected: %v\nGot: %v\n", tc.expected, snapshots)
			}
		})
	}
}

func TestRecursiveUnMarshalArrayString(t *testing.T) {
	var expect []interface{}

	str := `["test2", "testing2"]`

	expect = append(expect, test2, "testing2")

	myMap, err := recursiveUnmarshalArrayString(str)
	if err != nil {
		t.Errorf("did not expect error: %v", err)
	}

	if !reflect.DeepEqual(myMap, expect) {
		t.Errorf("Expected: %v\nGot: %v\n", expect, myMap)
	}
}

func TestRecursiveUnmarshalString(t *testing.T) {
	var eArray []interface{}

	eMap := make(map[string]interface{})
	eMap[test3] = "testing3"

	eArray = append(eArray, "test4", "testing4")
	eMap["test4"] = eArray
	expect := eMap
	cases := []struct {
		input    string
		expected interface{}
	}{
		{`{"test3": "testing3", "test4": ["test4", "testing4"]}`, expect},
		{`"{\"test3\": \"testing3\", \"test4\": [\"test4\", \"testing4\"]}"`, expect},
		{`%22%7B%5C%22test3%5C%22%3A%20%5C%22testing3%5C%22%2C%20%5C%22test4%5C%22%3A%20%5B%5C%22test4%5C%22%2C%20%5C%22testing4%5C%22%5D%7D%22`,
			expect},
		{"null", make(map[string]interface{})},
	}

	for _, tc := range cases {
		myMap, err := recursiveUnmarshalString(tc.input)
		if err != nil {
			t.Errorf("did not expect error: %v", err)
		}

		if !reflect.DeepEqual(myMap, tc.expected) {
			t.Errorf("Expected (%T): %v\nGot (%T): %v\n", tc.expected, tc.expected, myMap, myMap)
		}
	}
}

func TestRecursiveUnmarshalMap(t *testing.T) {
	input := make(map[string]interface{})
	input["string"] = testString
	input["stringMap"] = `{"map": {"test5": "testing5"}}`
	input["stringArray"] = `["test1", "test2", "test3"]`
	input["map"] = make(map[string]interface{})
	input["map"].(map[string]interface{})["test5"] = testing5
	input["interfaceArray"] = make([]interface{}, 3)
	input["interfaceArray"].([]interface{})[0] = test1
	input["interfaceArray"].([]interface{})[1] = test2
	input["interfaceArray"].([]interface{})[2] = test3
	expected := make(map[string]interface{})
	expected["string"] = testString
	expected["stringMap"] = make(map[string]interface{})
	expected["stringMap"].(map[string]interface{})["map"] = make(map[string]interface{})
	expected["stringMap"].(map[string]interface{})["map"].(map[string]interface{})["test5"] = testing5
	expected["stringArray"] = make([]interface{}, 3)
	expected["stringArray"].([]interface{})[0] = test1
	expected["stringArray"].([]interface{})[1] = test2
	expected["stringArray"].([]interface{})[2] = test3
	expected["map"] = expected["stringMap"].(map[string]interface{})["map"]
	expected["interfaceArray"] = expected["stringArray"].([]interface{})

	result, err := recursiveUnmarshalMap(input)
	if err != nil {
		t.Errorf("did not expect error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected (%T):\n%v\nGot (%T):\n%v\n", expected, expected, result, result)
		t.Errorf("Types string: %T/%T", expected["string"], result["string"])
		t.Errorf("Types stringMap: %T/%T", expected["stringMap"], result["stringMap"])
		t.Errorf("Types stringArray: %T/%T", expected["stringArray"], result["stringArray"])
	}
}

func TestRecursiveUnMarshalMapString(t *testing.T) {
	str := `{"test1": "testing1"}`
	expect := make(map[string]interface{})
	expect[test1] = "testing1"

	myMap, err := recursiveUnmarshalMapString(str)
	if err != nil {
		t.Errorf("did not expect error: %v", err)
	}

	if !reflect.DeepEqual(myMap, expect) {
		t.Errorf("Expected: %v\nGot: %v\n", expect, myMap)
	}
}
