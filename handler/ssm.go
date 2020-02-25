package main

import (
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/ssm"
)

func updateLastExecution(t time.Time, c *config, s client.ConfigProvider) {
	svc := ssm.New(s)
	input := ssm.PutParameterInput{
		Description: aws.String("Config Diff LastSuccessfulEvaluationTime"),
		KeyId:       aws.String(c.KmsKeyArn),
		Name:        aws.String(c.ParameterStore),
		Overwrite:   aws.Bool(true),
		Type:        aws.String("SecureString"),
		Value:       aws.String(t.Format(time.RFC3339)),
	}

	_, err := svc.PutParameter(&input)
	if err != nil {
		log.Printf("error updating last_execution: %v", err)
	}
}

func getPreviousExecution(c *config, s client.ConfigProvider) (t time.Time, err error) {
	svc := ssm.New(s)
	input := ssm.GetParameterInput{
		Name:           aws.String(c.ParameterStore),
		WithDecryption: aws.Bool(true),
	}

	res, err := svc.GetParameter(&input)
	if err != nil {
		return t, err
	}

	return time.Parse(time.RFC3339, aws.StringValue(res.Parameter.Value))
}
