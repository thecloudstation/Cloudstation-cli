project = "cloudstation-cli"

app "cs" {
  path = "./cmd/cloudstation"

  build {
    use = "goreleaser"

    name       = "cs"
    path       = "./cmd/cloudstation"
    version    = env("VERSION")
    output_dir = "./dist"

    targets = [
      "linux/amd64",
      "linux/arm64",
      "darwin/amd64",
      "darwin/arm64",
      "windows/amd64"
    ]

    ldflags = "-X 'main.DefaultAPIURL=${CS_API_URL}' -X 'main.DefaultAuthURL=${CS_AUTH_URL}'"
  }

  registry {
    use = "github"

    repository     = "thecloudstation/cloudstation-orchestrator"
    token          = env("GITHUB_TOKEN")
    tag_name       = env("VERSION")
    release_name   = "CloudStation CLI v${VERSION}"
    generate_notes = true
    checksums      = true
  }

  deploy {
    use = "noop"
  }
}
