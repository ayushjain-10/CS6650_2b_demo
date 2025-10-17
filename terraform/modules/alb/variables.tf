variable "service_name" {
  description = "Base name for ALB resources"
  type        = string
}

variable "container_port" {
  description = "Port the container listens on"
  type        = number
}

variable "vpc_id" {
  description = "VPC ID for the ALB"
  type        = string
}

variable "subnet_ids" {
  description = "Subnets for the ALB"
  type        = list(string)
}

variable "security_group_ids" {
  description = "Security groups for the ALB"
  type        = list(string)
}

variable "health_check_path" {
  description = "Health check path"
  type        = string
  default     = "/health"
}

