#tfsec:ignore:aws-lambda-enable-tracing
resource "aws_lambda_function" "self" {
  filename         = var.source_file
  function_name    = local.app_name
  description      = "Emails diffs in config service history"
  role             = aws_iam_role.self.arn
  handler          = "grace-config-differ"
  source_code_hash = filebase64sha256(var.source_file)
  kms_key_arn      = var.kms_key_arn
  runtime          = "go1.x"
  timeout          = 900

  # Limit to a single execution at a time
  reserved_concurrent_executions = 1

  environment {
    variables = {
      sender              = var.sender
      recipients          = var.recipients
      char_set            = var.char_set
      s3_bucket           = var.s3_bucket
      ssm_parameter_store = var.ssm_parameter_store
      kms_key_arn         = var.kms_key_arn
    }
  }
}

resource "aws_lambda_permission" "self" {
  statement_id  = "AllowExecutionFromS3Bucket"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.self.function_name
  principal     = "s3.amazonaws.com"
  source_arn    = local.s3_bucket_arn
}

resource "aws_s3_bucket_notification" "self" {
  bucket = var.s3_bucket

  lambda_function {
    lambda_function_arn = aws_lambda_function.self.arn
    events              = ["s3:ObjectCreated:*"]
    filter_prefix       = "awsconfig/AWSLogs/"
    filter_suffix       = ".json.gz"
  }
}
