package main

import (
	"encoding/json"
	"log"
	"reflect"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/aws/aws-sdk-go/service/configservice/configserviceiface"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/ses"
)

const (
	nullStr = "null"
	window  = 5 // +/- minutes for earlier/later time
)

// config ... struct for holding environment variables
type config struct {
	DefaultRegion  string   `env:"AWS_DEFAULT_REGION" envDefault:"us-east-1"`
	Sender         string   `env:"sender,required"`
	Recipients     []string `env:"recipients,required" envSeparator:","`
	CharSet        string   `env:"char_set" envDefault:"UTF-8"`
	S3Bucket       string   `env:"s3_bucket,required"`
	ParameterStore string   `env:"ssm_parameter_store,required"`
	KmsKeyArn      string   `env:"kms_key_arn,required"`
}

// CfgSvc ... provides interface to AWS Config Service
type CfgSvc struct {
	Client configserviceiface.ConfigServiceAPI
}

// parseItemsToMap ... converts slice of items to a slice of maps recursively
// parsing any JSON values
func parseItemsToMap(items []*configservice.ConfigurationItem) ([]map[string]interface{}, error) {
	sortItemSlices(items)

	a := make([]map[string]interface{}, 0)

	for _, i := range items {
		slice, err := json.Marshal(i)
		if err != nil {
			return nil, err
		}

		myMap, err := recursiveUnmarshalMapString(string(slice))
		if err != nil {
			return nil, err
		}

		a = append(a, myMap)
	}

	return a, nil
}

func removeNulls(m map[string]interface{}) map[string]interface{} {
	for k, i := range m {
		if i == nil {
			delete(m, k)
		} else if k != "AccessControlList" {
			switch v := i.(type) {
			case map[string]interface{}:
				m[k] = removeNulls(v)
			case []interface{}:
				m[k] = removeNullsArray(v)
			default: // Do nothing
			}
		}
	}

	return m
}

func removeNullsArray(a []interface{}) []interface{} {
	for j, i := range a {
		switch v := i.(type) {
		case map[string]interface{}:
			a[j] = removeNulls(v)
		case []interface{}:
			a[j] = removeNullsArray(v)
		default: // Do nothing
		}
	}

	return a
}

// diffItems ... compares changed ConfigurationItems to lastest Snapshot
func diffItems(
	items []*configservice.ConfigurationItem,
	t time.Time,
	svc s3iface.S3API,
	cfg *config) ([]map[string]interface{}, *s3.Object, error) {
	ssObject, ssString, err := getPreviousSnapshot(items, t, cfg.S3Bucket, cfg.DefaultRegion, svc)
	if err != nil {
		log.Fatalf("error getting previous snapshot: %v\n", err)
		return nil, nil, err
	}

	snapshots, err := unmarshalSnapshot([]byte(ssString))
	if err != nil {
		return nil, nil, err
	}

	snapshotMap, err := parseItemsToMap(snapshots.ConfigurationItems)
	if err != nil {
		return nil, nil, err
	}

	itemsMap, err := parseItemsToMap(items)
	if err != nil {
		return nil, nil, err
	}

	var diffs []map[string]interface{}

	for _, v := range itemsMap {
		snapshot := getSnapshotOfItem(v, snapshotMap)
		if snapshot != nil {
			v["diffs"] = makeDiffs(removeNulls(snapshot), removeNulls(v))
			if len(v["diffs"].(map[string]interface{})) != 0 {
				diffs = append(diffs, v)
			}
		}
	}

	return diffs, ssObject, nil
}

func makeDiffs(old, newer map[string]interface{}) map[string]interface{} {
	diffs := make(map[string]interface{})

	for key, value := range newer {
		if !reflect.DeepEqual(old[key], value) {
			if key == "Configuration" || key == "SupplementaryConfiguration" {
				jold, _ := json.Marshal(old[key])
				jnew, _ := json.Marshal(value)

				if !matchJSON(string(jold), string(jnew)) {
					diffs[key] = make(map[string]interface{})
					diffs[key].(map[string]interface{})["diffs"] = makeDiffs(
						old[key].(map[string]interface{}),
						removeNulls(value.(map[string]interface{})))
				}
			} else {
				diffs[key] = old[key]
			}
		}
	}

	return diffs
}

// diffsExist ... Returns false if there we no configuration changes
func diffsExist(i interface{}) (ret bool) {
	if i != nil {
		switch v := i.(type) {
		case map[string]interface{}:
			if _, ok := v["diffs"]; ok {
				return true
			}

			for _, w := range v {
				if !ret {
					ret = diffsExist(w)
				}
			}
		case []map[string]interface{}:
			for _, w := range v {
				if !ret {
					ret = diffsExist(w)
				}
			}
		case []interface{}:
			for _, w := range v {
				if !ret {
					ret = diffsExist(w)
				}
			}
		}
	}

	return ret
}

func alreadyChecked(t time.Time, cfg *config, sess client.ConfigProvider) bool {
	previous, err := getPreviousExecution(cfg, sess)
	if err != nil {
		updateLastExecution(t, cfg, sess)
		return false
	}

	if previous == t {
		return true
	}

	updateLastExecution(t, cfg, sess)

	return false
}

func configItemChangeReport() {
	cfg, sess, err := getSess()
	if err != nil {
		return
	}

	c := CfgSvc{
		Client: configservice.New(sess),
	}

	lastExecution, err := c.GetLastExecution()
	if err != nil {
		log.Fatalf("error getting last execution time: %v\n", err)
		return
	}

	if alreadyChecked(lastExecution, &cfg, sess) {
		log.Printf("Already checked this config service history execution: %v", lastExecution)
		return
	}

	items, err := c.GetItems(lastExecution)
	if err != nil {
		return
	}

	if len(items) > 0 {
		itemsMap, ssObject, err := diffItems(items, lastExecution, s3.New(sess), &cfg)
		if err != nil {
			log.Fatalf("error getting diff of items: %v", err)
			return
		}

		if diffsExist(itemsMap) {
			_, err = sendEmail(itemsMap, lastExecution, ssObject, ses.New(sess), &cfg)
			if err != nil {
				log.Fatalf("error sending email: %v\n", err)
				return
			}
		} else {
			log.Printf("no configuration changes since last snapshot")
			return
		}
	} else {
		log.Printf("no configuration changes during time frame (%v +/- %v min)\n", lastExecution, window)
		return
	}
}

func main() {
	lambda.Start(configItemChangeReport)
}
