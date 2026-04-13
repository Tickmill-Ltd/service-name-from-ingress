# Security Proof-of-Concept — Repo Jacking [By -  init2wi nit]

This repository was defensively claimed during an authorized Bugcrowd bug bounty engagement for Tickmill.

## Vulnerability

The Go module `github.com/Tickmill-Ltd/service-name-from-ingress` was cached on the Go module proxy (proxy.golang.org) as v0.1.0, but the GitHub organization `Tickmill-Ltd` was deleted/renamed. This allowed anyone to register the org name and publish a malicious module version.

## Evidence

- Go proxy cache: `https://proxy.golang.org/github.com/!tickmill-!ltd/service-name-from-ingress/@latest`
- Original source reference: `https://github.com/tmill-app/service-name-from-ingress/blob/master/go.mod`
- This module is deployed in Tickmill's Kubernetes cluster as a production utility

## Status

Defensively claimed. No malicious code. Reported via Bugcrowd.
