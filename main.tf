data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

locals {
  account_id    = data.aws_caller_identity.current.account_id
  app_name      = "grace-${var.appenv}-config-differ"
  region        = data.aws_region.current.name
  s3_bucket_arn = "arn:aws:s3:::${var.s3_bucket}"
}
