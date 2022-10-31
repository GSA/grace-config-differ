package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"encoding/json"
	"log"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/ses/sesiface"
	"gopkg.in/gomail.v2"
)

const (
	style = `<head>
<style>
	table {border-collapse: collapse;}
	td, th {border: 1px solid Black;}
	th {background: LightGray;}
	tr:nth-child(even) {background: #F3F3F3;}
	tr:nth-child(odd) {background: White;}
	.resource {background-color: RoyalBlue; color: White; font-weight: bold;}
	.blank {background-color: White; border: none;}
	.group {background-color: LightBlue;}
</style>
</head>
`
	jsonFile = "/tmp/items.json"
)

// sendEmail ... sends an email to recipients specified in environment variable
func sendEmail(
	itemsMap []map[string]interface{},
	t time.Time,
	ssObject *s3.Object,
	svc sesiface.SESAPI,
	cfg *config) (htmlBody string, err error) {
	html, err := parseItemsToHTML(itemsMap)
	if err != nil {
		log.Fatalf("error parsing configuration items: %v", err)
	}

	htmlBody = fmt.Sprintf("%s<h1>Configuration Changes at %v (+/- %v min)</h1>\n",
		style, t, window)
	htmlBody += fmt.Sprintf("<table>\n<tr><td class=\"resource\">Snapshot</td><td colspan=3>%s</td></tr>\n%s</table>",
		filepath.Base(aws.StringValue(ssObject.Key)), html)

	slice, err := json.MarshalIndent(itemsMap, "", "  ")
	if err != nil {
		log.Fatalf("error marshaling itemsMap: %v", err)
		return htmlBody, err
	}

	err = os.WriteFile(jsonFile, slice, 0600)
	if err != nil {
		log.Fatalf("error writing items to file: %v\n", err)
		return htmlBody, err
	}

	subject := fmt.Sprintf("Changed/Discovered Configuration Items (%v)", t)

	input, err := buildEmailInput(subject, htmlBody, jsonFile, cfg)
	if err != nil {
		log.Fatalf("error building raw email input: %v", err)
		return htmlBody, err
	}

	result, err := svc.SendRawEmail(input)
	if err != nil {
		log.Fatalf("error sending email: %v", err)
		return htmlBody, err
	}

	log.Printf("Email sent to address: %v\n", cfg.Recipients)
	log.Println(result)

	return htmlBody, err
}

func buildEmailInput(subject, htmlBody, jsonFile string, cfg *config) (*ses.SendRawEmailInput, error) {
	msg := gomail.NewMessage()
	msg.SetHeader("From", cfg.Sender)
	msg.SetHeader("To", strings.Join(cfg.Recipients, ","))
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", htmlBody)
	msg.Attach(jsonFile)

	var s bytes.Buffer

	_, err := msg.WriteTo(&s)
	if err != nil {
		log.Fatalf("error writing to buffer: %v", err)
		return nil, err
	}

	raw := ses.RawMessage{
		Data: s.Bytes(),
	}
	input := &ses.SendRawEmailInput{
		Destinations: aws.StringSlice(cfg.Recipients),
		Source:       aws.String(cfg.Sender),
		RawMessage:   &raw,
	}

	return input, nil
}
