package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/ses/sesiface"
)

// AWS Service Mocks //
type mockSESClient struct {
	sesiface.SESAPI
}

func (m *mockSESClient) SendEmail(in *ses.SendEmailInput) (*ses.SendEmailOutput, error) {
	return nil, nil
}

func (m *mockSESClient) SendRawEmail(in *ses.SendRawEmailInput) (*ses.SendRawEmailOutput, error) {
	return nil, nil
}

// test functions //
func TestSendEmail(t *testing.T) {
	tt := map[string]struct {
		mapFile       string
		lastExecution time.Time
		expected      string
		expectedLog   string
	}{
		"test1": {
			mapFile:       "testdata/test1_map.json",
			lastExecution: time.Date(2020, 1, 30, 13, 35, 19, 0, time.UTC),
			expected:      unparsedTestItems(t, "testdata/test1_body.html"),
			expectedLog:   "Email sent to address:",
		},
	}
	mockSvc := &mockSESClient{}
	cfg := config{}

	for name, tc := range tt {
		tc := tc

		t.Run(name, func(t *testing.T) {
			year, month, day := tc.lastExecution.Date()
			key := fmt.Sprintf(
				"123456789012_Config_us-east-1_ConfigSnapshot_%04d%02d%02dT133519Z_2e72344a-338f-4768-b01f-98cd83211635.json.gz",
				year, int(month), day)
			itemsMap := parseTestMap(t, tc.mapFile)
			ssObject := &s3.Object{
				Key: aws.String(key),
			}

			var buf bytes.Buffer
			log.SetOutput(&buf)
			defer func() {
				log.SetOutput(os.Stderr)
			}()

			htmlBody, err := sendEmail(itemsMap, tc.lastExecution, ssObject, mockSvc, &cfg)
			if err != nil {
				t.Errorf("sendEmail() failed. Unexpected error: %v\n", err)
			}
			if htmlBody != tc.expected {
				t.Errorf("sendEmail() failed. Expecting: \n%s\nGot:\n%s\n", tc.expected, htmlBody)
			}
			if tc.expectedLog == "" && buf.String() != "" {
				t.Errorf("sendEmail() failed. Expecting empty log. Got: %s\n", buf.String())
			} else if !strings.Contains(buf.String(), tc.expectedLog) {
				t.Errorf("sendEmail() failed. Expecting log to contain: %s\nGot: %s\n", tc.expectedLog, buf.String())
			}
		})
	}
}
