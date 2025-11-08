# Region to deploy into
variable "aws_region" {
  type    = string
  default = "us-west-2"
}

# ECR & ECS settings
variable "ecr_repository_name" {
  type    = string
  default = "ecr_service"
}

variable "service_name" {
  type    = string
  default = "CS6650L2"
}

variable "container_port" {
  type    = number
  default = 8080
}

variable "ecs_count" {
  type    = number
  default = 1
}

# How long to keep logs
variable "log_retention_days" {
  type    = number
  default = 7
}

# Tag of the image to deploy from the created ECR repo
variable "image_tag" {
  type        = string
  description = "Image tag to deploy from repository_url"
  default     = "latest"
}

# Database password (should be set via environment variable or terraform.tfvars)
variable "db_password" {
  type        = string
  description = "Master password for RDS MySQL instance"
  sensitive   = true
  default     = "MySecurePass123!"
}
