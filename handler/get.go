package main

import (
	"bytes"
	"errors"
	"io"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/configservice"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/caarlos0/env"
)

const (
	maxRetries        = 20
	snapshotFrequency = 3 // frequency of ConfigSnapshots in hours
)

// sortItemSlices ... Sorts attributes of ConfigrationItems that are slices
func sortItemSlices(i []*configservice.ConfigurationItem) {
	for _, v := range i {
		e := v.RelatedEvents
		r := v.Relationships

		sort.SliceStable(e, func(i, j int) bool {
			return sliceSorter(e[i], e[j])
		})
		sort.SliceStable(r, func(i, j int) bool {
			return sliceSorter(r[i], r[j])
		})

		v.RelatedEvents = e
		v.Relationships = r
	}
}

// GetLastExecution ... gets the time of the most recent execution of all config services
func (c *CfgSvc) GetLastExecution() (time.Time, error) {
	t := time.Time{}

	stats, err := c.GetStatus()
	if err != nil {
		return t, err
	}

	if len(stats) == 0 {
		return t, errors.New("empty config rule evaluation status array")
	}

	for _, s := range stats {
		if t.Before(aws.TimeValue(s.LastSuccessfulEvaluationTime)) {
			t = aws.TimeValue(s.LastSuccessfulEvaluationTime)
		}
	}

	return t, nil
}

// GetItems ... gets AWS Config Service Configuration Items from resource history pages
func (c *CfgSvc) GetItems(lastExecution time.Time) (items []*configservice.ConfigurationItem, err error) {
	res, err := c.GetDiscoveredResources()
	if err != nil {
		log.Fatalf("Error getting discovered resources: %v\n", err)
		return nil, err
	}

	for _, r := range res {
		var results []*configservice.ConfigurationItem

		input := &configservice.GetResourceConfigHistoryInput{
			ResourceType: r.ResourceType,
			ResourceId:   r.ResourceId,
			EarlierTime:  aws.Time(lastExecution.Add(time.Minute * time.Duration(-window))),
			LaterTime:    aws.Time(lastExecution.Add(time.Minute * time.Duration(window))),
		}
		err := c.Client.GetResourceConfigHistoryPages(input,
			func(page *configservice.GetResourceConfigHistoryOutput, lastPage bool) bool {
				results = append(results, page.ConfigurationItems...)
				return !lastPage
			})

		if err != nil {
			log.Fatalf("error getting resource config history (Input: %v):\n%v\n", input, err)
			return nil, err
		}

		items = append(items, results...)
	}

	sortItemSlices(items)

	return items, nil
}

// GetStatus ... performs DescribeConfigRuleEvaluationStatus for all config rules
func (c *CfgSvc) GetStatus() ([]*configservice.ConfigRuleEvaluationStatus, error) {
	params := configservice.DescribeConfigRuleEvaluationStatusInput{}
	result, err := c.Client.DescribeConfigRuleEvaluationStatus(&params)

	if err != nil {
		return nil, err
	}

	status := result.ConfigRulesEvaluationStatus

	for aws.StringValue(result.NextToken) != "" {
		params.NextToken = result.NextToken
		result, err = c.Client.DescribeConfigRuleEvaluationStatus(&params)

		if err != nil {
			return nil, err
		}

		status = append(status, result.ConfigRulesEvaluationStatus...)
	}

	return status, nil
}

// GetDiscoveredResources ... loops through all specified resourceTypes
// Lists all resources by Type (Will need to loop over all cfg.ResourceType* types)
// https://docs.aws.amazon.com/config/latest/developerguide/resource-config-reference.html#supported-resources
func (c *CfgSvc) GetDiscoveredResources() ([]*configservice.ResourceIdentifier, error) {
	// List of resource types pulled from
	// github.com/aws/aws-sdk-go/models/apis/config/2014-11-12/api-2.json
	var resourceTypes = [...]string{"AWS::EC2::CustomerGateway", "AWS::EC2::EIP",
		"AWS::EC2::Host", "AWS::EC2::Instance", "AWS::EC2::InternetGateway",
		"AWS::EC2::NetworkAcl", "AWS::EC2::NetworkInterface", "AWS::EC2::RouteTable",
		"AWS::EC2::SecurityGroup", "AWS::EC2::Subnet", "AWS::CloudTrail::Trail",
		"AWS::EC2::Volume", "AWS::EC2::VPC", "AWS::EC2::VPNConnection",
		"AWS::EC2::VPNGateway", "AWS::IAM::Group", "AWS::IAM::Policy", "AWS::IAM::Role",
		"AWS::IAM::User", "AWS::ACM::Certificate", "AWS::RDS::DBInstance",
		"AWS::RDS::DBSubnetGroup", "AWS::RDS::DBSecurityGroup", "AWS::RDS::DBSnapshot",
		"AWS::RDS::EventSubscription", "AWS::ElasticLoadBalancingV2::LoadBalancer",
		"AWS::S3::Bucket", "AWS::SSM::ManagedInstanceInventory", "AWS::Redshift::Cluster",
		"AWS::Redshift::ClusterSnapshot", "AWS::Redshift::ClusterParameterGroup",
		"AWS::Redshift::ClusterSecurityGroup", "AWS::Redshift::ClusterSubnetGroup",
		"AWS::Redshift::EventSubscription", "AWS::CloudWatch::Alarm", "AWS::CloudFormation::Stack",
		"AWS::DynamoDB::Table", "AWS::AutoScaling::AutoScalingGroup", "AWS::AutoScaling::LaunchConfiguration",
		"AWS::AutoScaling::ScalingPolicy", "AWS::AutoScaling::ScheduledAction", "AWS::CodeBuild::Project",
		"AWS::WAF::RateBasedRule", "AWS::WAF::Rule", "AWS::WAF::WebACL", "AWS::WAFRegional::RateBasedRule",
		"AWS::WAFRegional::Rule", "AWS::WAFRegional::WebACL", "AWS::CloudFront::Distribution",
		"AWS::CloudFront::StreamingDistribution", "AWS::WAF::RuleGroup", "AWS::WAFRegional::RuleGroup",
		"AWS::Lambda::Function", "AWS::ElasticBeanstalk::Application",
		"AWS::ElasticBeanstalk::ApplicationVersion", "AWS::ElasticBeanstalk::Environment",
		"AWS::ElasticLoadBalancing::LoadBalancer", "AWS::XRay::EncryptionConfig",
		"AWS::SSM::AssociationCompliance", "AWS::SSM::PatchCompliance", "AWS::Shield::Protection",
		"AWS::ShieldRegional::Protection", "AWS::Config::ResourceCompliance", "AWS::CodePipeline::Pipeline",
	}
	// nolint: prealloc
	var res []*configservice.ResourceIdentifier

	for _, t := range &resourceTypes {
		input := &configservice.ListDiscoveredResourcesInput{
			ResourceType: aws.String(t),
		}

		result, err := c.Client.ListDiscoveredResources(input)
		if err != nil {
			log.Fatalf("Error ListDiscoveredResources (ResourceType: %s): %v\n", t, err)
			return nil, err
		}

		res = append(res, result.ResourceIdentifiers...)

		for aws.StringValue(result.NextToken) != "" {
			input.NextToken = result.NextToken

			result, err = c.Client.ListDiscoveredResources(input)
			if err != nil {
				log.Fatalf("Error ListDiscoveredResources (Input: %v): %v\n", input, err)
				return nil, err
			}

			res = append(res, result.ResourceIdentifiers...)
		}
	}

	return res, nil
}

// getSnapshotOfItem ... finds ConfigurationItem in Snaphot with matching ResourceId and ResourceType
func getSnapshotOfItem(item map[string]interface{}, snapshots []map[string]interface{}) map[string]interface{} {
	id := item["ResourceId"]
	resType := item["ResourceType"]

	for _, s := range snapshots {
		m := s
		if id == m["ResourceId"].(string) && resType == m["ResourceType"].(string) {
			return m
		}
	}

	return nil
}

// getPreviousSnapshot ... gets the name of the config snapshot bucket object
// created prior to the lastExecution time
// Assumes snapshots are taken every three hours - gets snapshot older than
// lastExecution time but less than three hours before lastExecution time
func getPreviousSnapshot(
	items []*configservice.ConfigurationItem,
	t time.Time,
	bucket, region string,
	svc s3iface.S3API) (*s3.Object, string, error) {
	// Get time from three hours before change...since snapshots are taken every
	// three hours, this will ensure we are looking in the correct folder by date
	prevTime := t.Add(time.Hour * time.Duration(-snapshotFrequency))
	year, month, day := prevTime.Date()
	account := aws.StringValue(items[0].AccountId)
	prefix := strings.Join([]string{
		"awsconfig",
		"AWSLogs",
		account,
		"Config",
		region,
		strconv.Itoa(year),
		strconv.Itoa(int(month)),
		strconv.Itoa(day),
		"ConfigSnapshot",
	}, "/")
	input := &s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}

	results, err := svc.ListObjects(input)
	if err != nil {
		return nil, "", err
	}

	for _, o := range results.Contents {
		m := aws.TimeValue(o.LastModified)
		if m.After(prevTime) && m.Before(t) {
			return getSnapshot(svc, bucket, o)
		}
	}

	return nil, "", errors.New("snapshot not found")
}

func getSnapshot(svc s3iface.S3API, bucket string, o *s3.Object) (*s3.Object, string, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    o.Key,
	}

	result, err := svc.GetObject(input)
	if err != nil {
		return o, "", err
	}

	defer result.Body.Close()

	b := bytes.Buffer{}
	if _, err := io.Copy(&b, result.Body); err != nil {
		return o, "", err
	}

	return o, b.String(), nil
}

func getSess() (config, *session.Session, error) {
	cfg := config{}

	err := env.Parse(&cfg)
	if err != nil {
		log.Fatalf("error parsing env config: %v", err)
		return cfg, nil, err
	}

	sess, err := session.NewSession(
		&aws.Config{
			Region:     aws.String(cfg.DefaultRegion),
			MaxRetries: aws.Int(maxRetries),
		})

	if err != nil {
		log.Fatalf("error creating new session: %v\n", err)
		return cfg, nil, err
	}

	return cfg, sess, nil
}
