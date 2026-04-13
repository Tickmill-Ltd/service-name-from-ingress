package main

// SECURITY PROOF-OF-CONCEPT
// This repository was claimed as part of an authorized Bugcrowd bug bounty engagement
// for Tickmill (https://bugcrowd.com/tickmill).
//
// The original GitHub organization "Tickmill-Ltd" was deleted/renamed, leaving the
// Go module path github.com/Tickmill-Ltd/service-name-from-ingress claimable.
// The Go module proxy (proxy.golang.org) has v0.1.0 cached from the original org.
//
// This proves that an attacker could publish a newer version (v0.2.0+) containing
// malicious code that would be pulled by any Go build resolving this module.
//
// No malicious code is included. This is a defensive claim to prevent real attackers
// from exploiting this supply chain vulnerability.
//
// Reporter: [YOUR BUGCROWD HANDLE]
// Program: Tickmill Managed Bug Bounty Engagement

func main() {
	println("SECURITY POC: This module was claimed during authorized bug bounty testing.")
	println("See: https://bugcrowd.com/tickmill")
}
