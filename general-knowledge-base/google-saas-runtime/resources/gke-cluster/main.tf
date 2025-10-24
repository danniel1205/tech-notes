# main.tf

# -----------------------------------------------------------------------------
# REQUIRED SAAS RUNTIME INPUT VARIABLES
# -----------------------------------------------------------------------------
# These variables are automatically populated by SaaS Runtime when a unit
# is created. You MUST declare them so Terraform can accept the values.
# -----------------------------------------------------------------------------

variable "tenant_project_id" {
  type        = string
  description = "The project ID of the tenant project where resources will be deployed."
}

variable "tenant_project_number" {
  type        = string
  description = "The project number of the tenant project."
}

# -----------------------------------------------------------------------------
# PROVIDER CONFIGURATION
# -----------------------------------------------------------------------------
# The provider now uses the injected 'project_id' from SaaS Runtime
# instead of a hardcoded value.
# -----------------------------------------------------------------------------

provider "google" {
  project = var.tenant_project_id # USE THE VARIABLE HERE
  region  = "<replace>" # e.g., us-west2
}

# -----------------------------------------------------------------------------
# RESOURCE DEFINITION
# -----------------------------------------------------------------------------
# Your GKE cluster resource remains largely the same.
# -----------------------------------------------------------------------------

resource "google_container_cluster" "primary" {
  # Add the project attribute here for clarity, using the passed-in variable.
  project  = var.tenant_project_id
  name     = "<replace>" # e.g., basic-gke-cluster
  location = "<replace>" # e.g., us-west2

  initial_node_count = 1

  node_config {
    machine_type = "e2-medium"
    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform"
    ]
  }
}