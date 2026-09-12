package service

import (
	"context"
	"fmt"
	"strings"

	oidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// LocalOIDCAuthProviderConfig configures the local/CI-only authorization-code client.
type LocalOIDCAuthProviderConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
}

// LocalOIDCAuthProvider uses discovery plus verified ID tokens; it is not a GitHub test-double adapter.
type LocalOIDCAuthProvider struct {
	oauthConfig *oauth2.Config
	verifier    OIDCIdentityVerifier
}

func NewLocalOIDCAuthProvider(ctx context.Context, cfg LocalOIDCAuthProviderConfig) (*LocalOIDCAuthProvider, error) {
	issuer := strings.TrimSpace(cfg.Issuer)
	if issuer == "" || strings.TrimSpace(cfg.ClientID) == "" || strings.TrimSpace(cfg.ClientSecret) == "" {
		return nil, fmt.Errorf("service: local oidc issuer, client id, and client secret are required")
	}
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("service: discover local oidc provider: %w", err)
	}
	return &LocalOIDCAuthProvider{
		oauthConfig: &oauth2.Config{ClientID: cfg.ClientID, ClientSecret: cfg.ClientSecret, Endpoint: provider.Endpoint(), Scopes: []string{oidc.ScopeOpenID, "profile", "email"}},
		verifier:    NewCoreOIDCIdentityVerifier(provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})),
	}, nil
}

func (p *LocalOIDCAuthProvider) AuthorizationURL(redirectURI string, state string) string {
	return p.oauthConfig.AuthCodeURL(state, oauth2.SetAuthURLParam("redirect_uri", strings.TrimSpace(redirectURI)))
}

func (p *LocalOIDCAuthProvider) ExchangeIdentity(ctx context.Context, code string, redirectURI string) (AuthIdentity, error) {
	token, err := p.oauthConfig.Exchange(ctx, strings.TrimSpace(code), oauth2.SetAuthURLParam("redirect_uri", strings.TrimSpace(redirectURI)))
	if err != nil {
		return AuthIdentity{}, err
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || strings.TrimSpace(rawIDToken) == "" {
		return AuthIdentity{}, fmt.Errorf("service: local oidc token response omitted id_token")
	}
	return normalizedOIDCIdentity(ctx, p.verifier, authProviderLocalOIDC, rawIDToken)
}

// OIDCVerifiedClaims is the subset of an OIDC verified token needed for normalized identity extraction.
type OIDCVerifiedClaims interface {
	Claims(any) error
}

// OIDCIdentityVerifier verifies a raw ID token and exposes its claims.
type OIDCIdentityVerifier interface {
	Verify(ctx context.Context, rawIDToken string) (OIDCVerifiedClaims, error)
}

// CoreOIDCIdentityVerifier adapts go-oidc's verifier to the repo-local seam.
type CoreOIDCIdentityVerifier struct {
	verifier *oidc.IDTokenVerifier
}

// NewCoreOIDCIdentityVerifier constructs the default go-oidc-backed verifier adapter.
func NewCoreOIDCIdentityVerifier(verifier *oidc.IDTokenVerifier) CoreOIDCIdentityVerifier {
	return CoreOIDCIdentityVerifier{verifier: verifier}
}

// Verify validates the raw ID token and exposes its claims for normalization.
func (v CoreOIDCIdentityVerifier) Verify(ctx context.Context, rawIDToken string) (OIDCVerifiedClaims, error) {
	if v.verifier == nil {
		return nil, fmt.Errorf("service: oidc verifier is required")
	}
	return v.verifier.Verify(ctx, strings.TrimSpace(rawIDToken))
}

type oidcIdentityClaims struct {
	Subject           string `json:"sub"`
	Email             string `json:"email"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
}

func normalizedOIDCIdentity(ctx context.Context, verifier OIDCIdentityVerifier, provider string, rawIDToken string) (AuthIdentity, error) {
	if verifier == nil {
		return AuthIdentity{}, fmt.Errorf("service: oidc verifier is required")
	}
	token, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return AuthIdentity{}, err
	}
	var claims oidcIdentityClaims
	if err := token.Claims(&claims); err != nil {
		return AuthIdentity{}, err
	}
	login := strings.TrimSpace(claims.PreferredUsername)
	if login == "" {
		login = strings.TrimSpace(claims.Name)
	}
	if strings.TrimSpace(provider) == "" || strings.TrimSpace(claims.Subject) == "" || login == "" {
		return AuthIdentity{}, fmt.Errorf("service: oidc identity claims are incomplete")
	}
	return AuthIdentity{
		Provider: strings.TrimSpace(provider),
		Subject:  strings.TrimSpace(claims.Subject),
		Login:    login,
		Email:    strings.TrimSpace(claims.Email),
	}, nil
}
