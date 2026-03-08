terraform {
  backend "gcs" {
    bucket = "clario360-terraform-state"
    prefix = "production"
  }
}
