// Command cdadmin manages shops and staff until self-serve signup exists.
//
// Nobody hands out PINs: every command that creates or resets an account prints a one-time
// setup link. The person opens it and chooses their own PIN. Links expire (CD_SETUP_LINK_TTL, 48h).
//
//	cdadmin create-table                      (once per environment, if CDK didn't create the table)
//	cdadmin create-shop  -slug imran-xerox -name "Imran Xerox" -address "Station Rd, Pune" -owner Imran
//	cdadmin add-staff    -shop imran-xerox -name Sana [-role staff|owner]
//	cdadmin reset-pin    -shop imran-xerox -name Sana
//	cdadmin remove-staff -shop imran-xerox -name Sana
//	cdadmin list-staff   -shop imran-xerox
//
// It talks to the DynamoDB table (CD_DYNAMODB_TABLE, CD_DYNAMODB_REGION; CD_DYNAMODB_ENDPOINT for DynamoDB Local)
// with your AWS credentials.
// Set CD_PUBLIC_WEB_URL to the address shops open (e.g. https://counterdrop.in) so links are correct.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"counter-drop/api/internal/backend"
	"counter-drop/api/internal/config"
	"counter-drop/api/internal/domain"
	"counter-drop/api/internal/store"
	"counter-drop/api/internal/store/ddbstore"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	cfg := config.Load()
	ctx := context.Background()
	if os.Args[1] == "create-table" {
		createTable(ctx, cfg)
		return
	}
	st, err := backend.Open(ctx, cfg, domain.DefaultPolicy())
	if err != nil {
		fail(err.Error())
	}
	defer st.Close()

	cmd, args := os.Args[1], os.Args[2:]
	switch cmd {
	case "create-shop":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		slug := fs.String("slug", "", "link name, e.g. imran-xerox (printed on the QR: /s/<slug>; can't change later)")
		name := fs.String("name", "", "shop name")
		address := fs.String("address", "", "address or landmark shown to customers")
		owner := fs.String("owner", "", "owner's first name (shown on the sign-in screen)")
		_ = fs.Parse(args)
		if *owner == "" {
			fail("-owner is required")
		}
		sh, err := st.CreateShop(ctx, store.CreateShopInput{Slug: *slug, Name: *name, Address: *address, OwnerName: *owner})
		if err != nil {
			fail(err.Error())
		}
		ownerID, err := st.OwnerID(ctx, sh.ID)
		if err != nil {
			fail(err.Error())
		}
		link, err := st.IssueSetupLink(ctx, sh.ID, ownerID, "setup", "cdadmin", cfg.SetupLinkTTL)
		if err != nil {
			fail(err.Error())
		}
		fmt.Printf("Created shop %q\n  Customer QR page: %s/s/%s\n\n", sh.Name, cfg.PublicWebURL, sh.Slug)
		printLink(cfg, sh.Name, *owner, "owner", link)
	case "add-staff":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		slug := fs.String("shop", "", "shop link name")
		name := fs.String("name", "", "person's first name")
		role := fs.String("role", "staff", "staff | owner")
		_ = fs.Parse(args)
		sh := shop(ctx, st, *slug)
		m, link, err := st.InviteStaff(ctx, sh.ID, *name, *role, "cdadmin", cfg.SetupLinkTTL)
		if err != nil {
			fail(err.Error())
		}
		printLink(cfg, sh.Name, m.Name, m.Role, link)
	case "reset-pin":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		slug := fs.String("shop", "", "shop link name")
		name := fs.String("name", "", "person's name")
		_ = fs.Parse(args)
		sh := shop(ctx, st, *slug)
		m, err := st.StaffByName(ctx, sh.ID, *name)
		if err != nil {
			fail(fmt.Sprintf("no current staff called %q at %s", *name, sh.Slug))
		}
		purpose := "reset"
		if m.Pending {
			purpose = "setup"
		}
		link, err := st.IssueSetupLink(ctx, sh.ID, m.ID, purpose, "cdadmin", cfg.SetupLinkTTL)
		if err != nil {
			fail(err.Error())
		}
		if purpose == "reset" {
			fmt.Printf("%s's old PIN no longer works and they were signed out everywhere.\n\n", m.Name)
		}
		printLink(cfg, sh.Name, m.Name, m.Role, link)
	case "remove-staff":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		slug := fs.String("shop", "", "shop link name")
		name := fs.String("name", "", "person's name")
		_ = fs.Parse(args)
		sh := shop(ctx, st, *slug)
		m, err := st.StaffByName(ctx, sh.ID, *name)
		if err != nil {
			fail(fmt.Sprintf("no current staff called %q at %s", *name, sh.Slug))
		}
		if err := st.RemoveStaff(ctx, sh.ID, m.ID, "cdadmin"); err != nil {
			fail(err.Error())
		}
		fmt.Printf("Removed %s from %s. They are signed out and can't sign in again.\n", m.Name, sh.Slug)
	case "list-staff":
		fs := flag.NewFlagSet(cmd, flag.ExitOnError)
		slug := fs.String("shop", "", "shop link name")
		_ = fs.Parse(args)
		sh := shop(ctx, st, *slug)
		list, err := st.ListStaff(ctx, sh.ID)
		if err != nil {
			fail(err.Error())
		}
		fmt.Printf("%s (%s)\n", sh.Name, sh.Slug)
		for _, m := range list {
			status := "active"
			if m.Pending {
				status = "waiting for setup link"
			}
			fmt.Printf("  %-20s %-6s %s\n", m.Name, m.Role, status)
		}
	default:
		usage()
	}
}

// createTable creates the DynamoDB table with its indexes, TTL and point-in-time recovery.
func createTable(ctx context.Context, cfg config.Config) {
	ds, err := ddbstore.New(ctx, ddbstore.Config{Table: cfg.Dynamo.Table, Region: cfg.Dynamo.Region, Endpoint: cfg.Dynamo.Endpoint}, domain.DefaultPolicy())
	if err != nil {
		fail(err.Error())
	}
	if err := ds.EnsureTable(ctx); err != nil {
		fail(err.Error())
	}
	if err := ds.Configure(ctx); err != nil {
		fail(err.Error())
	}
	fmt.Printf("Table %s is ready in %s (4 indexes, TTL on \"ttl\", point-in-time recovery %d days).\n", cfg.Dynamo.Table, cfg.Dynamo.Region, ddbstore.PITRDays)
}

func shop(ctx context.Context, st store.Repository, slug string) domain.Shop {
	sh, err := st.GetShopBySlug(ctx, slug)
	if err != nil {
		fail("shop not found: " + slug)
	}
	return sh
}

// printLink prints the link and a ready-to-send message. Send it only to that person: until it's used,
// the link is as good as their PIN.
func printLink(cfg config.Config, shopName, person, role string, link store.SetupLink) {
	url := cfg.SetupURL(link.Token)
	ist := time.FixedZone("IST", 5*3600+1800)
	expires := link.ExpiresAt.In(ist).Format("2 Jan, 3:04 pm")
	verb := "set up your"
	if link.Purpose == "reset" {
		verb = "choose a new"
	}
	fmt.Printf("One-time link for %s (%s), valid until %s IST:\n  %s\n\n", person, role, expires, url)
	fmt.Println("Message to send (only to " + person + "):")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("Hi %s, open this link on the phone or computer you'll use at the counter to %s Counter Drop PIN for %s:\n%s\nIt works once and expires on %s. Don't share it.\n", person, verb, shopName, url, expires)
	fmt.Println(strings.Repeat("-", 60))
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  cdadmin create-table
  cdadmin create-shop  -slug S -name N -owner O [-address A]
  cdadmin add-staff    -shop S -name N [-role staff|owner]
  cdadmin reset-pin    -shop S -name N
  cdadmin remove-staff -shop S -name N
  cdadmin list-staff   -shop S`)
	os.Exit(2)
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, "error:", msg)
	os.Exit(1)
}
