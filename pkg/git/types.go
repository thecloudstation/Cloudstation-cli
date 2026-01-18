package git

// Provider represents a Git provider
type Provider string

const (
	GitHub    Provider = "github"
	GitLab    Provider = "gitlab"
	Bitbucket Provider = "bitbucket"
)

// GetBaseURL returns the base URL for a provider
func GetBaseURL(provider Provider) string {
	switch provider {
	case GitHub:
		return "github.com"
	case GitLab:
		return "gitlab.com"
	case Bitbucket:
		return "bitbucket.org"
	default:
		return "github.com"
	}
}

// BuildAuthURL builds an authenticated Git URL for the given repository and provider
func BuildAuthURL(repository, token string, provider Provider) string {
	baseURL := GetBaseURL(provider)

	if token == "" {
		// Public repository
		return "https://" + baseURL + "/" + repository + ".git"
	}

	if provider == Bitbucket {
		// Bitbucket uses x-token-auth
		return "https://x-token-auth:" + token + "@" + baseURL + "/" + repository + ".git"
	}

	// GitHub and GitLab use x-access-token
	return "https://x-access-token:" + token + "@" + baseURL + "/" + repository + ".git"
}
