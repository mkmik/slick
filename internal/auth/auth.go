package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	slackAuthorizeURL = "https://slack.com/oauth/v2/authorize"
	slackTokenURL     = "https://slack.com/api/oauth.v2.access"
	userScopes        = "channels:history,groups:history,im:history,mpim:history,users:read"
)

// TokenPath returns the path to the stored token file.
func TokenPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "slick", "token"), nil
}

// LoadToken reads the stored OAuth token from disk.
func LoadToken() (string, error) {
	path, err := TokenPath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", fmt.Errorf("token file is empty")
	}
	return token, nil
}

// SaveToken writes the OAuth token to disk.
func SaveToken(token string) error {
	path, err := TokenPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(token+"\n"), 0600)
}

// Login performs the Slack OAuth v2 flow and returns the user access token.
// It starts a temporary local HTTP server, opens the browser for authorization,
// and exchanges the resulting code for a token.
func Login(clientID, clientSecret string) (string, error) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return "", fmt.Errorf("starting local server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)

	state, err := randomState()
	if err != nil {
		return "", fmt.Errorf("generating state: %w", err)
	}

	params := url.Values{
		"client_id":    {clientID},
		"user_scope":   {userScopes},
		"redirect_uri": {redirectURI},
		"state":        {state},
	}
	authURL := slackAuthorizeURL + "?" + params.Encode()

	type result struct {
		token string
		err   error
	}
	ch := make(chan result, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			ch <- result{err: fmt.Errorf("state mismatch")}
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}
		if errParam := r.URL.Query().Get("error"); errParam != "" {
			ch <- result{err: fmt.Errorf("OAuth error: %s", errParam)}
			fmt.Fprintf(w, "Authentication failed: %s. You can close this window.", errParam)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			ch <- result{err: fmt.Errorf("no code in callback")}
			http.Error(w, "Missing code", http.StatusBadRequest)
			return
		}

		token, err := exchangeCode(clientID, clientSecret, code, redirectURI)
		if err != nil {
			ch <- result{err: err}
			fmt.Fprintf(w, "Authentication failed: %s. You can close this window.", err)
			return
		}

		ch <- result{token: token}
		fmt.Fprint(w, "Authentication successful! You can close this window.")
	})

	server := &http.Server{Handler: mux}
	go server.Serve(listener)
	defer server.Shutdown(context.Background())

	fmt.Fprintf(os.Stderr, "Opening browser for Slack authentication...\n")
	fmt.Fprintf(os.Stderr, "If the browser doesn't open, visit:\n%s\n", authURL)
	openBrowser(authURL)

	res := <-ch
	return res.token, res.err
}

func exchangeCode(clientID, clientSecret, code, redirectURI string) (string, error) {
	data := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
	}

	resp, err := http.PostForm(slackTokenURL, data)
	if err != nil {
		return "", fmt.Errorf("exchanging code: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp struct {
		OK         bool   `json:"ok"`
		Error      string `json:"error"`
		AuthedUser struct {
			AccessToken string `json:"access_token"`
		} `json:"authed_user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("decoding token response: %w", err)
	}
	if !tokenResp.OK {
		return "", fmt.Errorf("token exchange failed: %s", tokenResp.Error)
	}
	if tokenResp.AuthedUser.AccessToken == "" {
		return "", fmt.Errorf("no user access token in response")
	}
	return tokenResp.AuthedUser.AccessToken, nil
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Run() // best-effort; user can copy the URL from stderr
}
