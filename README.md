# GRACE Config Differ [![License](https://img.shields.io/badge/license-CC0-blue)](LICENSE.md) [![GoDoc](https://img.shields.io/badge/go-documentation-blue.svg)](https://godoc.org/github.com/GSA/grace-config-differ/aws) [![CircleCI](https://circleci.com/gh/GSA/grace-config-differ.svg?style=shield)](https://circleci.com/gh/GSA/grace-config-differ) [![Go Report Card](https://goreportcard.com/badge/github.com/GSA/grace-config-differ)](https://goreportcard.com/report/github.com/GSA/grace-config-differ)

Lambda function to email a report of configuration changes detected by AWS Config Service

## Repository contents

**[handler](handler/)** - Lambda function written in [Go](golang.org) to to email a report of configuration changes detected by AWS Config Service

**[terraform](https://github.com/GSA/grace-config-differ)** - Terraform module to install the Lambda function and set IAM roles, policies and environment variables

## Security Compliance

**Component ATO status:** draft

**Relevant controls:**

Control    | CSP/AWS | HOST/OS | App/DB | How is it implemented?
---------- | ------- | ------- | ------ | ----------------------
[CM-8(3)(a)](https://nvd.nist.gov/800-53/Rev4/control/CM-8) | ╳ | | | Employs an automated Lambda function triggered by AWS Config Service writes to the ConfigHistory and ConfigSnapshot directories in the logging AWS S3 bucket. These occur approximately every three hours. Gets the latest execution of the Config Service history, gets the config history for all Config Service resource types at this time (+/- 5 minutes) and compares them to the previous snapshot.
[CM-8(3)(b)](https://nvd.nist.gov/800-53/Rev4/control/CM-8) | ╳ | | | When changes are detected, notifies personnel specified by the `recipients` variable (grace-dev-alerts@gsa.gov) via email using AWS Simple Email Service (SES).

## Usage

### Download (recommended)

Download the zip compressed executable (Note: Replace v0.1.0 with desired release):

```
mkdir -p release
curl -L https://github.com/GSA/grace-config-differ/releases/download/v0.1.0/grace-config-differ.zip -o release/grace-config-differ.zip
```

### Compile

Alternatively, you can compile the Lambda function handler yourself:

```
cd handler
GOOS=linux GOARCH=amd64 go build -o ../release/grace-config-differ -v
zip -j ../release/grace-config-differ.zip ../release/grace-config-differ
```

### Add Module

Add the module to your terraform project. Ensure path to `source_file` matches
where you downloaded the zip file. Replace v0.1.0 with desired release. Example below:

```
module "grace-config-differ-lambda" {
  source              = "github.com:GSA/grace-config-differ?ref=v0.1.0"
  source_file         = "../release/grace-config-differ.zip"
  appenv              = "development"
  sender              = "validated-sender@email.com"
  recipients          = "recipient@email.com,other-recipient@email.com"
  s3_bucket           = "config-service-logging-bucket"
  kms_key_arn         = "arn:aws:kms:us-east-1:123456789012:key/59a975ee-a788-4b09-baed-ef3ed0f6741c"
  ssm_parameter_store = "unique_name"
}
```

### Module variables ###

| Variable | Type | Default | Description |
| -------- | ---- | ------- | ----------- |
| appenv | string | development | (optional) The environment in which the script is running (development &vert; test &vert; production) |
| source_file | string | ../release/grace-config-differ.zip | (optional) full or relative path to zipped binary of lambda handler |
| sender | string | | (required) eMail address of sender for AWS SES |
| recipients | string | | (required) comma delimited list of AWS SES eMail recipients |
| char_set | string | UTF-8 | (optional) character set (Default: UTF-8) |
| s3_bucket | string | | (required) S3 bucket name/id where config service histories and snapshots are saved |
| kms_key_arn | string | | (required) ARN of KMS key to decrypt config service histories and snapshots |
| ssm_parameter_store | string | (required) Name of AWS parameter store for LastSuccessfulEvaluationTime |

## Public domain

This project is in the worldwide [public domain](LICENSE.md). As stated in [CONTRIBUTING](CONTRIBUTING.md):

> This project is in the public domain within the United States, and copyright and related rights in the work worldwide are waived through the [CC0 1.0 Universal public domain dedication](https://creativecommons.org/publicdomain/zero/1.0/).
>
> All contributions to this project will be released under the CC0 dedication. By submitting a pull request, you are agreeing to comply with this waiver of copyright interest.
test
