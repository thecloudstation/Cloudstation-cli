variable "job_name" {
  description = "The name to use as the job name which overrides using the pack name"
  type        = string
  default = ""
}

variable "region" {
  description = "The region where jobs will be deployed"
  type        = string
  default     = ""
}

variable "node_pool" {
  description = "The pool where jobs will be deployed"
  type        = string
  default     = "minions"
}

variable "gpu_type" {
  description = "The type of gpu to use"
  type        = string
  default     = ""
}

variable "datacenters" {
  description = "A list of datacenters in the region which are eligible for task placement"
  type        = list(string)
  default     = ["*"]
}

variable "private_registry" {
  description = "The private registry to use for the job"
  type        = string
  default     = ""
}

variable "user" {
  description = "docker user"
  type        = string
  default     = ""
}

variable "private_registry_provider" {
  description = "The private registry provider to use for the job"
  type        = string
  default     = ""
}

variable "job_config" {
  description = ""
  type = object({
    type    = string
    cron = string
    prohibit_overlap = bool
    payload = string
    meta_required = list(string)
    meta_optional = list(string)
  })
    default = {
    type    = "service"
    cron = ""
    prohibit_overlap = true
    payload = ""
    meta_required =[]
    meta_optional = []
    // meta_required = ["tenant_id", "project_id", "service_id"]
    }
  }

variable "image" {
  description = ""
  type        = string
  default     = "mnomitch/hello_world_server"
}

variable "entrypoint" {
  description = "list of entrypoint commands to run in the container"
  type        = list(string)
  default     = []
}

variable "args" {
  description = "list of commands to run in the container"
  type        = list(string)
  default     = []
}

variable "command" {
  description = "nomad command"
  type        = string
  default     = ""
}
variable "count" {
  description = "The number of app instances to deploy"
  type        = number
  default     = 1
}

variable "use_csi_volume" {
  description = "Use a csi volume as defined in the Nomad client configuration"
  type        = bool
  default     = false
}

variable "volume_name" {
  description = "The name of the volume to mount"
  type        = string
  default     = "volumename"
}

variable "volume_mount_destination" {
  description = "The destination path to mount the volume"
  type        = list(string)
  default     = ["/data"]
}

variable "autoscale" {
  description = "autoscaling enabled"
  type        = bool
  default     = false
}

variable "regions" {
  description = "list of regions to use "
  type        = string
  default     = ""
}

variable "multi_region" {
  description = "deploy in multi region "
  type        = bool
  default     = false
}

variable "config_files" {
  description = "Config files to be used by the job"
  type = list(object({
    path = string
    content = string
  }))
  default = []
  // default     = [
  //   {
  //     path     = "loki/local-config.yaml"
  //     content = "YXV0aF9lbmFibGVkOiB0cnVlCgpzZXJ2ZXI6CiAgaHR0cF9saXN0ZW5fcG9ydDogMzEwMAoKaW5nZXN0ZXI6CiAgbGlmZWN5Y2xlcjoKICAgIGFkZHJlc3M6IDEyNy4wLjAuMQogICAgcmluZzoKICAgICAga3ZzdG9yZToKICAgICAgICBzdG9yZTogaW5tZW1vcnkKICAgICAgcmVwbGljYXRpb25fZmFjdG9yOiAxCiAgICBmaW5hbF9zbGVlcDogMHMKICBjaHVua19pZGxlX3BlcmlvZDogMWgKICBtYXhfY2h1bmtfYWdlOiAxaAogIGNodW5rX3RhcmdldF9zaXplOiAxMDQ4NTc2CiAgY2h1bmtfcmV0YWluX3BlcmlvZDogMzBzCiAgbWF4X3RyYW5zZmVyX3JldHJpZXM6IDAKICAKc2NoZW1hX2NvbmZpZzoKICBjb25maWdzOgogICAgLSBmcm9tOiAyMDIzLTA0LTI0CiAgICAgIHN0b3JlOiBib2x0ZGItc2hpcHBlcgogICAgICBvYmplY3Rfc3RvcmU6IHMzCiAgICAgIHNjaGVtYTogdjExCiAgICAgIGluZGV4OgogICAgICAgIHByZWZpeDogaW5kZXhfCiAgICAgICAgcGVyaW9kOiAyNGgKCnN0b3JhZ2VfY29uZmlnOgogIGJvbHRkYl9zaGlwcGVyOgogICAgYWN0aXZlX2luZGV4X2RpcmVjdG9yeTogL2xva2kvaW5kZXgKICAgIGNhY2hlX2xvY2F0aW9uOiAvbG9raS9jYWNoZQogICAgY2FjaGVfdHRsOiAyNGgKICAgIHNoYXJlZF9zdG9yZTogczMKICBhd3M6CiAgICBzMzogaHR0cDovLzE3Mi4xNi4xLjEyOjIwMTc0L2xvZ3MKICAgIGJ1Y2tldG5hbWVzOiBsb2tpCiAgICBhY2Nlc3Nfa2V5X2lkOiBsb2tpCiAgICBzZWNyZXRfYWNjZXNzX2tleTogY2xvdWRzdGF0aW9ubG9ncwogICAgaW5zZWN1cmU6IHRydWUgCiAgICBzM2ZvcmNlcGF0aHN0eWxlOiB0cnVlCgoKY29tcGFjdG9yOgogIHdvcmtpbmdfZGlyZWN0b3J5OiAvdG1wL2xva2kvY29tcGFjdG9yCiAgc2hhcmVkX3N0b3JlOiBzMwoKbGltaXRzX2NvbmZpZzoKICByZWplY3Rfb2xkX3NhbXBsZXM6IHRydWUKICByZWplY3Rfb2xkX3NhbXBsZXNfbWF4X2FnZTogMTY4aAoKY2h1bmtfc3RvcmVfY29uZmlnOgogIG1heF9sb29rX2JhY2tfcGVyaW9kOiAwcwoKdGFibGVfbWFuYWdlcjoKICByZXRlbnRpb25fZGVsZXRlc19lbmFibGVkOiB0cnVlCiAgcmV0ZW50aW9uX3BlcmlvZDogNzIwaA=="
  //   }
  // ]
}

variable "script" {
  description = "some script"
  type        = string
  default     = ""
}

variable "secret_path" {
  description = "shared secret path"
  type        = string
  default     = ""
}

variable "shared_secret_path" {
  description = "shared secret path"
  type        = string
  default     = ""
}

variable "uses_kv_engine" {
  description = "Whether project uses shared kv engine or dedicated mount"
  type        = bool
  default     = false
}

variable "owner_uses_kv_engine" {
  description = "Whether owner/user uses shared kv engine or dedicated mount"
  type        = bool
  default     = false
}

variable "restart_attempts" {
  description = "The number of times the task should restart on updates"
  type        = number
  default     = 2
}

variable "restart_mode" {
  description = "The number of times the task should restart on updates"
  type        = string
  default     = "delay"
}

variable "restart" {
  description = "Restart policy configuration for the task"
  type = object({
    attempts = number
    interval = string
    delay    = string
    mode     = string
  })
  default = {
    attempts = 5
    interval = "3m"
    delay    = "15s"
    mode     = "delay"
  }
}

variable "ephemeral_disk" {
  description = "Ephemeral disk configuration for the task group"
  type = object({
    enabled = bool
    migrate = bool
    size    = number
    sticky  = bool
  })
  default = {
    enabled = false
    migrate = false
    size    = 300
    sticky  = true
  }
}

variable "privileged" {
  description = "Enable privileged mode for Docker containers"
  type        = bool
  default     = false
}

variable "health_check" {
  description = ""
  type = object({
    path     = string
    interval = string
    timeout  = string
    port     = number
    type     = string
  })

  default = {
    path     = "/"
    interval = "10s"
    timeout  = "30s"
    port     = 8000
    type     = "http"
  }
}

variable "has_vault_policies" {
  description = "If service has vault policies attached"
  type        = bool
  default     = false
}

variable "update" {
  description = "Job update parameters"
  type = object({
    min_healthy_time  = string
    healthy_deadline  = string
    progress_deadline = string
    auto_revert       = bool
    auto_promote      = bool
    max_parallel      = number
    canary            = number
    health_check      = string
    stagger           = string
  })
  default = {
    min_healthy_time  = "15s",
    healthy_deadline  = "40m",
    progress_deadline = "1h",
    auto_revert       = true,
    auto_promote      = true,
    max_parallel      = 1,
    canary            = 1
    health_check      = "checks",
    stagger           = "20s"
  }
}

variable "tcp_health_check" {
  description = "TCP Consul health check details"
  type = object({
    port     = number
    interval = string
    timeout  = string
  })
  default = {
    interval = "10s"
    timeout  = "2s"
    port     = 6000
  }
}


variable "cluster_domain" {
  description = "CloudStation cluster domain"
  type        = string
  default     = "cloud-station.app"
}

variable "network" {
  description = ""
  type = list(object({
    name = string
    port = number
    type = string
    public = bool
    domain = string
    custom_domain = string
    has_health_check = string
    health_check = object({
      type     = string
      path     = string
      interval = string
      timeout  = string
      port     = number
    })
  }))

  default = [{
    name = "http"
    port = 8000
    type = "http"
    public = false
    domain = ""
    custom_domain = ""
    has_health_check = "tcp"
    health_check = {
      type     = "tcp"
      path     = "/"
      interval = "10s"
      timeout  = "30s"
      port     = 80
    }
  }]
}
variable "env_vars" {
  description = ""
  type = list(object({
    key   = string
    value = string
  }))
  default = []
}


variable "consul_service_name" {
  description = "The consul service name for the application"
  type        = string
  default     = "service"
}

variable "consul_linked_services" {
  description = ""
  type = list(object({
    key   = string
    value = string
  }))
  default = []
  // default = [
  //   {
  //     key   = "service"
  //     value = "service"
  //   }
  // ]
}

variable "vault_linked_secrets" {
  description = ""
  type = list(object({
    prefix      = string
    secret_path = string
  }))
  default = []
}

variable "template" {
  description = ""
  type = list(object({
    name    = string
    pattern = string
    service_name = string
    service_secret_path = string
    linked_vars = list(string)
  }))
  default = []
  // default = [{
  //   name    = "Connexion"
  //   pattern = "mysql://{{ $KILLBILL_DAO_USER }}:{{ $MYSQL_ROOT_PASSWORD }}@{{ .Address }}:{{ .Port }}/your_database_name"
  //   service_name = "cst-mariadb-nmoqgbye-3306"
  //   service_secret_path = "img_6414ecdf-f255-46c2-938c-f1c9fc534fa9"
  //   linked_vars = ["MYSQL_ROOT_PASSWORD", "KILLBILL_DAO_USER"]
  // },
  // {
  //   name    = "Connexion2"
  //   pattern = "mysql2://{{ $KILLBILL_DAO_USER }}:{{ $MYSQL_ROOT_PASSWORD }}@{{ .Address }}:{{ .Port }}/your_database_name"
  //   service_name = "cst-mariadb-nmoqgbye-3306"
  //   service_secret_path = "img_6414ecdf-f255-46c2-938c-f1c9fc534fa9"
  //   linked_vars = ["MYSQL_ROOT_PASSWORD2", "KILLBILL_DAO_USER2"]
  // }]
}


variable "consul_service_port" {
  description = "The consul service name for the application"
  type        = string
  default     = "http"
}

variable "useGpu" {
  description = "to be deleted"
  type        = bool
  default     = false
}

variable "resources" {
  description = "The resource to assign to the Nginx system task that runs on every client"
  type = object({
    cpu    = number
    memory = number
    gpu    = number
    memory_max = number
  })
  default = {
    cpu    = 200,
    memory = 200,
    gpu    = 0,
    memory_max = 2000
  }
}


variable "user_id" {
  description = "tenant id"
  type        = string
  default     = ""
}

variable "alloc_id" {
  description = "alloc id"
  type        = string
  default     = "deployment-xxx"
}

variable "project_id" {
  description = "project id"
  type        = string
  default     = ""
}

variable "service_id" {
  description = "service id"
  type        = string
  default     = "prj_integ_8d0e89bc-71ad-4f7a-8bb9-ee6c72820248"
}

variable "use_tls" {
  description = "if service connected to an other service using ssl"
  type        = bool
  default     = false
}


variable "tls" {
  description = "tls cert"
  type = list(object({
    cert_path   = string
    key_path    = string
    common_name = string
    pka_path    = string
    ttl         = string
  }))
  default = [
    {
      cert_path   = "local/certs/kafka.keystore.pem"
      key_path    = "local/certs/kafka.keystore.key"
      common_name = "backend.clients.kafka.acme.com"
      pka_path    = "kafka-int-ca/issue/kafka-client"
      ttl         = "3h"
    }
  ]
}