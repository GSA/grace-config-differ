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
	var resourceTypes = [...]string{
		"AWS::AppStream::DirectoryConfig",
		"AWS::AppStream::Application",
		"AWS::AppFlow::Flow",
		"AWS::ApiGateway::Stage",
		"AWS::ApiGateway::RestApi",
		"AWS::ApiGatewayV2::Stage",
		"AWS::ApiGatewayV2::Api",
		"AWS::Athena::WorkGroup",
		"AWS::Athena::DataCatalog",
		"AWS::CloudFront::Distribution",
		"AWS::CloudFront::StreamingDistribution",
		"AWS::CloudWatch::Alarm",
		"AWS::CloudWatch::MetricStream",
		"AWS::RUM::AppMonitor",
		"AWS::Evidently::Project",
		"AWS::CodeGuruReviewer::RepositoryAssociation",
		"AWS::Connect::PhoneNumber",
		"AWS::CustomerProfiles::Domain",
		"AWS::Detective::Graph",
		"AWS::DynamoDB::Table",
		"AWS::EC2::Host",
		"AWS::EC2::EIP",
		"AWS::EC2::Instance",
		"AWS::EC2::NetworkInterface",
		"AWS::EC2::SecurityGroup",
		"AWS::EC2::NatGateway",
		"AWS::EC2::EgressOnlyInternetGateway",
		"AWS::EC2::EC2Fleet",
		"AWS::EC2::SpotFleet",
		"AWS::EC2::PrefixList",
		"AWS::EC2::FlowLog",
		"AWS::EC2::TransitGateway",
		"AWS::EC2::TransitGatewayAttachment",
		"AWS::EC2::TransitGatewayRouteTable",
		"AWS::EC2::VPCEndpoint",
		"AWS::EC2::VPCEndpointService",
		"AWS::EC2::VPCPeeringConnection",
		"AWS::EC2::RegisteredHAInstance",
		"AWS::EC2::SubnetRouteTableAssociation",
		"AWS::EC2::LaunchTemplate",
		"AWS::EC2::NetworkInsightsAccessScopeAnalysis",
		"AWS::EC2::TrafficMirrorTarget",
		"AWS::EC2::TrafficMirrorSession",
		"AWS::EC2::DHCPOptions",
		"AWS::EC2::IPAM",
		"AWS::EC2::NetworkInsightsPath",
		"AWS::EC2::TrafficMirrorFilter",
		"AWS::EC2::Volume",
		"AWS::ImageBuilder::ImagePipeline",
		"AWS::ImageBuilder::DistributionConfiguration",
		"AWS::ImageBuilder::InfrastructureConfiguration",
		"AWS::ECR::Repository",
		"AWS::ECR::RegistryPolicy",
		"AWS::ECR::PullThroughCacheRule",
		"AWS::ECR::PublicRepository",
		"AWS::ECS::Cluster",
		"AWS::ECS::TaskDefinition",
		"AWS::ECS::Service",
		"AWS::ECS::TaskSet",
		"AWS::EFS::FileSystem",
		"AWS::EFS::AccessPoint",
		"AWS::EKS::Cluster",
		"AWS::EKS::FargateProfile",
		"AWS::EKS::IdentityProviderConfig",
		"AWS::EKS::Addon",
		"AWS::EMR::SecurityConfiguration",
		"AWS::Events::EventBus",
		"AWS::Events::ApiDestination",
		"AWS::Events::Archive",
		"AWS::Events::Endpoint",
		"AWS::Events::Connection",
		"AWS::Events::Rule",
		"AWS::EC2::TrafficMirrorSession",
		"AWS::EventSchemas::RegistryPolicy",
		"AWS::EventSchemas::Discoverer",
		"AWS::EventSchemas::Schema",
		"AWS::Forecast::Dataset",
		"AWS::FraudDetector::Label",
		"AWS::FraudDetector::EntityType",
		"AWS::FraudDetector::Variable",
		"AWS::FraudDetector::Outcome",
		"AWS::GuardDuty::Detector",
		"AWS::GuardDuty::ThreatIntelSet",
		"AWS::GuardDuty::IPSet",
		"AWS::GuardDuty::Filter",
		"AWS::HealthLake::FHIRDatastore",
		"AWS::Cassandra::Keyspace",
		"AWS::IVS::Channel",
		"AWS::IVS::RecordingConfiguration",
		"AWS::IVS::PlaybackKeyPair",
		"AWS::Elasticsearch::Domain",
		"AWS::OpenSearch::Domain",
		"AWS::Elasticsearch::Domain",
		"AWS::Pinpoint::ApplicationSettings",
		"AWS::Pinpoint::Segment",
		"AWS::Pinpoint::App",
		"AWS::Pinpoint::Campaign",
		"AWS::Pinpoint::InAppTemplate",
		"AWS::QLDB::Ledger",
		"AWS::Kinesis::Stream",
		"AWS::Kinesis::StreamConsumer",
		"AWS::KinesisAnalyticsV2::Application",
		"AWS::KinesisFirehose::DeliveryStream",
		"AWS::KinesisVideo::SignalingChannel",
		"AWS::Lex::BotAlias",
		"AWS::Lex::Bot",
		"AWS::Lightsail::Disk",
		"AWS::Lightsail::Certificate",
		"AWS::Lightsail::Bucket",
		"AWS::Lightsail::StaticIp",
		"AWS::LookoutMetrics::Alert",
		"AWS::LookoutVision::Project",
		"AWS::AmazonMQ::Broker",
		"AWS::MSK::Cluster",
		"AWS::Redshift::Cluster",
		"AWS::Redshift::ClusterParameterGroup",
		"AWS::Redshift::ClusterSecurityGroup",
		"AWS::Redshift::ScheduledAction",
		"AWS::Redshift::ClusterSnapshot",
		"AWS::Redshift::ClusterSubnetGroup",
		"AWS::Redshift::EventSubscription",
		"AWS::RDS::DBInstance",
		"AWS::RDS::DBSecurityGroup",
		"AWS::RDS::DBSnapshot",
		"AWS::RDS::DBSubnetGroup",
		"AWS::RDS::EventSubscription",
		"AWS::RDS::DBCluster",
		"AWS::RDS::DBClusterSnapshot",
		"AWS::RDS::GlobalCluster",
		"AWS::Route53::HostedZone",
		"AWS::Route53::HealthCheck",
		"AWS::Route53Resolver::ResolverEndpoint",
		"AWS::Route53Resolver::ResolverRule",
		"AWS::Route53Resolver::ResolverRuleAssociation",
		"AWS::Route53Resolver::FirewallDomainList",
		"AWS::AWS::Route53Resolver::FirewallRuleGroupAssociation",
		"AWS::Route53RecoveryReadiness::Cell",
		"AWS::Route53RecoveryReadiness::ReadinessCheck",
		"AWS::Route53RecoveryReadiness::RecoveryGroup",
		"AWS::Route53RecoveryControl::Cluster",
		"AWS::Route53RecoveryControl::ControlPanel",
		"AWS::Route53RecoveryControl::RoutingControl",
		"AWS::Route53RecoveryControl::SafetyRule",
		"AWS::Route53RecoveryReadiness::ResourceSet",
		"AWS::SageMaker::CodeRepository",
		"AWS::SageMaker::Domain",
		"AWS::SageMaker::AppImageConfig",
		"AWS::SageMaker::Image",
		"AWS::SageMaker::Model",
		"AWS::SageMaker::NotebookInstance",
		"AWS::SageMaker::NotebookInstanceLifecycleConfig",
		"AWS::SageMaker::EndpointConfig",
		"AWS::SageMaker::Workteam",
		"AWS::SES::ConfigurationSet",
		"AWS::SES::ContactList",
		"AWS::SES::Template",
		"AWS::SES::ReceiptFilter",
		"AWS::SES::ReceiptRuleSet",
		"AWS::SNS::Topic",
		"AWS::SQS::Queue",
		"AWS::S3::Bucket",
		"AWS::S3::AccountPublicAccessBlock",
		"AWS::S3::MultiRegionAccessPoint",
		"AWS::S3::StorageLens",
		"AWS::EC2::CustomerGateway",
		"AWS::EC2::InternetGateway",
		"AWS::EC2::NetworkAcl",
		"AWS::EC2::RouteTable",
		"AWS::EC2::Subnet",
		"AWS::EC2::VPC",
		"AWS::EC2::VPNConnection",
		"AWS::EC2::VPNGateway",
		"AWS::NetworkManager::TransitGatewayRegistration",
		"AWS::NetworkManager::Site",
		"AWS::NetworkManager::Device",
		"AWS::NetworkManager::Link",
		"AWS::NetworkManager::GlobalNetwork",
		"AWS::WorkSpaces::ConnectionAlias",
		"AWS::WorkSpaces::Workspace",
		"AWS::Amplify::App",
		"AWS::AppConfig::Application",
		"AWS::AppConfig::Environment",
		"AWS::AppConfig::ConfigurationProfile",
		"AWS::AppConfig::DeploymentStrategy",
		"AWS::AppRunner::VpcConnector",
		"AWS::AppMesh::VirtualNode",
		"AWS::AppMesh::VirtualService",
		"AWS::AppSync::GraphQLApi",
		"AWS::AuditManager::Assessment",
		"AWS::AutoScaling::AutoScalingGroup",
		"AWS::AutoScaling::LaunchConfiguration",
		"AWS::AutoScaling::ScalingPolicy",
		"AWS::AutoScaling::ScheduledAction",
		"AWS::AutoScaling::WarmPool",
		"AWS::Backup::BackupPlan",
		"AWS::Backup::BackupSelection",
		"AWS::Backup::BackupVault",
		"AWS::Backup::RecoveryPoint",
		"AWS::Backup::ReportPlan",
		"AWS::Backup::BackupPlan",
		"AWS::Backup::BackupSelection",
		"AWS::Backup::BackupVault",
		"AWS::Backup::RecoveryPoint",
		"AWS::Batch::JobQueue",
		"AWS::Batch::ComputeEnvironment",
		"AWS::Budgets::BudgetsAction",
		"AWS::ACM::Certificate",
		"AWS::CloudFormation::Stack",
		"AWS::CloudTrail::Trail",
		"AWS::Cloud9::EnvironmentEC2",
		"AWS::ServiceDiscovery::Service",
		"AWS::ServiceDiscovery::PublicDnsNamespace",
		"AWS::ServiceDiscovery::HttpNamespace",
		"AWS::CodeArtifact::Repository",
		"AWS::CodeBuild::Project",
		"AWS::CodeDeploy::Application",
		"AWS::CodeDeploy::DeploymentConfig",
		"AWS::CodeDeploy::DeploymentGroup",
		"AWS::CodePipeline::Pipeline",
		"AWS::Config::ResourceCompliance",
		"AWS::Config::ConformancePackCompliance",
		"AWS::Config::ConfigurationRecorder",
		"AWS::Config::ResourceCompliance",
		"AWS::Config::ConfigurationRecorder",
		"AWS::Config::ConformancePackCompliance",
		"AWS::Config::ConfigurationRecorder",
		"AWS::DMS::EventSubscription",
		"AWS::DMS::ReplicationSubnetGroup",
		"AWS::DMS::ReplicationInstance",
		"AWS::DMS::ReplicationTask",
		"AWS::DMS::Certificate",
		"AWS::DataSync::LocationSMB",
		"AWS::DataSync::LocationFSxLustre",
		"AWS::DataSync::LocationFSxWindows",
		"AWS::DataSync::LocationS3",
		"AWS::DataSync::LocationEFS",
		"AWS::DataSync::LocationNFS",
		"AWS::DataSync::LocationHDFS",
		"AWS::DataSync::LocationObjectStorage",
		"AWS::DataSync::Task",
		"AWS::DeviceFarm::TestGridProject",
		"AWS::DeviceFarm::InstanceProfile",
		"AWS::DeviceFarm::Project",
		"AWS::ElasticBeanstalk::Application",
		"AWS::ElasticBeanstalk::ApplicationVersion",
		"AWS::ElasticBeanstalk::Environment",
		"AWS::FIS::ExperimentTemplate",
		"AWS::GlobalAccelerator::Listener",
		"AWS::GlobalAccelerator::EndpointGroup",
		"AWS::GlobalAccelerator::Accelerator",
		"AWS::Glue::Job",
		"AWS::Glue::Classifier",
		"AWS::Glue::MLTransform",
		"AWS::GroundStation::Config",
		"AWS::IAM::User",
		"AWS::IAM::SAMLProvider",
		"AWS::IAM::ServerCertificate",
		"AWS::IAM::Group",
		"AWS::IAM::Role",
		"AWS::IAM::Policy",
		"AWS::AccessAnalyzer::Analyzer",
		"AWS::IoT::Authorizer",
		"AWS::IoT::SecurityProfile",
		"AWS::IoT::RoleAlias",
		"AWS::IoT::Dimension",
		"AWS::IoT::Policy",
		"AWS::IoT::MitigationAction",
		"AWS::IoT::ScheduledAudit",
		"AWS::IoT::AccountAuditConfiguration",
		"AWS::IoTSiteWise::Gateway",
		"AWS::IoT::CustomMetric",
		"AWS::IoTWireless::ServiceProfile",
		"AWS::IoT::FleetMetric",
		"AWS::IoTAnalytics::Datastore",
		"AWS::IoTAnalytics::Dataset",
		"AWS::IoTAnalytics::Pipeline",
		"AWS::IoTAnalytics::Channel",
		"AWS::IoTEvents::Input",
		"AWS::IoTEvents::DetectorModel",
		"AWS::IoTEvents::AlarmModel",
		"AWS::IoTTwinMaker::Workspace",
		"AWS::IoTTwinMaker::Entity",
		"AWS::IoTTwinMaker::Scene",
		"AWS::IoTSiteWise::Dashboard",
		"AWS::IoTSiteWise::Project",
		"AWS::IoTSiteWise::Portal",
		"AWS::IoTSiteWise::AssetModel",
		"AWS::KMS::Key",
		"AWS::KMS::Alias",
		"AWS::Lambda::Function",
		"AWS::Lambda::Alias",
		"AWS::NetworkFirewall::Firewall",
		"AWS::NetworkFirewall::FirewallPolicy",
		"AWS::NetworkFirewall::RuleGroup",
		"AWS::NetworkFirewall::TLSInspectionConfiguration",
		"AWS:Panorama::Package",
		"AWS::ResilienceHub::ResiliencyPolicy",
		"AWS::RoboMaker::RobotApplicationVersion",
		"AWS::RoboMaker::RobotApplication",
		"AWS::RoboMaker::SimulationApplication",
		"AWS::Signer::SigningProfile",
		"AWS::SecretsManager::Secret",
		"AWS::ServiceCatalog::CloudFormationProduct",
		"AWS::ServiceCatalog::CloudFormationProvisionedProduct",
		"AWS::ServiceCatalog::Portfolio",
		"AWS::Shield::Protection",
		"AWS::ShieldRegional::Protection",
		"AWS::StepFunctions::Activity",
		"AWS::StepFunctions::StateMachine",
		"AWS::SSM::ManagedInstanceInventory",
		"AWS::SSM::PatchCompliance",
		"AWS::SSM::AssociationCompliance",
		"AWS::SSM::FileData",
		"AWS::Transfer::Agreement",
		"AWS::Transfer::Connector",
		"AWS::Transfer::Workflow",
		"AWS::WAF::RateBasedRule",
		"AWS::WAF::Rule",
		"AWS::WAF::WebACL",
		"AWS::WAF::RuleGroup",
		"AWS::WAFRegional::RateBasedRule",
		"AWS::WAFRegional::Rule",
		"AWS::WAFRegional::WebACL",
		"AWS::WAFRegional::RuleGroup",
		"AWS::WAFv2::WebACL",
		"AWS::WAFv2::RuleGroup",
		"AWS::WAFv2::ManagedRuleSet",
		"AWS::WAFv2::IPSet",
		"AWS::WAFv2::RegexPatternSet",
		"AWS::XRay::EncryptionConfig",
		"AWS::ElasticLoadBalancingV2::LoadBalancer",
		"AWS::ElasticLoadBalancingV2::Listener",
		"AWS::ElasticLoadBalancing::LoadBalancer",
		"AWS::ElasticLoadBalancingV2::LoadBalancer",
		"AWS::MediaPackage::PackagingGroup",
		"AWS::MediaPackage::PackagingConfiguration",
	}
	// nolint: prealloc
	var res []*configservice.ResourceIdentifier

	for _, t := range &resourceTypes {
		t := t
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
