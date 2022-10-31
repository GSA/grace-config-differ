package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	// "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

const (
	testMapFile    = "testdata/test1_map.json"
	numStackFrames = 2
)

// helper functions //
func chkErr(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("unexpected error calling %s: %v", getCallerFunc(), err)
	}
}

func getCallerFunc() string {
	pc := make([]uintptr, 1)
	if runtime.Callers(numStackFrames, pc) == 0 {
		return "nil"
	}

	frame, _ := runtime.CallersFrames([]uintptr{pc[0]}).Next()

	return fmt.Sprintf("%s:%d", frame.File, frame.Line)
}

func unparsedTestItems(t *testing.T, f string) string {
	s, err := os.ReadFile(f)
	if err != nil {
		t.Fatalf("Error reading %s: %v", f, err)
	}

	return strings.TrimSuffix(string(s), "\n")
}

func parseTestItems(t *testing.T, f string) []*configservice.ConfigurationItem {
	var items []*configservice.ConfigurationItem

	jsonFile, err := os.Open(f)
	if err != nil {
		t.Fatal(err)
	}

	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		t.Fatal(err)
	}

	err = json.Unmarshal(byteValue, &items)
	if err != nil {
		t.Fatal(err)
	}

	return items
}

func parseTestMap(t *testing.T, f string) []map[string]interface{} {
	var myMap []map[string]interface{}

	jsonFile, err := os.Open(f)
	if err != nil {
		t.Fatal(err)
	}

	defer jsonFile.Close()

	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		t.Fatal(err)
	}

	err = json.Unmarshal(byteValue, &myMap)
	if err != nil {
		t.Fatal(err)
	}

	return myMap
}

// AWS Service Mocks //
type mockS3 struct {
	s3iface.S3API
	Resp    string
	Objects s3.ListObjectsOutput
	Object  s3.GetObjectOutput
}

func (m *mockS3) ListObjects(in *s3.ListObjectsInput) (*s3.ListObjectsOutput, error) {
	return &m.Objects, nil
}

func (m *mockS3) GetObject(in *s3.GetObjectInput) (*s3.GetObjectOutput, error) {
	return &m.Object, nil
}

// test functions //
func TestParseItemsToMap(t *testing.T) {
	items := parseTestItems(t, "testdata/test1_items.json")

	myMap, err := parseItemsToMap(items)
	if err != nil {
		t.Fatalf("did not expect error: %v", err)
	}

	testMap := parseTestMap(t, testMapFile)
	if !reflect.DeepEqual(myMap, testMap) {
		t.Fatalf("Expected (%T): %v\nGot (%T): %v\n", testMap, testMap, myMap, myMap)
	}
}

func TestMakeDiffsSimple(t *testing.T) {
	items := []*configservice.ConfigurationItem{{
		ResourceType: aws.String("AWS::S3::Bucket"),
		ResourceId:   aws.String("test"),
		Relationships: []*configservice.Relationship{{
			RelationshipName: aws.String("Is associated with "),
			ResourceId:       aws.String("grace-development-access-logs"),
			ResourceType:     aws.String("AWS::S3::Bucket"),
		}},
	}}

	myMap, err := parseItemsToMap(items)
	if err != nil {
		t.Errorf("did not expect error: %v", err)
	}

	snapshots, err := unmarshalSnapshot([]byte(`{
		"configurationItems": [{
			"resourceType": "AWS::S3::Bucket",
			"resourceId":   "test",
			"supplementaryConfiguration": {},
			"relationships": [{
				"resourceId": "grace-development-access-logs",
				"resourceType": "AWS::S3::Bucket",
				"name": "Is associated with "
			}]
		}]
	}`))
	if err != nil {
		t.Errorf("did not expect error: %v", err)
	}

	snapshotMap, err := parseItemsToMap(snapshots.ConfigurationItems)
	if err != nil {
		t.Errorf("did not expect error: %v", err)
	}

	for index, i := range myMap {
		snapshot := getSnapshotOfItem(i, snapshotMap)
		if snapshot != nil {
			myMap[index]["diffs"] = makeDiffs(removeNulls(snapshot), removeNulls(i))
		}
	}

	str, err := parseItemsToHTML(myMap)
	if err != nil {
		t.Errorf("did not expect error: %v", err)
	}

	tStr := `<tr><td class="blank" colspan=4>&nbsp;</td></tr>
<tr><td class="resource" colspan=2>test</td><td class="resource" colspan=2>AWS::S3::Bucket</td></tr>
<tr><td class="blank" colspan=4>&nbsp;</td></tr>
<tr><td class="blank">&nbsp;</td><th>Property</th><th>Previous</th><th>Current</th></tr>
`
	if str != tStr {
		t.Errorf("Expected:\n%s\n", tStr)
		t.Errorf("Got:\n%s\n", str)
	}
}

func TestMakeDiffsComplex(t *testing.T) {
	// test items (complex test) ... diffs
	items := parseTestItems(t, "testdata/test2_items.json")

	myMap, err := parseItemsToMap(items)
	if err != nil {
		t.Errorf("did not expect error: %v", err)
	}

	ssString := unparsedTestItems(t, "testdata/test2_snapshot.json")

	snapshots, err := unmarshalSnapshot([]byte(ssString))
	if err != nil {
		t.Errorf("did not expect error: %v", err)
	}

	snapshotMap, err := parseItemsToMap(snapshots.ConfigurationItems)
	if err != nil {
		t.Errorf("did not expect error: %v", err)
	}

	for index, i := range myMap {
		snapshot := getSnapshotOfItem(i, snapshotMap)
		if snapshot != nil {
			myMap[index]["diffs"] = makeDiffs(removeNulls(snapshot), removeNulls(i))
		}
	}

	str, err := parseItemsToHTML(myMap)
	if err != nil {
		t.Errorf("did not expect error: %v", err)
	}

	tStr := unparsedTestItems(t, "testdata/test2.html")
	if str != tStr {
		t.Errorf("Expected:\n%s\n", tStr)
		t.Errorf("Got:\n%s\n", str)
	}
}

func TestDiffItems(t *testing.T) {
	items := []*configservice.ConfigurationItem{{
		ResourceType:  aws.String("AWS::S3::Bucket"),
		ResourceId:    aws.String("test"),
		AccountId:     aws.String("test"),
		Relationships: []*configservice.Relationship{},
	}}
	lastExecution := time.Date(2019, 6, 24, 15, 29, 0, 0, time.UTC)
	f, _ := os.Open("testdata/test3_snapshot.json")
	m := &mockS3{
		Objects: s3.ListObjectsOutput{
			Contents: []*s3.Object{
				{LastModified: aws.Time(lastExecution.Add(time.Minute * time.Duration(-1)))},
			},
		},
		Object: s3.GetObjectOutput{
			Body: f,
		},
	}
	cfg := config{
		S3Bucket:      "test",
		DefaultRegion: "test",
	}

	itemsMap, ssObject, err := diffItems(items, lastExecution, m, &cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if aws.TimeValue(ssObject.LastModified) != lastExecution.Add(time.Minute*time.Duration(-1)) {
		t.Errorf("expected: %v\ngot: %v\n", lastExecution.Add(time.Minute*time.Duration(-1)), ssObject.LastModified)
		t.Errorf("expected: %T\ngot: %T\n", lastExecution.Add(time.Minute*time.Duration(-1)), aws.TimeValue(ssObject.LastModified))
	}

	if len(itemsMap) != 0 {
		t.Errorf("expected no items.  Got %d\n", len(itemsMap))
	}
}

func TestDiffItemsPolicyOrder(t *testing.T) {
	items := parseTestItems(t, "testdata/test4_items.json")
	lastExecution := time.Date(2019, 10, 17, 22, 5, 0, 0, time.UTC)
	f, _ := os.Open("testdata/test4_snapshot.json")
	m := &mockS3{
		Objects: s3.ListObjectsOutput{
			Contents: []*s3.Object{
				{LastModified: aws.Time(lastExecution.Add(time.Minute * time.Duration(-1)))},
			},
		},
		Object: s3.GetObjectOutput{
			Body: f,
		},
	}
	cfg := config{
		S3Bucket:      "test",
		DefaultRegion: "test",
	}

	itemsMap, ssObject, err := diffItems(items, lastExecution, m, &cfg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if aws.TimeValue(ssObject.LastModified) != lastExecution.Add(time.Minute*time.Duration(-1)) {
		t.Errorf("expected: %v\ngot: %v\n", lastExecution.Add(time.Minute*time.Duration(-1)), ssObject.LastModified)
		t.Errorf("expected: %T\ngot: %T\n", lastExecution.Add(time.Minute*time.Duration(-1)), aws.TimeValue(ssObject.LastModified))
	}

	if len(itemsMap) != 0 {
		t.Errorf("expected no items.  Got %d\n", len(itemsMap))

		b, err := json.MarshalIndent(itemsMap[0]["diffs"], "", "  ")
		chkErr(t, err)
		t.Logf("Diffs:\n%s", string(b))

		a, err := json.MarshalIndent(itemsMap[0]["SupplementaryConfiguration"], "", "  ")
		chkErr(t, err)
		t.Logf("SupplementaryConfiguration:\n%s", string(a))
	}
}

func TestDiffsExist(t *testing.T) {
	tt := map[string]struct {
		i        interface{}
		expected bool
	}{
		"no_diffs": {
			i: []map[string]interface{}{
				{"a": "test"},
				{"b": "test"},
			},
			expected: false,
		},
		"has_diffs": {
			i: []map[string]interface{}{
				{"a": "test"},
				{"diffs": "test"},
			},
			expected: true,
		},
		"nil": {
			i:        nil,
			expected: false,
		},
		"slice_of_interface": {
			i: []interface{}{
				[]map[string]interface{}{
					{"a": "test"},
					{"b": "test"},
				},
				[]map[string]interface{}{
					{"a": "test"},
					{"diffs": "test"},
				},
			},
			expected: true,
		},
	}
	for name, tc := range tt {
		tc := tc

		t.Run(name, func(t *testing.T) {
			actual := diffsExist(tc.i)
			if actual != tc.expected {
				t.Errorf("diffsExist()failed. Expecting %t, Got %t", tc.expected, actual)
			}
		})
	}
}
