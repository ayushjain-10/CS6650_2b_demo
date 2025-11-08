# Wire together four focused modules: network, ecr, logging, ecs.

module "network" {
  source         = "./modules/network"
  service_name   = var.service_name
  container_port = var.container_port
}

module "ecr" {
  source          = "./modules/ecr"
  repository_name = var.ecr_repository_name
}

module "logging" {
  source            = "./modules/logging"
  service_name      = var.service_name
  retention_in_days = var.log_retention_days
}

module "alb" {
  source             = "./modules/alb"
  service_name       = var.service_name
  container_port     = var.container_port
  vpc_id             = module.network.vpc_id
  subnet_ids         = module.network.subnet_ids
  security_group_ids = [module.network.security_group_id]
}

# Reuse an existing IAM role for ECS tasks
data "aws_iam_role" "lab_role" {
  name = "LabRole"
}

module "rds" {
  source                 = "./modules/rds"
  service_name           = var.service_name
  vpc_id                 = module.network.vpc_id
  subnet_ids             = module.network.subnet_ids
  ecs_security_group_id  = module.network.security_group_id
  db_password            = var.db_password
}

module "dynamodb" {
  source       = "./modules/dynamodb"
  service_name = var.service_name
}

module "ecs" {
  source             = "./modules/ecs"
  service_name       = var.service_name
  image              = "${module.ecr.repository_url}:${var.image_tag}"
  container_port     = var.container_port
  subnet_ids         = module.network.subnet_ids
  security_group_ids = [module.network.security_group_id]
  execution_role_arn = data.aws_iam_role.lab_role.arn
  task_role_arn      = data.aws_iam_role.lab_role.arn
  log_group_name     = module.logging.log_group_name
  ecs_count          = var.ecs_count
  region             = var.aws_region
  target_group_arn   = module.alb.target_group_arn
  
  # Pass RDS connection details as environment variables
  db_endpoint = module.rds.db_instance_endpoint
  db_name     = module.rds.db_name
  db_password = var.db_password
  
  # Pass DynamoDB table name
  dynamodb_table_name = module.dynamodb.table_name
}


