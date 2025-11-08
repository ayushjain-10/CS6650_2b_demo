output "table_name" {
  description = "Name of the DynamoDB shopping carts table"
  value       = aws_dynamodb_table.shopping_carts.name
}

output "table_arn" {
  description = "ARN of the DynamoDB shopping carts table"
  value       = aws_dynamodb_table.shopping_carts.arn
}

output "table_id" {
  description = "ID of the DynamoDB shopping carts table"
  value       = aws_dynamodb_table.shopping_carts.id
}

