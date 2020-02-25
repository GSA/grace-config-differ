package main

import (
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

func TestParseItemsToHTMLCase1(t *testing.T) {
	// empty map
	myMap := make([]map[string]interface{}, 0)

	str, err := parseItemsToHTML(myMap)
	if err != nil {
		t.Errorf("did not expect error: %v", err)
	}

	if str != "" {
		t.Errorf("Expecting:\n%s\nGot:\n%s", "", str)
	}
}

func TestParseItemsToHTMLCase2(t *testing.T) {
	// simple map ... no diffs
	tMap := make(map[string]interface{})
	tMap["ResourceId"] = "testID1"
	tMap["ResourceName"] = "testName1"
	tMap["ResourceType"] = "testType1"
	myMap := make([]map[string]interface{}, 0)
	myMap = append(myMap, tMap)
	html := `<tr><td class="blank" colspan=4>&nbsp;</td></tr>
<tr><td class="resource" colspan=2>testName1</td><td class="resource" colspan=2>testType1 (New Item)</td></tr>
<tr><td>&nbsp</td><td colspan=3>{<br />
&nbsp;&nbsp;<strong>"ResourceId":</strong> "testID1",<br />
&nbsp;&nbsp;<strong>"ResourceName":</strong> "testName1",<br />
&nbsp;&nbsp;<strong>"ResourceType":</strong> "testType1"<br />
}</td></tr>
`

	str, err := parseItemsToHTML(myMap)
	if err != nil {
		t.Errorf("did not expect error: %v", err)
	}

	if str != html {
		t.Errorf("Expecting:\n%s\nGot:\n%s\n", html, str)
	}
	// simple map ... diff Name
	tMap["diffs"] = make(map[string]interface{})
	tMap["diffs"].(map[string]interface{})["ResourceName"] = "oldName1"
	myMap[0] = tMap
	html = `<tr><td class="blank" colspan=4>&nbsp;</td></tr>
<tr><td class="resource" colspan=2>testName1</td><td class="resource" colspan=2>testType1</td></tr>
<tr><td class="blank" colspan=4>&nbsp;</td></tr>
<tr><td class="blank">&nbsp;</td><th>Property</th><th>Previous</th><th>Current</th></tr>
<tr><td class="blank">&nbsp;</td><th>ResourceName</th><td>"oldName1"</td><td>"testName1"</td></tr>
`

	str, err = parseItemsToHTML(myMap)
	if err != nil {
		t.Errorf("did not expect error: %v", err)
	}

	if str != html {
		t.Errorf("Expecting:\n%s\nGot:\n%s\n", html, str)
	}

	// test items (complex test) ... no diffs
	items := parseTestItems(t, "testdata/test1_items.json")

	myMap, err = parseItemsToMap(items)
	if err != nil {
		t.Errorf("did not expect error: %v", err)
	}

	str, err = parseItemsToHTML(myMap)
	if err != nil {
		t.Errorf("did not expect error: %v", err)
	}

	tStr := unparsedTestItems(t, "testdata/test1.html")
	if str != tStr {
		t.Errorf("Expecting:\n%s\nGot:\n%s", tStr, str)
	}
}

// TestIssue26 ... SupplementaryConfiguration.unsupportedResources is not being
// unmarshalled into JSON:
//  	[map[resourceId:376979777136 resourceType:AWS::::Account]]
func TestIssue26(t *testing.T) {
	lastExecution := time.Date(2019, 10, 17, 22, 5, 0, 0, time.UTC)
	expected := unparsedTestItems(t, "testdata/issue26.html")
	items := parseTestItems(t, "testdata/issue26_items.json")
	f, _ := os.Open("testdata/issue26_snapshot.json")
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

	itemsMap, _, err := diffItems(items, lastExecution, m, &cfg)
	if err != nil {
		t.Errorf("diffItems experienced an unexpected error: %v", err)
	}

	str, err := parseItemsToHTML(itemsMap)
	if err != nil {
		t.Errorf("parseItemsToHTML experienced an unexpected error: %v", err)
	}

	if str != expected {
		t.Errorf("Issue26 failed: expected: \n%s\ngot: \n%s\n", expected, str)
	}
}
