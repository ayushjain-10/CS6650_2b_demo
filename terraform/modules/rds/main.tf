# Create DB subnet group
resource "aws_db_subnet_group" "this" {
  name       = "${var.service_name}-db-subnet-group"
  subnet_ids = var.subnet_ids

  tags = {
    Name = "${var.service_name}-db-subnet-group"
  }
}

# Security group for RDS instance
resource "aws_security_group" "rds" {
  name        = "${var.service_name}-rds-sg"
  description = "Security group for RDS MySQL instance - allows access from ECS only"
  vpc_id      = var.vpc_id

  # Allow MySQL/Aurora from ECS security group only
  ingress {
    from_port       = 3306
    to_port         = 3306
    protocol        = "tcp"
    security_groups = [var.ecs_security_group_id]
    description     = "MySQL access from ECS tasks"
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    description = "Allow all outbound traffic"
  }

  tags = {
    Name = "${var.service_name}-rds-sg"
  }
}

# RDS MySQL instance
resource "aws_db_instance" "this" {
  identifier           = "${var.service_name}-mysql"
  engine               = "mysql"
  engine_version       = var.engine_version
  instance_class       = var.instance_class
  allocated_storage    = var.allocated_storage
  storage_type         = "gp2"
  storage_encrypted    = false

  db_name  = var.db_name
  username = var.db_username
  password = var.db_password

  db_subnet_group_name   = aws_db_subnet_group.this.name
  vpc_security_group_ids = [aws_security_group.rds.id]

  # Make it publicly accessible from within VPC but not from internet
  publicly_accessible = false

  # Backup configuration
  backup_retention_period = 0
  skip_final_snapshot     = true
  deletion_protection     = false

  # Monitoring
  enabled_cloudwatch_logs_exports = ["error", "general", "slowquery"]

  # Performance Insights (optional but useful)
  performance_insights_enabled = false

  # Parameter group for MySQL 8.0
  parameter_group_name = "default.mysql8.0"

  tags = {
    Name = "${var.service_name}-mysql"
  }
}

