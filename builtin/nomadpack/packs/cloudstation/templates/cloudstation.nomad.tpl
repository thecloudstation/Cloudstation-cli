job [[ template "job_name" . ]] {
meta {
  run_uuid = "${uuidv4()}"
}

[[- if gt (var "resources.gpu" .) 0 ]]
node_pool = "gpu-minions"
[[- if var "gpu_type" . ]]
constraint {
  attribute = "${node.class}"
  operator  = "set_contains_any"
  value     = [[ var "gpu_type" . | quote ]]
}
[[- end ]]
[[- else ]]
[[- if var "node_pool" . ]]
node_pool = [[ var "node_pool" . | quote ]]
[[- end ]]
[[- end ]]


[[- if and (var "regions" .) (eq (var "resources.gpu" .) 0) ]]

constraint {
  attribute = "${node.class}"
  operator  = "set_contains_any"
  value     = [[ var "regions" . | quote ]]
}
[[- end ]]

[[- if var "multi_region" . ]]
constraint {
  distinct_hosts = true
}
[[- end ]]
[[ template "region" . ]]
type = [[ var "job_config.type" . | quote ]]

[[- if var "job_config.cron" . ]]
periodic {
  cron             = [[ var "job_config.cron" . | quote ]]
  prohibit_overlap = [[ var "job_config.prohibit_overlap" . ]]
}
[[- end ]]
[[- if var "job_config.meta_required" . ]]
parameterized {
  payload       = [[ var "job_config.payload" . | quote ]]
  meta_required = [[ var "job_config.meta_required" . | toStringList ]]
  [[- if var "job_config.meta_optional" . ]]
  meta_optional = [[ var "job_config.meta_optional" . | toStringList ]]
  [[- end ]]
}
[[- end]]


vault {
  [[- if var "private_registry" . ]]
  [[- $project_id := var "project_id" . ]]
  [[- $private_registry := var "private_registry" . ]]
  policies = [
    "[[ $private_registry ]]",
    "[[ $project_id ]]"
  ]
  [[- else ]]
  policies = ["[[ var "project_id" . ]]", "acr"]
  [[- end]]

  change_mode = "restart"
}
[[- if eq (var "job_config.type" .) "service" ]]

update {
  min_healthy_time  = [[ var "update.min_healthy_time" . | quote ]]
  healthy_deadline  = [[ var "update.healthy_deadline" . | quote ]]
  progress_deadline = [[ var "update.progress_deadline" . | quote ]]
  auto_revert       = [[ var "update.auto_revert" . ]]
  stagger           = [[ var "update.stagger" . | quote ]]
  health_check      = [[ var "update.health_check" . | quote ]]
  [[- if var "use_csi_volume" . ]]
  auto_promote = false
  canary       = 0
  [[- else ]]
  auto_promote = [[ var "update.auto_promote" . ]]
  canary       = [[ var "update.canary" . ]]
  [[- end ]]
}

migrate {
    max_parallel     = 2
    health_check     = "checks"
    min_healthy_time = [[ var "update.min_healthy_time" . | quote ]]
    healthy_deadline = [[ var "update.healthy_deadline" . | quote ]]
}
[[- end]]
group "[[ var "consul_service_name" . ]]" {
  count = [[ var "count" . ]]
  [[- if var "use_csi_volume" . ]]
  volume "[[ var "volume_name" . ]]" {
    type            = "csi"
    source          = "[[ var "volume_name" . ]]"
    read_only       = false
    access_mode     = "single-node-writer"
    attachment_mode = "file-system"
    per_alloc       = true

  }
  [[- end ]]
  [[- if var "ephemeral_disk.enabled" . ]]
  ephemeral_disk {
    migrate = [[ var "ephemeral_disk.migrate" . ]]
    size    = [[ var "ephemeral_disk.size" . ]]
    sticky  = [[ var "ephemeral_disk.sticky" . ]]
  }
  [[- end ]]
  network {
    [[- range $network := var "network" . ]]
    port [[ $network.name | quote ]] {
    to = [[ $network.port ]]
  }
  [[- end ]]
}

[[ $consulServiceNname := var "consul_service_name" . ]]
[[ $healthCheck := var "health_check" . ]]
[[ $tcpHealthCheck := var "tcp_health_check" . ]]
[[ $cluster_domain := var "cluster_domain" . ]]
[[- if var "network" . ]]
[[- range $network := var "network" . ]]
service {
  name = "[[ $consulServiceNname ]]-[[ $network.name ]]"
  port = "[[ $network.port ]]"
  [[- if ne $network.health_check.type "none" ]]
  check {
    type     = "[[ $network.health_check.type ]]"
    interval = "[[ $network.health_check.interval ]]"
    timeout  = "[[ $network.health_check.timeout ]]"
    port     = "[[ $network.health_check.port ]]"
    [[- if eq $network.health_check.type "http" ]]
      [[- if $network.health_check.path ]]
          path = "[[ $network.health_check.path ]]"
      [[- else ]]
          path = "/"
      [[- end ]]
    [[- end ]]
  }
  [[- else ]]
  # No health check
  [[- end ]]
  [[- if $network.public ]]
  [[- $domain := $network.domain ]]
  [[- $custom_domain := $network.custom_domain ]]
  tags = [
    "blue"
    [[- if ne $network.domain "" ]],
    "urlprefix-[[ $domain ]].[[ $cluster_domain ]]"
    [[- end ]]
    [[- if ne $network.custom_domain "" ]],
    "custom-[[ $custom_domain ]]"
    [[- end ]]
    [[- if eq $network.type "tcp" ]],
    "tcp-lb"
    [[- end ]]
  ]
  canary_tags = [
    "green",
    [[- if ne $network.domain "" ]]
    "urlprefix-canary-[[ $domain ]].[[ $cluster_domain ]]",
    [[- end ]]
    [[- if ne $network.custom_domain "" ]]
    "custom-canary-[[ $custom_domain ]]",
    [[- end ]]
  ]
  [[- else ]]
  tags = [
    "blue"
  ]
  [[- end ]]
}
[[- else ]]
service {
  name = "[[ $consulServiceNname ]]"
  tags = ["urlprefix-[[ $consulServiceNname ]].internal.[[ $cluster_domain ]]"]
}
[[- end ]]
[[- end ]]

restart {
  attempts = [[ var "restart.attempts" . ]]
  interval = [[ var "restart.interval" . | quote ]]
  delay    = [[ var "restart.delay" . | quote ]]
  mode     = [[ var "restart.mode" . | quote ]]
}

task "[[ var "alloc_id" . ]]" {
  driver = "docker"
  [[- if var "user" . ]]
  user = [[ var "user" . | quote ]]
  [[- else ]]
  user = "0"
  [[- end ]]
  [[- if var "use_csi_volume" . ]]
  [[- $volumeName := var "volume_name" . ]]
  [[- range $dest := var "volume_mount_destination" . ]]
  volume_mount {
    volume      = "[[ $volumeName ]]"
    destination = "[[ $dest ]]"
    read_only   = false
  }
  [[- end ]]
  [[- end ]]

  config {
    image              = [[var "image" . | quote]]
    image_pull_timeout = "30m"
    [[- if var "privileged" . ]]
    privileged = true
    [[- end ]]
    [[- if or (regexMatch ".*acrbc001.azurecr.io/.*" (var "image" .)) (var "private_registry" .) ]]
    auth {
      username = "${registry_username}"
      password = "${registry_password}"
    }
    [[- end ]]

    [[- if or (var "args" .) (var "entrypoint" .) ]]
    [[- if var "entrypoint" . ]]
    entrypoint = [[var "entrypoint" . | toStringList]]
    [[- else ]]
    entrypoint = ["/bin/sh", "-c"]
    [[- end ]]

    [[- if var "args" . ]]
    args = [[var "args" . | toStringList]]
    [[- end ]]
    [[- end ]]

    [[- if var "command" . ]]
    command = [[var "command" . | quote]]
    [[- end ]]
    [[- if var "network" . ]]
    ports = [
      [[- range $network := var "network" . ]]
      "[[ $network.name ]]",
      [[- end ]]
    ]
    [[- end ]]

    [[- if var "config_files" . ]]
    volumes = [
      [[- range $config := var "config_files" . ]]
      "local/[[ $config.path ]]:/[[ $config.path ]]",
      [[- end ]]
    ]
    [[- end ]]

    labels = {
      job        = "[[ var "job_name" . ]]"
      group      = "[[ var "consul_service_name" . ]]"
      task       = "[[ var "consul_service_name" . ]]"
      alloc_id   = "${NOMAD_ALLOC_ID}"
      user_id    = "[[ var "user_id" . ]]"
      project_id = "[[ var "project_id" . ]]"
      service_id = "[[ var "service_id" . ]]"
    }
  }
  template {
    data        = <<EOH
      {{ with secret "/kv/data/[[var "project_id" .]]/[[ var "service_id" . ]]" }}
          {{- range $key, $value := .Data.data }}
          {{ $key }}={{ $value }}
          {{- end }}
      {{ end }}
      EOH
    destination = "secrets/file.env"
    change_mode = "noop"
    env         = true
  }
  [[ if var "shared_secret_path" . ]]
  template {
    data        = <<EOH
      {{ with secret "/kv/data/[[var "project_id" .]]/[[ var "shared_secret_path" . ]]" }}
          {{- range $key, $value := .Data.data }}
          {{ $key }}={{ $value }}
          {{- end }}
      {{ end }}
      EOH
    destination = "secrets/shared-secrets.env"
    change_mode = "noop"
    env         = true
  }
  [[end]]

  [[- if var "private_registry" . ]]
  template {
    data        = <<EOH
      {{ with secret "/kv/data/[[var "private_registry" .]]/[[ var "private_registry_provider" . ]]" }}
          {{- range $key, $value := .Data.data }}
          {{ $key }}={{ $value }}
          {{- end }}
      {{ end }}
      EOH
    destination = "private_registry/file.env"
    change_mode = "noop"
    env         = true
  }
  [[- else ]]
  template {
    data        = <<EOH
      {{ with secret "/acr/data/registry" }}
          {{- range $key, $value := .Data.data }}
          {{ $key }}={{ $value }}
          {{- end }}
      {{ end }}
      EOH
    destination = "private_registry/file.env"
    change_mode = "noop"
    env         = true
  }
  [[- end]]

  [[ if var "script" . ]]
  template {
    data        = <<EOH
 #!/bin/ash
 [[ var "script" . ]]
 exec "$@"

      EOH
    destination = "local/script.sh"
    change_mode = "restart"
    env         = false
    perms       = "777"

  }
  [[end]]

  [[ if var "consul_linked_services" . ]]
  template {
    data        = <<EOH
    [[- range $service := var "consul_linked_services" . ]]
      [[- $service_name := $service.value  ]]
      [[- if regexMatch ".*_HOST$" $service.value ]]
          [[- $service_name = trimSuffix "_HOST" $service.value ]]
      [[- else if regexMatch ".*_PORT$" $service.value ]]
          [[- $service_name = trimSuffix "_PORT" $service.value ]]
      [[- end ]]

      {{- with service "[[ $service_name ]]" }}
          {{- if . }}
            {{- with index . 0 }}
              [[- if regexMatch ".*_HOST$" $service.value ]]
                [[ $service.key ]] = {{ .Address }}
              [[- else if regexMatch ".*_PORT$" $service.value ]]
                [[ $service.key ]] = {{ .Port }}
              [[- else ]]
                [[ $service.key ]] = {{ .Address }}:{{ .Port }}
              [[- end ]]
            {{- end }}
          {{- end }}
      {{- end }}
    [[- end ]]
    EOH
    destination = "secrets/consul_services.env"
    change_mode = "restart"
    env         = true
  }
  [[ end ]]

  [[ if var "template" . ]]
  template {
    data        = <<EOH
    [[ $project_id := var "project_id" . ]]
    [[ $uses_kv := var "uses_kv_engine" . ]]
    [[- range $template := var "template" . ]]
    [[ $secret_path := $template.service_secret_path ]]
    [[ $service_name := $template.service_name ]]
    [[ $pattern := $template.pattern]]
      {{ with secret "/kv/data/[[ $project_id ]]/[[ $secret_path ]]" }}
      [[- range $var := $template.linked_vars ]]
            {{- $[[ $var ]] := .Data.data.[[ $var ]] }}
      [[- end]]

          {{- with service "[[ $service_name ]]" }}
                {{- if . }}
                  {{- with index . 0 }}
                [[ $template.name ]]= [[ $pattern | quote ]]
                  {{- end }}
                {{- end }}
            {{- end }}
      {{ end }}
  [[ end ]]
    EOH
    destination = "secrets/connection.env"
    change_mode = "restart"
    env         = true
  }
  [[ end ]]

  [[- if var "use_tls" . ]]
  [[- range $cert := var "tls" . ]]
  template {
    data        = <<EOF
{{ with secret "[[ $cert.pka_path ]]" "common_name=[[ $cert.common_name ]]" "ttl=[[ $cert.ttl ]]" }}
{{ .Data.certificate }}
{{ end }}
EOF
    destination = "[[ $cert.cert_path ]]"
    perms       = "777"
  }

  template {
    data        = <<EOF
{{ with secret "[[ $cert.pka_path ]]" "common_name=[[ $cert.common_name ]]" "ttl=[[ $cert.ttl ]]" }}
{{ .Data.private_key }}
{{ end }}
EOF
    destination = "[[ $cert.key_path ]]"
    perms       = "777"
  }
  [[- end ]]
  [[- end ]]

  [[ if var "config_files" . ]]
  [[- range $config := var "config_files" . ]]
  template {
    data        = <<EOH
          {{- $configBase64 := [[ $config.content | quote ]]  -}}
          {{- $config := $configBase64 | base64Decode -}}
          {{ $config }}
        EOH
    destination = "local/[[ $config.path ]]"
    env         = false
  }
  [[ end ]]
  [[ end ]]

  resources {
    [[- if gt (var "resources.gpu" .) 0 ]]
    device "letmutx/gpu" {
      count = [[ var "resources.gpu" . ]]
    }
    [[- end ]]
    cpu        = [[ var "resources.cpu" . ]]
    memory     = [[ var "resources.memory" . ]]
    memory_max = [[ var "resources.memory_max" . ]]
  }
}

[[- if var "autoscale" . ]]
scaling {
  enabled = true
  min     = 1
  max     = 3

  policy {
    cooldown            = "5m"
    evaluation_interval = "1m"

    check "cpu_usage" {
      source = "prometheus"
      query = "avg(nomad_client_allocs_cpu_total_percent{exported_job=\"[[ var "consul_service_name" . ]]\"}) "
      strategy "target-value" {
        target    = 90
        threshold = 0.4  // Increased from 0.3 to 0.4
      }
    }

    check "memory_usage" {
      source = "prometheus"
      query  = "avg(nomad_client_allocs_memory_usage{exported_job=\"[[ var "consul_service_name" . ]]\"}) / 1024 / 1024"
      strategy "target-value" {
        target    = 240
        threshold = 0.4 // Increased from 0.3 to 0.4
      }
    }
  }
}
[[- end ]]
}
}