// Copyright (c) 2025 Binadox (https://binadox.com)
// This software is licensed under the zlib license. See LICENSE file for details.

package dto

// PrincipalKind represents the type of principal identifier
type PrincipalKind string

const (
	KindUsername PrincipalKind = "USERNAME"
	KindIP       PrincipalKind = "IP"
)

// PrincipalDTO identifies the user whose browser history was scanned
type PrincipalDTO struct {
	Name string        `json:"name"`
	Kind PrincipalKind `json:"kind"`
}

// VisitedSite represents a single browser history entry
type VisitedSite struct {
	URL       string `json:"url"`
	Timestamp int64  `json:"timestamp"` // Unix milliseconds
}

// VisitedSitesDTO is the payload sent to the server
type VisitedSitesDTO struct {
	Principal    PrincipalDTO  `json:"principal"`
	VisitedSites []VisitedSite `json:"visitedSites"`
	Source       string        `json:"source"`
}

// NewUserPrincipal creates a PrincipalDTO with USERNAME kind
func NewUserPrincipal(username string) PrincipalDTO {
	return PrincipalDTO{
		Name: username,
		Kind: KindUsername,
	}
}

// NewIPPrincipal creates a PrincipalDTO with IP kind
func NewIPPrincipal(ip string) PrincipalDTO {
	return PrincipalDTO{
		Name: ip,
		Kind: KindIP,
	}
}
