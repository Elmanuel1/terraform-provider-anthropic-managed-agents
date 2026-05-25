// local-oidc is a minimal OIDC provider for local WIF testing.
//
// Start the server, then curl /mint to get a token signed with the same key:
//
//	go run ./cmd/local-oidc --issuer https://<ngrok-url>
//	export TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC=$(curl -s 'http://localhost:8080/mint?sub=local:test&aud=https://api.anthropic.com')
//	terraform apply
package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	issuer = flag.String("issuer", "", "Public issuer URL (e.g. https://abc123.ngrok.io) — required")
	port   = flag.Int("port", 8080, "Port to listen on")
)

func main() {
	flag.Parse()
	if *issuer == "" {
		log.Fatal("--issuer is required (e.g. https://abc123.ngrok.io)")
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("generating RSA key: %v", err)
	}

	serve(key, *issuer, *port)
}

// serve starts the OIDC discovery, JWKS, and mint endpoints.
func serve(key *rsa.PrivateKey, issuer string, port int) {
	kid := keyID(key)

	http.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"issuer":   issuer,
			"jwks_uri": issuer + "/jwks",
		})
	})

	http.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		pub := key.Public().(*rsa.PublicKey)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]any{
				{
					"kty": "RSA",
					"use": "sig",
					"alg": "RS256",
					"kid": kid,
					"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
				},
			},
		})
	})

	// /mint returns a signed JWT. Query params: sub, aud, ttl (e.g. 10m).
	http.HandleFunc("/mint", func(w http.ResponseWriter, r *http.Request) {
		sub := r.URL.Query().Get("sub")
		if sub == "" {
			sub = "local:test"
		}
		aud := r.URL.Query().Get("aud")
		if aud == "" {
			aud = "https://api.anthropic.com"
		}
		ttl := 10 * time.Minute
		if s := r.URL.Query().Get("ttl"); s != "" {
			if d, err := time.ParseDuration(s); err == nil {
				ttl = d
			}
		}

		token, err := mintToken(key, issuer, sub, aud, ttl)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, token)
	})

	addr := fmt.Sprintf(":%d", port)
	log.Printf("local-oidc listening on %s", addr)
	log.Printf("issuer:    %s", issuer)
	log.Printf("jwks_uri:  %s/jwks", issuer)
	log.Printf("discovery: %s/.well-known/openid-configuration", issuer)
	log.Printf("")
	log.Printf("Mint a token:")
	log.Printf(`  export TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC=$(curl -s 'http://localhost:%d/mint?sub=local:test&aud=https://api.anthropic.com')`, port)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func mintToken(key *rsa.PrivateKey, issuer, sub, aud string, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": issuer,
		"sub": sub,
		"aud": aud,
		"iat": now.Unix(),
		"exp": now.Add(ttl).Unix(),
		"jti": randomID(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = keyID(key)
	return tok.SignedString(key)
}

func keyID(key *rsa.PrivateKey) string {
	pub := key.Public().(*rsa.PublicKey)
	return fmt.Sprintf("local-%x", pub.N.Bytes()[:4])
}

func randomID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
