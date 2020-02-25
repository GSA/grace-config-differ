resource "aws_iam_role" "self" {
  name        = local.app_name
  description = "Role for GRACE Config Differ Lambda function"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF

}

resource "aws_iam_policy" "self" {
  name        = local.app_name
  description = "Policy for GRACE Config Differ Lambda function"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "config:DescribeConfigRuleEvaluationStatus",
        "config:GetResourceConfigHistory",
        "config:ListDiscoveredResources",
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents",
        "ses:SendRawEmail"
      ],
      "Effect": "Allow",
      "Resource": "*"
    },
    {
      "Action": [
        "s3:GetObject",
        "s3:ListBucket"
      ],
      "Effect": "Allow",
      "Resource": [
        "${local.s3_bucket_arn}",
        "${local.s3_bucket_arn}/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "kms:Decrypt",
        "kms:Encrypt"
      ],
      "Resource": "${var.kms_key_arn}"
    },
    {
      "Effect": "Allow",
      "Action": [
        "ssm:PutParameter",
        "ssm:GetParameter"
      ],
      "Resource": "arn:aws:ssm:${local.region}:${local.account_id}:parameter/${var.ssm_parameter_store}"
    }
  ]
}
EOF

}

resource "aws_iam_role_policy_attachment" "iam_role_policy_attachment" {
  role       = aws_iam_role.self.name
  policy_arn = aws_iam_policy.self.arn
}
