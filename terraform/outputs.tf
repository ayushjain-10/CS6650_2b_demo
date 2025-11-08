output "ecs_cluster_name" {
  description = "Name of the created ECS cluster"
  value       = module.ecs.cluster_name
}

output "ecs_service_name" {
  description = "Name of the running ECS service"
  value       = module.ecs.service_name
}

output "ecr_repository_url" {
  description = "ECR repository URL to push images"
  value       = module.ecr.repository_url
}

output "alb_dns_name" {
  description = "Public DNS name of the Application Load Balancer"
  value       = module.alb.dns_name
}

output "rds_endpoint" {
  description = "RDS MySQL endpoint"
  value       = module.rds.db_instance_endpoint
}

output "rds_database_name" {
  description = "RDS database name"
  value       = module.rds.db_name
}

output "dynamodb_table_name" {
  description = "DynamoDB shopping carts table name"
  value       = module.dynamodb.table_name
}