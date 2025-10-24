# outputs.tf (Corrected for ClusterIP)

data "kubernetes_service" "nginx_service" {
  metadata {
    name      = "${helm_release.nginx_ingress.name}-nginx"
    namespace = helm_release.nginx_ingress.namespace
  }
  depends_on = [
    helm_release.nginx_ingress,
    time_sleep.wait_for_service_ip
  ]
}

output "nginx_internal_cluster_ip" {
  description = "The internal cluster IP of the NGINX service."
  # Read the .spec.cluster_ip instead of the load balancer status
  value = data.kubernetes_service.nginx_service.spec[0].cluster_ip
}