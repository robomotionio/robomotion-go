package runtime

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/pkg/browser"
	"golang.org/x/oauth2"
)

const (
	// OAuth2RedirectURL is the fixed callback URL - must match what's configured in OAuth app
	OAuth2RedirectURL = "http://localhost:9876/oauth2/callback"
	// OAuth2CallbackPort is the port for the OAuth callback server
	OAuth2CallbackPort = "9876"

	// OAuth2PortRetryInterval is how long to wait between port binding attempts
	OAuth2PortRetryInterval = 5 * time.Second
	// OAuth2PortMaxTimeout is the maximum time to wait for port to become available
	OAuth2PortMaxTimeout = 300 * time.Second
	// OAuth2AuthTimeout is the maximum time to wait for user to complete authorization
	OAuth2AuthTimeout = 5 * time.Minute
)

var (
	oauthMutex          sync.Mutex
	activeOAuthServer   *http.Server
	activeOAuthListener net.Listener
)

// OpenOAuthDialog opens a browser for OAuth2 authorization and waits for the callback.
// It implements a retry mechanism for port binding: if the port is busy (e.g., another
// OAuth flow is in progress), it will retry every 5 seconds up to 300 seconds total.
// The server is always stopped after the OAuth flow completes (success or failure).
//
// Usage:
//
//	config := &oauth2.Config{
//	    ClientID:     "your-client-id",
//	    ClientSecret: "your-client-secret",
//	    Scopes:       []string{"scope1", "scope2"},
//	    Endpoint:     google.Endpoint,
//	    RedirectURL:  runtime.OAuth2RedirectURL,
//	}
//	code, err := runtime.OpenOAuthDialog(config)
//	if err != nil {
//	    return err
//	}
//	token, err := config.Exchange(context.Background(), code)
func OpenOAuthDialog(cfg *oauth2.Config) (string, error) {
	// Ensure redirect URL is set correctly
	cfg.RedirectURL = OAuth2RedirectURL

	// Try to bind the port with retry logic
	ln, err := acquireOAuthPort()
	if err != nil {
		return "", err
	}

	// Ensure we always clean up the server when done
	defer func() {
		oauthMutex.Lock()
		defer oauthMutex.Unlock()

		if activeOAuthServer != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			activeOAuthServer.Shutdown(ctx)
			activeOAuthServer = nil
		}

		if activeOAuthListener != nil {
			activeOAuthListener.Close()
			activeOAuthListener = nil
		}
	}()

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errMsg := r.URL.Query().Get("error")
			errDesc := r.URL.Query().Get("error_description")
			if errMsg == "" {
				errMsg = "no authorization code received"
			}
			if errDesc != "" {
				errMsg = fmt.Sprintf("%s: %s", errMsg, errDesc)
			}
			errCh <- fmt.Errorf("OAuth error: %s", errMsg)
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>Authorization Failed</title>
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #f5f5f5; }
		.container { text-align: center; padding: 40px; background: white; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
		h1 { color: #f44336; margin-bottom: 16px; }
		p { color: #666; }
	</style>
</head>
<body>
	<div class="container">
		<h1>Authorization Failed</h1>
		<p>` + errMsg + `</p>
		<p>You can close this window and try again.</p>
	</div>
</body>
</html>`))
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>Authorization Successful</title>
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #f5f5f5; }
		.container { text-align: center; padding: 40px; background: white; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
		h1 { color: #4CAF50; margin-bottom: 16px; }
		p { color: #666; }
	</style>
</head>
<body>
	<div class="container">
		<h1>Authorization Successful</h1>
		<p>You can close this window and return to the application.</p>
	</div>
</body>
</html>`))

		codeCh <- code
	})

	server := &http.Server{Handler: mux}

	oauthMutex.Lock()
	activeOAuthServer = server
	oauthMutex.Unlock()

	go func() {
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// AccessTypeOffline requests a refresh_token, prompt=consent forces consent screen every time
	// This ensures we always get a refresh_token (Google only returns it on first consent otherwise)
	authURL := cfg.AuthCodeURL("state",
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("prompt", "consent"),
	)
	if err := browser.OpenURL(authURL); err != nil {
		return "", fmt.Errorf("failed to open browser: %w", err)
	}

	select {
	case code := <-codeCh:
		return code, nil
	case err := <-errCh:
		return "", err
	case <-time.After(OAuth2AuthTimeout):
		return "", fmt.Errorf("OAuth authorization timed out after %v waiting for user to complete authorization in browser", OAuth2AuthTimeout)
	}
}

// acquireOAuthPort attempts to bind to the OAuth callback port with retry logic.
// If the port is busy, it retries every OAuth2PortRetryInterval until OAuth2PortMaxTimeout.
func acquireOAuthPort() (net.Listener, error) {
	startTime := time.Now()
	attempt := 0

	for {
		attempt++
		ln, err := net.Listen("tcp", "127.0.0.1:"+OAuth2CallbackPort)
		if err == nil {
			// Successfully bound to port
			oauthMutex.Lock()
			activeOAuthListener = ln
			oauthMutex.Unlock()

			if attempt > 1 {
				log.Printf("OAuth: Successfully acquired port %s after %d attempts (elapsed: %v)",
					OAuth2CallbackPort, attempt, time.Since(startTime).Round(time.Second))
			}
			return ln, nil
		}

		// Check if we've exceeded the maximum timeout
		elapsed := time.Since(startTime)
		if elapsed >= OAuth2PortMaxTimeout {
			return nil, fmt.Errorf("OAuth failed: could not bind to port %s after %v (%d attempts). "+
				"Port is in use by another process. Please wait for the other OAuth flow to complete or restart the application",
				OAuth2CallbackPort, OAuth2PortMaxTimeout, attempt)
		}

		// Log the retry (only on first attempt and then periodically)
		remaining := OAuth2PortMaxTimeout - elapsed
		if attempt == 1 {
			log.Printf("OAuth: Port %s is busy, waiting for it to become available (will retry every %v, timeout in %v)...",
				OAuth2CallbackPort, OAuth2PortRetryInterval, remaining.Round(time.Second))
		} else if attempt%6 == 0 { // Log every 30 seconds (6 * 5s)
			log.Printf("OAuth: Still waiting for port %s... (attempt %d, %v remaining)",
				OAuth2CallbackPort, attempt, remaining.Round(time.Second))
		}

		// Wait before retrying
		time.Sleep(OAuth2PortRetryInterval)
	}
}
