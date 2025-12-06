# Terraform configuration for Spring PetClinic
# This is a simplified example. In production, you should provide your own terraform configuration.

terraform {
  required_version = ">= 1.0"
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "~> 3.0"
    }
  }
}

provider "docker" {}

# Docker container resource
resource "docker_container" "app" {
  name  = "petclinic-001"
  image = "eclipse-temurin:17-jre"
  
  command = ["java", "-jar", "/app/app.jar"]
  
  volumes {
    host_path      = "/home/chun/Develop/alm/workspace/spring-petclinic/source/spring-petclinic/target/spring-petclinic-4.0.0-SNAPSHOT.jar"
    container_path = "/app/app.jar"
  }
  
  ports {
    internal = 8080
    external = 8088
  }
}

output "container_id" {
  value = docker_container.app.id
}

