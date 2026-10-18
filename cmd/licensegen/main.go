// Command licensegen is the vendor-side tool: it creates the signing key pair
// and issues license keys. It is not shipped to customers.
//
//	licensegen keygen                      → prints private + public key (base64url)
//	licensegen issue -key <private> -org "ACME" -seats 50 -days 365
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/b1ll1eee/messario/internal/license"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "keygen":
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			fail(err)
		}
		fmt.Println("private (keep secret):", base64.RawURLEncoding.EncodeToString(priv))
		fmt.Println("public  (MESSARIO_LICENSE_PUBLIC_KEY):", base64.RawURLEncoding.EncodeToString(pub))
	case "issue":
		fs := flag.NewFlagSet("issue", flag.ExitOnError)
		keyB64 := fs.String("key", os.Getenv("LICENSE_PRIVATE_KEY"), "private key (base64url) or LICENSE_PRIVATE_KEY")
		org := fs.String("org", "", "customer organization name")
		seats := fs.Int("seats", 0, "seat count (0 = unlimited)")
		days := fs.Int("days", 0, "validity in days (0 = perpetual)")
		_ = fs.Parse(os.Args[2:])
		raw, err := base64.RawURLEncoding.DecodeString(*keyB64)
		if err != nil || len(raw) != ed25519.PrivateKeySize {
			fail(fmt.Errorf("bad private key"))
		}
		if *org == "" {
			fail(fmt.Errorf("-org is required"))
		}
		p := license.Payload{Org: *org, Seats: *seats, IssuedAt: time.Now().UTC()}
		if *days > 0 {
			p.Expires = p.IssuedAt.AddDate(0, 0, *days)
		}
		key, err := license.Sign(p, ed25519.PrivateKey(raw))
		if err != nil {
			fail(err)
		}
		fmt.Println(key)
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: licensegen keygen | issue -key <private> -org <name> [-seats N] [-days N]")
	os.Exit(2)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
