package main

import (
	"encoding/json"
	"io"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/aws/aws-sdk-go/service/configservice/configserviceiface"
)

const (
	numResourceTypes = 336
)

// helper functions //
func parseStatusOutput(t *testing.T) configservice.DescribeConfigRuleEvaluationStatusOutput {
	var status configservice.DescribeConfigRuleEvaluationStatusOutput

	jsonFile, err := os.Open("testdata/status_output.json")
	if err != nil {
		t.Fatal(err)
	}

	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	err = json.Unmarshal(byteValue, &status)
	if err != nil {
		t.Fatal(err)
	}

	return status
}

// AWS Service Mocks //
type mockCfgSvcClient struct {
	configserviceiface.ConfigServiceAPI
	StatusResp    configservice.DescribeConfigRuleEvaluationStatusOutput
	ResourcesResp configservice.ListDiscoveredResourcesOutput
	HistoryResp   configservice.GetResourceConfigHistoryOutput
}

func (m *mockCfgSvcClient) DescribeConfigRuleEvaluationStatus(
	in *configservice.DescribeConfigRuleEvaluationStatusInput) (*configservice.DescribeConfigRuleEvaluationStatusOutput, error) {
	return &m.StatusResp, nil
}

func (m *mockCfgSvcClient) ListDiscoveredResources(
	in *configservice.ListDiscoveredResourcesInput) (*configservice.ListDiscoveredResourcesOutput, error) {
	return &m.ResourcesResp, nil
}

func (m *mockCfgSvcClient) GetResourceConfigHistoryPages(
	in *configservice.GetResourceConfigHistoryInput, fn func(*configservice.GetResourceConfigHistoryOutput, bool) bool) error {
	fn(&m.HistoryResp, true)
	return nil
}

// test functions //
func TestGetLastExecution(t *testing.T) {
	c := CfgSvc{
		Client: &mockCfgSvcClient{StatusResp: parseStatusOutput(t)},
	}

	a, err := c.GetLastExecution()
	if err != nil {
		t.Errorf("did not expect error: %v", err)
	}

	e := time.Date(2019, time.October, 7, 16, 35, 0, 0, time.UTC)
	if a != e {
		t.Errorf("Exptected %v\nGot %v", e, a)
	}
}

func TestGetDiscoveredResources(t *testing.T) {
	// Empty case
	resp := configservice.ListDiscoveredResourcesOutput{}
	c := CfgSvc{
		Client: &mockCfgSvcClient{ResourcesResp: resp},
	}

	a, err := c.GetDiscoveredResources()
	if err != nil {
		t.Errorf("did not expect error: %v", err)
	}

	if len(a) != 0 {
		t.Errorf("Expected empty result.  Got:%v", a)
	}
	// Case of a resource for all types (336)
	resp = configservice.ListDiscoveredResourcesOutput{
		NextToken: nil,
		ResourceIdentifiers: []*configservice.ResourceIdentifier{
			{
				ResourceDeletionTime: nil,
				ResourceId:           aws.String("test"),
				ResourceName:         aws.String("test"),
				ResourceType:         aws.String("test"),
			},
		},
	}

	c = CfgSvc{
		Client: &mockCfgSvcClient{ResourcesResp: resp},
	}

	a, err = c.GetDiscoveredResources()
	if err != nil {
		t.Errorf("did not expect error: %v", err)
	}

	if len(a) != numResourceTypes {
		t.Errorf("Expected %d resources.  Got %d\n%v\n", numResourceTypes, len(a), a)
	}
}

func TestGetItems(t *testing.T) {
	c := CfgSvc{
		Client: &mockCfgSvcClient{
			ResourcesResp: configservice.ListDiscoveredResourcesOutput{
				NextToken: nil,
				ResourceIdentifiers: []*configservice.ResourceIdentifier{
					{
						ResourceDeletionTime: nil,
						ResourceId:           aws.String("test"),
						ResourceName:         aws.String("test"),
						ResourceType:         aws.String("test"),
					},
				},
			},
			HistoryResp: configservice.GetResourceConfigHistoryOutput{
				ConfigurationItems: []*configservice.ConfigurationItem{
					{AccountId: aws.String("0123456789012"),
						Configuration: aws.String("test")},
				},
			},
		},
	}

	a, err := c.GetItems(time.Date(2019, 6, 24, 15, 29, 0, 0, time.UTC))
	if err != nil {
		t.Errorf("did not expect error: %v\n", err)
	}

	if len(a) != numResourceTypes {
		t.Fatalf("Expected %d. Got %d\n", numResourceTypes, len(a))
	}

	if aws.StringValue(a[0].AccountId) != "0123456789012" {
		t.Errorf("Expected first account ID == 0123456789012.  Got %v", aws.StringValue(a[0].AccountId))
	}
}

/*
	// Place holder for creating additional mocks
	func TestGetItems(t *testing.T) {
		sess := session.Must(session.NewSession())
		c := CfgSvc{
			Client: configservice.New(sess, &aws.Config{Region: aws.String("us-east-1")}),
		}
		r, err := c.GetItems(time.Date(2019, 6, 24, 15, 29, 0, 0, time.UTC))
		if err != nil {
			t.Errorf("did not expect error: %v", err)
		}
		b, _ := json.MarshalIndent(r, "", "  ")
		fmt.Print(string(b))
	// fmt.Printf("DiscoveredResources:\n%v\n", res)
	// lastExecution = time.Date(2019, 6, 24, 15, 29, 0, 0, time.UTC)
	// lastExecution = time.Date(2019, 6, 27, 19, 44, 0, 0, time.UTC)
	// lastExecution = time.Date(2019, 7, 2, 5, 18, 0, 0, time.UTC)
	// lastExecution = time.Date(2019, 7, 9, 13, 58, 20, 0, time.UTC)
	// lastExecution = time.Date(2019, 7, 15, 13, 30, 0, 0, time.UTC)
	// lastExecution = time.Date(2019, 7, 16, 11, 18, 0, 0, time.UTC)
	// lastExecution = time.Date(2019, 9, 4, 10, 5, 0, 0, time.UTC)
}
*/
