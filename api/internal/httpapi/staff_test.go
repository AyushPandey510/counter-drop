package httpapi_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"counter-drop/api/internal/store"
)

type linkT struct {
	Purpose   string `json:"purpose"`
	SetupPath string `json:"setupPath"`
	SetupURL  string `json:"setupUrl"`
}

func tokenOf(t *testing.T, l linkT) string {
	t.Helper()
	i := strings.Index(l.SetupPath, "#")
	if !strings.HasPrefix(l.SetupPath, "/shop/setup#") || i < 0 || len(l.SetupPath[i+1:]) < 20 {
		t.Fatalf("bad setup path %q", l.SetupPath)
	}
	if !strings.HasSuffix(l.SetupURL, l.SetupPath) {
		t.Fatalf("setupUrl %q doesn't end with path", l.SetupURL)
	}
	return l.SetupPath[i+1:]
}

func (e *env) names() []string {
	e.t.Helper()
	return must[struct {
		Names []string `json:"names"`
	}](e.t, e.do("GET", "/api/v1/cd/shop/staff-names?shop=demo-print", nil, nil), 200).Names
}

func (e *env) setupWith(token, pin string, want int) map[string]string {
	e.t.Helper()
	r := e.do("POST", "/api/v1/cd/shop/setup", map[string]string{"token": token, "pin": pin}, nil)
	if want != 200 {
		if r.Status != want {
			e.t.Fatalf("setup status %d, want %d", r.Status, want)
		}
		return nil
	}
	out := must[struct {
		Token string `json:"token"`
		Staff struct {
			Name string `json:"name"`
		} `json:"staff"`
	}](e.t, r, 200)
	return map[string]string{"Authorization": "Bearer " + out.Token}
}

func TestStaffSetupLinks(t *testing.T) {
	e := setup(t)
	owner := e.login("Owner", "1234")
	kavita := e.login("Kavita", "1111")
	const api = "/api/v1/cd/shop"

	// Only owners manage staff.
	if r := e.do("POST", api+"/staff", map[string]string{"name": "Sana"}, kavita); r.Status != 403 {
		t.Fatalf("staff invite should be 403, got %d", r.Status)
	}

	// Invite: Sana is pending, not on the sign-in tiles, can't log in.
	inv := must[struct {
		Staff store.StaffMember `json:"staff"`
		Link  linkT             `json:"link"`
	}](t, e.do("POST", api+"/staff", map[string]string{"name": "Sana", "role": "staff"}, owner), 201)
	if !inv.Staff.Pending || inv.Link.Purpose != "setup" {
		t.Fatalf("invite: %+v", inv)
	}
	tok := tokenOf(t, inv.Link)
	if slices.Contains(e.names(), "Sana") {
		t.Fatal("pending staff must not appear on sign-in tiles")
	}
	if r := e.do("POST", api+"/login", map[string]string{"shop": "demo-print", "name": "Sana", "pin": "4821"}, nil); r.Status != 401 {
		t.Fatalf("pending login: %d", r.Status)
	}
	// Duplicate name refused.
	if r := e.do("POST", api+"/staff", map[string]string{"name": "sana"}, owner); r.Status != 422 {
		t.Fatalf("duplicate name: %d", r.Status)
	}

	// Link info, weak PIN refused (link still usable), then setup signs Sana in.
	info := must[store.SetupInfo](t, e.do("POST", api+"/setup/info", map[string]string{"token": tok}, nil), 200)
	if info.StaffName != "Sana" || info.ShopSlug != "demo-print" || info.Role != "staff" {
		t.Fatalf("info: %+v", info)
	}
	e.setupWith(tok, "1234", 422)
	e.setupWith(tok, "12a4", 422)
	sana := e.setupWith(tok, "4821", 200)
	must[map[string]any](t, e.do("GET", api+"/me", nil, sana), 200)
	if !slices.Contains(e.names(), "Sana") {
		t.Fatal("Sana should be on the tiles after setup")
	}
	// A link works once.
	e.setupWith(tok, "5190", 410)
	if r := e.do("POST", api+"/setup/info", map[string]string{"token": tok}, nil); r.Status != 410 {
		t.Fatalf("used link info: %d", r.Status)
	}
	e.login("Sana", "4821")

	// Change PIN: needs the current PIN, refuses weak/same, signs out other sessions only.
	other := e.login("Sana", "4821")
	if r := e.do("PUT", api+"/me/pin", map[string]string{"currentPin": "0000", "newPin": "5190"}, sana); r.Status != 401 {
		t.Fatalf("wrong current pin: %d", r.Status)
	}
	if r := e.do("PUT", api+"/me/pin", map[string]string{"currentPin": "4821", "newPin": "4821"}, sana); r.Status != 422 {
		t.Fatalf("same pin: %d", r.Status)
	}
	if r := e.do("PUT", api+"/me/pin", map[string]string{"currentPin": "4821", "newPin": "9999"}, sana); r.Status != 422 {
		t.Fatalf("weak pin: %d", r.Status)
	}
	must[map[string]bool](t, e.do("PUT", api+"/me/pin", map[string]string{"currentPin": "4821", "newPin": "5190"}, sana), 200)
	must[map[string]any](t, e.do("GET", api+"/me", nil, sana), 200)
	if r := e.do("GET", api+"/me", nil, other); r.Status != 401 {
		t.Fatalf("other session should be signed out: %d", r.Status)
	}
	if r := e.do("POST", api+"/login", map[string]string{"shop": "demo-print", "name": "Sana", "pin": "4821"}, nil); r.Status != 401 {
		t.Fatalf("old pin still works: %d", r.Status)
	}
	e.login("Sana", "5190")

	// Reset by owner: old PIN dead, Sana signed out, a newer link cancels the older one.
	list := must[struct {
		Staff []store.StaffMember `json:"staff"`
	}](t, e.do("GET", api+"/staff", nil, owner), 200).Staff
	var sanaID, ownerID string
	for _, m := range list {
		switch m.Name {
		case "Sana":
			sanaID = m.ID
		case "Owner":
			ownerID = m.ID
		}
	}
	first := must[struct {
		Link linkT `json:"link"`
	}](t, e.do("POST", api+"/staff/"+sanaID+"/link", map[string]any{}, owner), 200).Link
	if first.Purpose != "reset" {
		t.Fatalf("purpose %q", first.Purpose)
	}
	if r := e.do("GET", api+"/me", nil, sana); r.Status != 401 {
		t.Fatalf("reset should sign Sana out: %d", r.Status)
	}
	second := must[struct {
		Link linkT `json:"link"`
	}](t, e.do("POST", api+"/staff/"+sanaID+"/link", map[string]any{}, owner), 200).Link
	if second.Purpose != "setup" { // PIN already cleared, so she's pending again
		t.Fatalf("second purpose %q", second.Purpose)
	}
	e.setupWith(tokenOf(t, first), "7302", 410)
	sana = e.setupWith(tokenOf(t, second), "7302", 200)

	// Links expire.
	late := must[struct {
		Link linkT `json:"link"`
	}](t, e.do("POST", api+"/staff/"+sanaID+"/link", map[string]any{}, owner), 200).Link
	e.clock.Advance(49 * time.Hour)
	e.setupWith(tokenOf(t, late), "7302", 410)
	e.clock.Advance(-49 * time.Hour)
	owner = e.login("Owner", "1234")

	// Owner can't reset or remove themselves, and can't remove the last owner.
	if r := e.do("POST", api+"/staff/"+ownerID+"/link", map[string]any{}, owner); r.Status != 409 {
		t.Fatalf("self link: %d", r.Status)
	}
	if r := e.do("DELETE", api+"/staff/"+ownerID, nil, owner); r.Status != 409 {
		t.Fatalf("self remove: %d", r.Status)
	}
	ctx := context.Background()
	sh, _ := e.store.GetShopBySlug(ctx, "demo-print")
	coOwner := must[struct {
		Staff store.StaffMember `json:"staff"`
		Link  linkT             `json:"link"`
	}](t, e.do("POST", api+"/staff", map[string]string{"name": "Ravi", "role": "owner"}, owner), 201)
	ravi := e.setupWith(tokenOf(t, coOwner.Link), "3871", 200)
	must[map[string]bool](t, e.do("DELETE", api+"/staff/"+ownerID, nil, ravi), 200)
	if err := e.store.RemoveStaff(ctx, sh.ID, coOwner.Staff.ID, "cdadmin"); err != store.ErrLastOwner {
		t.Fatalf("last owner removal: %v", err)
	}

	// Removing Kavita signs her out and frees the name for someone new.
	var kavitaID string
	for _, m := range must[struct {
		Staff []store.StaffMember `json:"staff"`
	}](t, e.do("GET", api+"/staff", nil, ravi), 200).Staff {
		if m.Name == "Kavita" {
			kavitaID = m.ID
		}
	}
	must[map[string]bool](t, e.do("DELETE", api+"/staff/"+kavitaID, nil, ravi), 200)
	if r := e.do("GET", api+"/me", nil, kavita); r.Status != 401 {
		t.Fatalf("removed staff still signed in: %d", r.Status)
	}
	if slices.Contains(e.names(), "Kavita") {
		t.Fatal("removed staff still on tiles")
	}
	must[map[string]any](t, e.do("POST", api+"/staff", map[string]string{"name": "Kavita"}, ravi), 201)

	// A staff member from another shop can't touch this shop's people.
	if _, err := e.store.CreateShop(ctx, store.CreateShopInput{Slug: "other-shop", Name: "Other Shop", OwnerName: "Zed", OwnerPIN: "6152"}); err != nil {
		t.Fatal(err)
	}
	zed := must[struct {
		Token string `json:"token"`
	}](t, e.do("POST", api+"/login", map[string]string{"shop": "other-shop", "name": "Zed", "pin": "6152"}, nil), 200)
	zh := map[string]string{"Authorization": "Bearer " + zed.Token}
	if r := e.do("DELETE", api+"/staff/"+sanaID, nil, zh); r.Status != 404 {
		t.Fatalf("cross-shop remove: %d", r.Status)
	}
	if r := e.do("POST", api+"/staff/"+sanaID+"/link", map[string]any{}, zh); r.Status != 404 {
		t.Fatalf("cross-shop link: %d", r.Status)
	}
	// The expired reset link above left Sana signed out and pending until she gets a fresh link.
	if r := e.do("GET", api+"/me", nil, sana); r.Status != 401 {
		t.Fatalf("reset should have signed Sana out: %d", r.Status)
	}
	if slices.Contains(e.names(), "Sana") {
		t.Fatal("Sana is pending again and should not be on the tiles")
	}
}
