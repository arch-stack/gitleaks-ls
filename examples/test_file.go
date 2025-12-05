package examples

// This file contains intentional test secrets for demonstrating gitleaks-ls.
// DO NOT use these in production - they are fake/example values.

const (
	// Example AWS Access Key (fake - for testing only)
	awsKey = "AKIATESTKEYEXAMPLE7A"

	// Example GitHub Personal Access Token (fake - for testing only)
	githubToken = "ghp_1234567890abcdefghijklmnopqrstuvwx"
)

func main() {
	// This file is used to test the language server
	println(awsKey, githubToken)
}
