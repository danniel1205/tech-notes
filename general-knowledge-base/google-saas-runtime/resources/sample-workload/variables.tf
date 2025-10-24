# -----------------------------------------------------------------------------
# REQUIRED SAAS RUNTIME INPUT VARIABLES
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
# CUSTOM INPUT VARIABLES (FROM GKE CLUSTER UNIT)
# -----------------------------------------------------------------------------
# These variables will be populated by the outputs from Unit Kind 1.
# SaaS Runtime handles this mapping when you define the dependency.
# -----------------------------------------------------------------------------

variable "gke_cluster_name" {
  type        = string
  description = "The name of the GKE cluster to deploy to (from GKE unit output)."
}

variable "gke_cluster_location" {
  type        = string
  description = "The location of the GKE cluster (from GKE unit output)."
}