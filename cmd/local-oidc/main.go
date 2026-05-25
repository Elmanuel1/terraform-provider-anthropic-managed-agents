// local-oidc is a minimal OIDC provider for local WIF testing.
//
// It generates an RSA key pair, serves the OIDC discovery document and JWKS
// endpoint, and can mint signed JWTs for use as TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC.
//
// Usage:
//
//	# Step 1 — start the server (keep running)
//	go run ./cmd/local-oidc --issuer https://<ngrok-url>
//
//	# Step 2 — mint a token (in a second terminal)
//	go run ./cmd/local-oidc --issuer https://<ngrok-url> --mint \
//	  --sub "local:test" --aud "https://api.anthropic.com"
//
//	# Step 3 — export and run terraform
//	export TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC=$(go run ./cmd/local-oidc ...)
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
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	issuer = flag.String("issuer", "", "Public issuer URL (e.g. https://abc123.ngrok.io) — required")
	port   = flag.Int("port", 8080, "Port to listen on")
	mint   = flag.Bool("mint", false, "Mint a JWT and print it to stdout instead of serving")
	sub    = flag.String("sub", "local:test:workspace:dev", "JWT subject claim")
	aud    = flag.String("aud", "https://api.anthropic.com", "JWT audience claim")
	ttl    = flag.Duration("ttl", 10*time.Minute, "JWT TTL")
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

	if *mint {
		token, err := mintToken(key, *issuer, *sub, *aud, *ttl)
		if err != nil {
			log.Fatalf("minting token: %v", err)
		}
		fmt.Print(token)
		os.Exit(0)
	}

	serve(key, *issuer, *port)
}

// mintToken creates a signed JWT using the given key.
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

// serve starts the OIDC discovery + JWKS HTTP server.
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

	addr := fmt.Sprintf(":%d", port)
	log.Printf("local-oidc listening on %s", addr)
	log.Printf("issuer:   %s", issuer)
	log.Printf("jwks_uri: %s/jwks", issuer)
	log.Printf("discovery: %s/.well-known/openid-configuration", issuer)
	log.Printf("")
	log.Printf("Mint a token with:")
	log.Printf("  go run ./cmd/local-oidc --issuer %s --mint --sub 'local:test' --aud 'https://api.anthropic.com'", issuer)
	log.Fatal(http.ListenAndServe(addr, nil))
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
