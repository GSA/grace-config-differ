variable "appenv" {
  type        = string
  description = "(optional) The environment in which the script is running (development | test | production)"
  default     = "development"
}

variable "source_file" {
  type        = string
  description = "(optional) full or relative path to zipped binary of lambda handler"
  default     = "../release/grace-config-differ.zip"
}

variable "sender" {
  type        = string
  description = "(required) eMail address of sender for AWS SES"
}

variable "recipients" {
  type        = string
  description = "(required) comma delimited list of AWS SES eMail recipients"
}

variable "char_set" {
  type        = string
  description = "(optional) character set (Default: UTF-8)"
  default     = "UTF-8"
}

variable "s3_bucket" {
  type        = string
  description = "(required) S3 bucket name/id where config service histories and snapshots are saved"
}

variable "kms_key_arn" {
  type        = string
  description = "(required) ARN of KMS key to decrypt config service histories and snapshots"
}

variable "ssm_parameter_store" {
  type        = string
  description = "(required) Name of AWS parameter store for LastSuccessfulEvaluationTime"
}
