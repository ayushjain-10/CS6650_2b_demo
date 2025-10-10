# Specify where to find the AWS & Docker providers
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# Configure AWS credentials & region
provider "aws" {
  region     = var.aws_region
}

# Configure Docker provider; authentication should be handled by your local Docker/ECR helper
 