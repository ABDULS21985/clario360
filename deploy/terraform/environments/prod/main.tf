terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.27"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.12"
    }
  }

  backend "s3" {
    bucket = "clario360-terraform-state"
    key    = "prod/terraform.tfstate"
    region = "me-south-1"
  }
}

provider "aws" {
  region = var.aws_region
}

variable "aws_region" {
  default     = "me-south-1"
  description = "AWS region (Bahrain for Saudi proximity)"
}

variable "environment" {
  default = "prod"
}

variable "cluster_name" {
  default = "clario360-prod"
}
