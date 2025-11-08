# DynamoDB Table for Shopping Carts
# Single-table design with embedded cart items

resource "aws_dynamodb_table" "shopping_carts" {
  name           = "${var.service_name}-shopping-carts"
  billing_mode   = var.billing_mode
  hash_key       = "cart_id"
  
  # Only set capacity if using PROVISIONED billing mode
  read_capacity  = var.billing_mode == "PROVISIONED" ? var.read_capacity : null
  write_capacity = var.billing_mode == "PROVISIONED" ? var.write_capacity : null

  attribute {
    name = "cart_id"
    type = "S"  # String type for UUID
  }

  attribute {
    name = "customer_id"
    type = "S"
  }

  # Global Secondary Index for querying carts by customer
  global_secondary_index {
    name            = "CustomerIndex"
    hash_key        = "customer_id"
    projection_type = "ALL"
    
    # Only set capacity for PROVISIONED mode
    read_capacity  = var.billing_mode == "PROVISIONED" ? var.read_capacity : null
    write_capacity = var.billing_mode == "PROVISIONED" ? var.write_capacity : null
  }

  # Enable point-in-time recovery for production
  point_in_time_recovery {
    enabled = false  # Disabled for assignment/cost
  }

  # TTL for automatic cart cleanup (optional)
  ttl {
    attribute_name = "expires_at"
    enabled        = true
  }

  tags = {
    Name        = "${var.service_name}-shopping-carts"
    Environment = "Development"
  }
}

