# main.tf (FOR K8S APP UNIT)

# -----------------------------------------------------------------------------
# PROVIDER CONFIGURATION
# -----------------------------------------------------------------------------

provider "google" {
  project = var.tenant_project_id
}

# -----------------------------------------------------------------------------
# DATA SOURCE: GET GKE CLUSTER CREDENTIALS
# -----------------------------------------------------------------------------
# Use the input variables (from Unit 1) to find the cluster and get its auth data.
# -----------------------------------------------------------------------------

data "google_container_cluster" "cluster" {
  project  = var.tenant_project_id
  name     = var.gke_cluster_name
  location = var.gke_cluster_location
}

# -----------------------------------------------------------------------------
# PROVIDER CONFIGURATION: KUBERNETES & HELM
# -----------------------------------------------------------------------------
# Configure the Kubernetes and Helm providers using the cluster data we just fetched.
# -----------------------------------------------------------------------------

provider "kubernetes" {
  host                   = "https://${data.google_container_cluster.cluster.endpoint}"
  cluster_ca_certificate = base64decode(data.google_container_cluster.cluster.master_auth[0].cluster_ca_certificate)
  token                  = data.google_client_config.default.access_token
}

provider "helm" {
  kubernetes = {
    host                   = "https://${data.google_container_cluster.cluster.endpoint}"
    cluster_ca_certificate = base64decode(data.google_container_cluster.cluster.master_auth[0].cluster_ca_certificate)
    token                  = data.google_client_config.default.access_token
  }
}

data "google_client_config" "default" {}

# -----------------------------------------------------------------------------
# RESOURCE: DEPLOY NGINX VIA HELM
# -----------------------------------------------------------------------------
# This creates a Helm release for NGINX from the public Bitnami chart repo.
# We set the service type to LoadBalancer so we can get an external IP.
# -----------------------------------------------------------------------------

resource "helm_release" "nginx_ingress" {
  name       = "my-nginx"
  repository = "https://charts.bitnami.com/bitnami"
  chart      = "nginx"
  version    = "15.5.1" # Pinning version is best practice
  namespace  = "default"

  # Set values for the Helm chart
  set = [
    {
      name  = "service.type"
      value = "ClusterIP"
    }
  ]
}

resource "time_sleep" "wait_for_service_ip" {
  # This sleep will only start AFTER the helm release is "done"
  depends_on = [helm_release.nginx_ingress]

  create_duration = "60s"
}