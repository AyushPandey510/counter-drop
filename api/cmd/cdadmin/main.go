// Command cdadmin manages shops and staff until the self-serve signup (build-plan step 20) exists.
//
//	go run ./cmd/cdadmin create-shop -slug imran-xerox -name "Imran Xerox" -owner Imran -pin 4821
//	go run ./cmd/cdadmin add-staff -shop imran-xerox -name Kavita -pin 1111
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"counter-drop/api/internal/config"
	"counter-drop/api/internal/domain"
	"counter-drop/api/internal/store"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		fail("CD_DATABASE_URL is required")
	}
	ctx := context.Background()
	st, err := store.New(ctx, cfg.DatabaseURL, domain.DefaultPolicy())
	if err != nil {
		fail(err.Error())
	}
	defer st.Close()
	if err := st.ApplyMigrations(ctx, cfg.MigrationsDir); err != nil {
		fail(err.Error())
	}

	switch os.Args[1] {
	case "create-shop":
		fs := flag.NewFlagSet("create-shop", flag.ExitOnError)
		slug := fs.String("slug", "", "link name, e.g. imran-xerox (used in the QR: /s/<slug>)")
		name := fs.String("name", "", "shop name")
		address := fs.String("address", "", "address or landmark")
		owner := fs.String("owner", "Owner", "owner's name for the login screen")
		pin := fs.String("pin", "", "owner's 4-digit PIN")
		_ = fs.Parse(os.Args[2:])
		sh, err := st.CreateShop(ctx, store.CreateShopInput{Slug: *slug, Name: *name, Address: *address, OwnerName: *owner, OwnerPIN: *pin})
		if err != nil {
			fail(err.Error())
		}
		fmt.Printf("created shop %q (%s)\n  drop page: /s/%s\n  staff login: /shop/login?shop=%s as %q\n", sh.Name, sh.ID, sh.Slug, sh.Slug, *owner)
	case "add-staff":
		fs := flag.NewFlagSet("add-staff", flag.ExitOnError)
		slug := fs.String("shop", "", "shop link name")
		name := fs.String("name", "", "staff name")
		pin := fs.String("pin", "", "4-digit PIN")
		role := fs.String("role", "staff", "staff | owner")
		_ = fs.Parse(os.Args[2:])
		sh, err := st.GetShopBySlug(ctx, *slug)
		if err != nil {
			fail("shop not found: " + *slug)
		}
		if *role != "staff" && *role != "owner" {
			fail("role must be staff or owner")
		}
		if err := st.AddStaff(ctx, sh.ID, *name, *role, *pin); err != nil {
			fail(err.Error())
		}
		fmt.Printf("added %s %q to %s\n", *role, *name, sh.Slug)
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: cdadmin create-shop -slug S -name N -owner O -pin 1234 | add-staff -shop S -name N -pin 1234 [-role staff|owner]")
	os.Exit(2)
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "error:", msg)
	os.Exit(1)
}
