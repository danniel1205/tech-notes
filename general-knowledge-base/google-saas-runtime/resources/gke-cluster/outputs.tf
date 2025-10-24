# outputs.tf (FOR GKE CLUSTER UNIT)

output "gke_cluster_name" {
  description = "The name of the created GKE cluster."
  value       = google_container_cluster.primary.name
}

output "gke_cluster_location" {
  description = "The location (region) of the GKE cluster."
  value       = google_container_cluster.primary.location
}