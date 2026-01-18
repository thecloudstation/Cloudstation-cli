job [[ var "job_name" . | quote ]] {
  datacenters = ["dc1"]
  type        = "service"

  group "app" {
    count = 1

    network {
      port "http" {
        to = [[ var "port" . ]]
      }
    }

    task "server" {
      driver = "docker"

      config {
        image = [[ var "image" . | quote ]]
        ports = ["http"]
      }

      resources {
        cpu    = [[ var "cpu" . ]]
        memory = [[ var "memory" . ]]
      }
    }
  }
}
