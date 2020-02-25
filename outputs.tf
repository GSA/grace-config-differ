output "lambda_function_arn" {
  value       = aws_lambda_function.self.arn
  description = "The Amazon Resource Name (ARN) identifying the Lambda Function"
}

output "lambda_function_last_modified" {
  value       = aws_lambda_function.self.last_modified
  description = "The date this resource was last modified"
}

