# Copyright (c) Cloudstation, Inc.

app {
  url    = "https://github.com/cloudstation/cloudstation"
  author = "cloudstation"
}

pack {
  name        = "cloudstation"
  description = "CloudStation Pack - Modern Nomad Pack v2 syntax for deploying standalone service instances"
  url         = "https://github.com/thecloudstation/cloudstation-packs/packs/cloudstation"
  version     = "1.0.0"
}

integration {
  identifier = "nomad/hashicorp/cloudstation"
  name       = "cloudstation"
}