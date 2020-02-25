package main

import (
	"encoding/json"
	"log"
	"net/url"
	"regexp"
	"sort"
	"strconv"

	"github.com/aws/aws-sdk-go/service/configservice"
)

type snapshot struct {
	FileVersion        string                             `json:"fileVersion"`
	ConfigSnapshotID   string                             `json:"configSnapshotId"`
	ConfigurationItems []*configservice.ConfigurationItem `json:"configurationItems"`
}

func unmarshalSnapshot(s []byte) (result snapshot, err error) {
	n, err := normalizeSnapshotJSON(s)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(n, &result)

	return result, err
}

func normalizeSnapshotJSON(s []byte) ([]byte, error) {
	var m map[string]interface{}

	err := json.Unmarshal(s, &m)
	if err != nil {
		return nil, err
	}

	for k, v := range m {
		if k == "configurationItems" {
			m[k], err = normalizeConfigurationItems(v.([]interface{}))
			if err != nil {
				return nil, err
			}
		}
	}

	return json.Marshal(m)
}

func normalizeConfigurationItems(oldMap []interface{}) ([]interface{}, error) {
	for i, m := range oldMap {
		for k, v := range m.(map[string]interface{}) {
			switch k {
			case "configuration":
				// The JSON is a map, but ConfigurationItem expects a string
				s, err := json.Marshal(v)
				if err != nil {
					return oldMap, err
				}

				oldMap[i].(map[string]interface{})[k] = string(s)
			case "configurationStateId":
				// JSON is an number, but ConfigurationItem expects a string
				oldMap[i].(map[string]interface{})[k] = strconv.Itoa(int(v.(float64)))
			case "supplementaryConfiguration":
				s, err := normalizeSupplementaryConfiguration(v)
				if err != nil {
					return oldMap, err
				}

				oldMap[i].(map[string]interface{})[k] = s
			case "relationships":
				// Correct resourceName key (JSON uses name)
				s := normalizeRelationships(v.([]interface{}))
				oldMap[i].(map[string]interface{})[k] = s
			case "configurationStateMd5Hash":
				// Change key configurationStateMd5Hash to configurationItemMD5Hash
				delete(oldMap[i].(map[string]interface{}), "configurationStateMd5Hash")

				oldMap[i].(map[string]interface{})["configurationItemMD5Hash"] = v
			case "configurationItemVersion":
				// Change key configurationItemVersion to version
				delete(oldMap[i].(map[string]interface{}), "configurationItemVersion")

				oldMap[i].(map[string]interface{})["version"] = v
			case "awsAccountId":
				// Change key awsAccountId to accountId
				delete(oldMap[i].(map[string]interface{}), "awsAccountId")

				oldMap[i].(map[string]interface{})["accountId"] = v
			case "configurationItemCaptureTime":
				// Remove fractional part of seconds
				re := regexp.MustCompile(`\.\d*Z`)
				oldMap[i].(map[string]interface{})["configurationItemCaptureTime"] = re.ReplaceAllString(v.(string), "Z")
			}
		}
	}

	return oldMap, nil
}

func normalizeSupplementaryConfiguration(c interface{}) (interface{}, error) {
	for k, v := range c.(map[string]interface{}) {
		s, err := json.Marshal(v)
		if err != nil {
			return c, err
		}

		c.(map[string]interface{})[k] = string(s)
	}

	return c, nil
}

func normalizeRelationships(r []interface{}) []interface{} {
	for i, m := range r {
		if v, ok := m.(map[string]interface{})["name"]; ok {
			delete(m.(map[string]interface{}), "name")

			m.(map[string]interface{})["relationshipName"] = v
		}

		r[i] = m
	}

	return r
}

func recursiveUnmarshalMapString(str string) (map[string]interface{}, error) {
	myMap := make(map[string]interface{})

	err := json.Unmarshal([]byte(str), &myMap)
	if err != nil {
		log.Printf("error unmarshalling: %v", str)
		return nil, err
	}

	return recursiveUnmarshalMap(myMap)
}

func recursiveUnmarshalString(str string) (interface{}, error) {
	if str != "" {
		switch string(str[0]) {
		case "{":
			return recursiveUnmarshalMapString(str)
		case "[":
			return recursiveUnmarshalArrayString(str)
		case "\"":
			u, err := strconv.Unquote(str)
			if err != nil {
				return nil, err
			}

			return recursiveUnmarshalString(u)
		case "%":
			u, err := url.QueryUnescape(str)
			if err != nil {
				return nil, err
			}

			return recursiveUnmarshalString(u)
		default:
			if str == nullStr {
				return make(map[string]interface{}), nil
			}
		}
	}

	return str, nil
}

func recursiveUnmarshalMap(myMap map[string]interface{}) (map[string]interface{}, error) {
	var err error

	for key, value := range myMap {
		switch t := value.(type) {
		case string:
			myMap[key], err = recursiveUnmarshalString(t)
			if err != nil {
				return myMap, err
			}
		case map[string]interface{}:
			myMap[key], err = recursiveUnmarshalMap(t)
			if err != nil {
				return myMap, err
			}
		case []interface{}:
			for i, v := range t {
				t[i], err = recursiveUnmarshalInterface(v)
				if err != nil {
					return myMap, err
				}
			}

			myMap[key] = t
		}
	}

	return myMap, nil
}

func recursiveUnmarshalInterface(i interface{}) (interface{}, error) {
	switch t := i.(type) {
	case string:
		return recursiveUnmarshalString(t)
	case map[string]interface{}:
		return recursiveUnmarshalMap(t)
	case []interface{}:
		for i, v := range t {
			tmp, err := recursiveUnmarshalInterface(v)
			if err != nil {
				return t, err
			}

			t[i] = tmp
		}

		return t, nil
	}

	return i, nil
}

func recursiveUnmarshalArrayString(str string) ([]interface{}, error) {
	myArray := make([]interface{}, 0)

	err := json.Unmarshal([]byte(str), &myArray)
	if err != nil {
		log.Printf("error unmarshalling: %v", str)
		return nil, err
	}

	myArray = sortSlice(myArray)

	return myArray, nil
}

func sortSlice(s []interface{}) []interface{} {
	sort.SliceStable(s, func(i, j int) bool {
		return sliceSorter(s[i], s[j])
	})

	return s
}
