package ddbstore

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"counter-drop/api/internal/store"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const maxAttempts = 6

var errBusy = errors.New("ddb: too many concurrent changes, try again")

func (s *Store) newStaff(shopID, name, role, pin string) (staffRec, error) {
	name, hash, err := store.NewStaffRecord(name, role, pin)
	if err != nil {
		return staffRec{}, err
	}
	id := store.NewID("staff")
	now := s.now()
	r := staffRec{PK: "S#" + shopID, SK: staffSK(id), Type: "staff", ID: id, ShopID: shopID, Name: name, Role: role,
		PinHash: hash, Active: true, CreatedAt: now}
	if hash != "" {
		r.PinSetAt = &now
	}
	return r, nil
}

// staffPuts writes a new staff member together with the guard that keeps active names unique.
func (s *Store) staffPuts(r staffRec) []types.TransactWriteItem {
	return []types.TransactWriteItem{
		s.put(nameRec{PK: r.PK, SK: nameSK(r.Name), Type: "staffname", StaffID: r.ID}, "attribute_not_exists(PK)", nil, nil),
		s.put(r, "attribute_not_exists(PK)", nil, nil),
	}
}

// ownerDelta returns the shop update that keeps ownerCount right. When removing an owner it also
// refuses to go below one (the last-owner rule), atomically with the removal.
func (s *Store) ownerDelta(shopID string, delta int64) types.TransactWriteItem {
	u := &types.Update{
		TableName: &s.table, Key: key("S#"+shopID, "PROFILE"),
		UpdateExpression:          aws.String("ADD #oc :d"),
		ExpressionAttributeNames:  map[string]string{"#oc": "OwnerCount"},
		ExpressionAttributeValues: map[string]types.AttributeValue{":d": nv(delta)},
		ConditionExpression:       aws.String("attribute_exists(PK)"),
	}
	if delta < 0 {
		u.ConditionExpression = aws.String("#oc > :one")
		u.ExpressionAttributeValues[":one"] = nv(1)
	}
	return types.TransactWriteItem{Update: u}
}

// addStaff writes a new member (and bumps the owner count for owners). The name guard's condition
// is item 0, so a conflict there means the name is taken.
func (s *Store) addStaff(ctx context.Context, r staffRec, extra ...types.TransactWriteItem) error {
	items := s.staffPuts(r)
	if r.Role == "owner" {
		items = append(items, s.ownerDelta(r.ShopID, 1))
	}
	items = append(items, extra...)
	i, err := s.transact(ctx, items...)
	if errors.Is(err, errConflict) && i == 0 {
		return store.ErrNameTaken(r.Name)
	}
	if errors.Is(err, errConflict) {
		return store.ErrNotFound // shop missing
	}
	return err
}

func (s *Store) AddStaff(ctx context.Context, shopID, name, role, pin string) error {
	r, err := s.newStaff(shopID, name, role, pin)
	if err != nil {
		return err
	}
	return s.addStaff(ctx, r)
}

func (s *Store) allStaff(ctx context.Context, shopID string) ([]staffRec, error) {
	var out []staffRec
	err := s.query(ctx, queryOpts{pk: "S#" + shopID, skOp: "begins_with", sk: "STAFF#", consistent: true}, func(item map[string]types.AttributeValue) (bool, error) {
		var r staffRec
		if err := attributevalue.UnmarshalMap(item, &r); err != nil {
			return false, err
		}
		out = append(out, r)
		return true, nil
	})
	return out, err
}

func activeSorted(list []staffRec, pinSetOnly bool) []staffRec {
	var out []staffRec
	for _, r := range list {
		if r.Active && (!pinSetOnly || r.PinHash != "") {
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(a, b int) bool {
		if (out[a].Role == "owner") != (out[b].Role == "owner") {
			return out[a].Role == "owner"
		}
		return strings.ToLower(out[a].Name) < strings.ToLower(out[b].Name)
	})
	return out
}

func member(r staffRec) store.StaffMember {
	return store.StaffMember{ID: r.ID, Name: r.Name, Role: r.Role, Pending: r.PinHash == "", PinSetAt: r.PinSetAt, CreatedAt: r.CreatedAt}
}

func (s *Store) StaffNames(ctx context.Context, slug string) ([]string, error) {
	sh, err := s.GetShopBySlug(ctx, slug)
	if errors.Is(err, store.ErrNotFound) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	list, err := s.allStaff(ctx, sh.ID)
	if err != nil {
		return nil, err
	}
	names := []string{}
	for _, r := range activeSorted(list, true) {
		names = append(names, r.Name)
	}
	return names, nil
}

func (s *Store) ListStaff(ctx context.Context, shopID string) ([]store.StaffMember, error) {
	list, err := s.allStaff(ctx, shopID)
	if err != nil {
		return nil, err
	}
	out := []store.StaffMember{}
	for _, r := range activeSorted(list, false) {
		out = append(out, member(r))
	}
	return out, nil
}

func (s *Store) StaffByName(ctx context.Context, shopID, name string) (store.StaffMember, error) {
	list, err := s.allStaff(ctx, shopID)
	if err != nil {
		return store.StaffMember{}, err
	}
	for _, r := range list {
		if r.Active && strings.EqualFold(r.Name, strings.TrimSpace(name)) {
			return member(r), nil
		}
	}
	return store.StaffMember{}, store.ErrNotFound
}

func (s *Store) OwnerID(ctx context.Context, shopID string) (string, error) {
	list, err := s.allStaff(ctx, shopID)
	if err != nil {
		return "", err
	}
	var best *staffRec
	for i := range list {
		r := &list[i]
		if r.Active && r.Role == "owner" && (best == nil || r.CreatedAt.Before(best.CreatedAt)) {
			best = r
		}
	}
	if best == nil {
		return "", store.ErrNotFound
	}
	return best.ID, nil
}

func (s *Store) staff(ctx context.Context, shopID, staffID string) (staffRec, error) {
	var r staffRec
	err := s.get(ctx, "S#"+shopID, staffSK(staffID), &r)
	return r, err
}

// --- sign-in ---------------------------------------------------------------------------------

func (s *Store) Login(ctx context.Context, slug, name, pin string, ttl time.Duration) (string, store.Principal, error) {
	sh, err := s.GetShopBySlug(ctx, slug)
	if errors.Is(err, store.ErrNotFound) {
		return "", store.Principal{}, store.ErrBadPIN
	}
	if err != nil {
		return "", store.Principal{}, err
	}
	list, err := s.allStaff(ctx, sh.ID)
	if err != nil {
		return "", store.Principal{}, err
	}
	var st *staffRec
	for i := range list {
		if list[i].Active && list[i].Name == name {
			st = &list[i]
		}
	}
	if st == nil {
		return "", store.Principal{}, store.ErrBadPIN
	}
	if err := s.checkPIN(ctx, *st, pin); err != nil {
		return "", store.Principal{}, err
	}
	return s.startSession(ctx, *st, ttl)
}

// checkPIN applies the lockout rule: 5 wrong tries lock the account for 15 minutes.
// Counters are updated with plain writes (last writer wins); an exact count is not needed for a lockout.
func (s *Store) checkPIN(ctx context.Context, st staffRec, pin string) error {
	now := s.now()
	if !st.Active {
		return store.ErrBadPIN
	}
	if st.PinHash == "" {
		return store.ErrSetupPending
	}
	if st.LockedUntil != nil && now.Before(*st.LockedUntil) {
		return store.ErrLocked
	}
	set := func(failed int, lock *time.Time) error {
		expr := "SET #f = :f REMOVE #l"
		values := map[string]types.AttributeValue{":f": nv(int64(failed))}
		if lock != nil {
			expr = "SET #f = :f, #l = :l"
			values[":l"] = sv(lock.UTC().Format(time.RFC3339Nano))
		}
		_, err := s.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
			TableName: &s.table, Key: key(st.PK, st.SK), UpdateExpression: aws.String(expr),
			ExpressionAttributeNames:  map[string]string{"#f": "FailedAttempts", "#l": "LockedUntil"},
			ExpressionAttributeValues: values,
		})
		return err
	}
	if !store.VerifyPIN(pin, st.PinHash) {
		failed := st.FailedAttempts + 1
		var lock *time.Time
		if failed >= store.MaxPINAttempts {
			t := now.Add(store.LockoutFor)
			lock, failed = &t, 0
		}
		_ = set(failed, lock)
		if lock != nil {
			return store.ErrLocked
		}
		return store.ErrBadPIN
	}
	if st.FailedAttempts == 0 && st.LockedUntil == nil {
		return nil
	}
	return set(0, nil)
}

func (s *Store) sessionPut(st staffRec, epoch int64, ttl time.Duration) (string, store.Principal, types.TransactWriteItem) {
	token := store.NewSecret()
	now := s.now()
	exp := now.Add(ttl)
	rec := sessionRec{PK: "SESS#" + store.HashSecret(token), SK: "SESS", Type: "session", StaffID: st.ID, ShopID: st.ShopID,
		Epoch: epoch, ExpiresAt: exp, TTL: ttlAfter(exp, 24*time.Hour)}
	p := store.Principal{Staff: store.Staff{ID: st.ID, ShopID: st.ShopID, Name: st.Name, Role: st.Role}, ExpiresAt: exp}
	return token, p, s.put(rec, "", nil, nil)
}

func (s *Store) startSession(ctx context.Context, st staffRec, ttl time.Duration) (string, store.Principal, error) {
	token, p, item := s.sessionPut(st, st.SessEpoch, ttl)
	if _, err := s.transact(ctx, item); err != nil {
		return "", store.Principal{}, err
	}
	return token, p, nil
}

func (s *Store) Session(ctx context.Context, token string) (store.Principal, error) {
	if token == "" {
		return store.Principal{}, store.ErrUnauthorized
	}
	var se sessionRec
	if err := s.get(ctx, "SESS#"+store.HashSecret(token), "SESS", &se); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return store.Principal{}, store.ErrUnauthorized
		}
		return store.Principal{}, err
	}
	if se.RevokedAt != nil || !se.ExpiresAt.After(s.now()) {
		return store.Principal{}, store.ErrUnauthorized
	}
	st, err := s.staff(ctx, se.ShopID, se.StaffID)
	if errors.Is(err, store.ErrNotFound) || (err == nil && (!st.Active || st.SessEpoch != se.Epoch)) {
		return store.Principal{}, store.ErrUnauthorized
	}
	if err != nil {
		return store.Principal{}, err
	}
	return store.Principal{Staff: store.Staff{ID: st.ID, ShopID: st.ShopID, Name: st.Name, Role: st.Role}, ExpiresAt: se.ExpiresAt}, nil
}

func (s *Store) Logout(ctx context.Context, token string) error {
	_, err := s.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &s.table, Key: key("SESS#"+store.HashSecret(token), "SESS"),
		UpdateExpression:          aws.String("SET #r = :r"),
		ConditionExpression:       aws.String("attribute_exists(PK)"),
		ExpressionAttributeNames:  map[string]string{"#r": "RevokedAt"},
		ExpressionAttributeValues: map[string]types.AttributeValue{":r": sv(s.now().Format(time.RFC3339Nano))},
	})
	var ccf *types.ConditionalCheckFailedException
	if errors.As(err, &ccf) {
		return nil
	}
	return err
}

// --- setup links -----------------------------------------------------------------------------

// staffUpdate replaces a staff item if nobody changed it since it was read (compare-and-swap on the
// fields that matter: the epochs and the PIN hash).
func (s *Store) staffSwap(old, updated staffRec) types.TransactWriteItem {
	return s.put(updated, "#se = :se AND #le = :le AND #ph = :ph AND #a = :a",
		map[string]string{"#se": "SessEpoch", "#le": "LinkEpoch", "#ph": "PinHash", "#a": "Active"},
		map[string]types.AttributeValue{":se": nv(old.SessEpoch), ":le": nv(old.LinkEpoch), ":ph": sv(old.PinHash), ":a": &types.AttributeValueMemberBOOL{Value: old.Active}})
}

func (s *Store) linkPut(st staffRec, epoch int64, purpose, createdBy string, ttl time.Duration) (store.SetupLink, types.TransactWriteItem) {
	now := s.now()
	link := store.SetupLink{Token: store.NewLinkToken(), Purpose: purpose, ExpiresAt: now.Add(ttl)}
	rec := linkRec{PK: "LINK#" + store.HashSecret(link.Token), SK: "LINK", Type: "setuplink", StaffID: st.ID, ShopID: st.ShopID,
		Purpose: purpose, CreatedBy: createdBy, Epoch: epoch, CreatedAt: now, ExpiresAt: link.ExpiresAt, TTL: ttlAfter(link.ExpiresAt, 7*24*time.Hour)}
	return link, s.put(rec, "", nil, nil)
}

func (s *Store) InviteStaff(ctx context.Context, shopID, name, role, createdBy string, ttl time.Duration) (store.StaffMember, store.SetupLink, error) {
	r, err := s.newStaff(shopID, name, role, "")
	if err != nil {
		return store.StaffMember{}, store.SetupLink{}, err
	}
	r.LinkEpoch = 1
	link, item := s.linkPut(r, r.LinkEpoch, "setup", createdBy, ttl)
	if err := s.addStaff(ctx, r, item); err != nil {
		return store.StaffMember{}, store.SetupLink{}, err
	}
	return member(r), link, nil
}

// IssueSetupLink creates a fresh link and cancels earlier unused ones (by bumping the link epoch).
// "reset" also clears the PIN and signs the person out everywhere.
func (s *Store) IssueSetupLink(ctx context.Context, shopID, staffID, purpose, createdBy string, ttl time.Duration) (store.SetupLink, error) {
	if purpose != "setup" && purpose != "reset" {
		return store.SetupLink{}, store.ErrBadPurpose
	}
	for attempt := 0; attempt < maxAttempts; attempt++ {
		st, err := s.staff(ctx, shopID, staffID)
		if err != nil {
			return store.SetupLink{}, err
		}
		if !st.Active {
			return store.SetupLink{}, store.ErrNotFound
		}
		next := st
		next.LinkEpoch++
		if purpose == "reset" && st.PinHash != "" {
			next.PinHash, next.PinSetAt, next.FailedAttempts, next.LockedUntil = "", nil, 0, nil
			next.SessEpoch++
		}
		link, item := s.linkPut(next, next.LinkEpoch, purpose, createdBy, ttl)
		_, err = s.transact(ctx, s.staffSwap(st, next), item)
		if errors.Is(err, errConflict) {
			continue
		}
		if err != nil {
			return store.SetupLink{}, err
		}
		return link, nil
	}
	return store.SetupLink{}, errBusy
}

// validLink returns a usable link and its staff member.
func (s *Store) validLink(ctx context.Context, token string) (linkRec, staffRec, error) {
	var l linkRec
	if err := s.get(ctx, "LINK#"+store.HashSecret(strings.TrimSpace(token)), "LINK", &l); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return l, staffRec{}, store.ErrLinkInvalid
		}
		return l, staffRec{}, err
	}
	if l.Used || !l.ExpiresAt.After(s.now()) {
		return l, staffRec{}, store.ErrLinkInvalid
	}
	st, err := s.staff(ctx, l.ShopID, l.StaffID)
	if errors.Is(err, store.ErrNotFound) || (err == nil && (!st.Active || st.LinkEpoch != l.Epoch)) {
		return l, st, store.ErrLinkInvalid
	}
	return l, st, err
}

func (s *Store) SetupLinkInfo(ctx context.Context, token string) (store.SetupInfo, error) {
	l, st, err := s.validLink(ctx, token)
	if err != nil {
		return store.SetupInfo{}, err
	}
	sh, err := s.GetShopByID(ctx, l.ShopID)
	if err != nil {
		return store.SetupInfo{}, err
	}
	return store.SetupInfo{ShopName: sh.Name, ShopSlug: sh.Slug, StaffName: st.Name, Role: st.Role, Purpose: l.Purpose, ExpiresAt: l.ExpiresAt}, nil
}

// CompleteSetup uses a link once: sets the chosen PIN and signs the person in on this device.
func (s *Store) CompleteSetup(ctx context.Context, token, pin string, sessionTTL time.Duration) (string, store.Principal, error) {
	if err := store.CheckNewPIN(pin); err != nil {
		return "", store.Principal{}, err
	}
	hash, err := store.HashPIN(pin)
	if err != nil {
		return "", store.Principal{}, err
	}
	for attempt := 0; attempt < maxAttempts; attempt++ {
		l, st, err := s.validLink(ctx, token)
		if err != nil {
			return "", store.Principal{}, err
		}
		now := s.now()
		next := st
		next.PinHash, next.PinSetAt, next.FailedAttempts, next.LockedUntil = hash, &now, 0, nil
		used := l
		used.UsedAt, used.Used = &now, true
		session, p, sessItem := s.sessionPut(next, next.SessEpoch, sessionTTL)
		_, err = s.transact(ctx,
			s.put(used, "#u = :f", map[string]string{"#u": "Used"}, map[string]types.AttributeValue{":f": &types.AttributeValueMemberBOOL{Value: false}}),
			s.staffSwap(st, next), sessItem)
		if errors.Is(err, errConflict) {
			continue
		}
		if err != nil {
			return "", store.Principal{}, err
		}
		return session, p, nil
	}
	return "", store.Principal{}, errBusy
}

// ChangePIN sets a new PIN and signs the person out everywhere except this session.
func (s *Store) ChangePIN(ctx context.Context, staffID, currentSession, currentPIN, newPIN string) error {
	if err := store.CheckNewPIN(newPIN); err != nil {
		return err
	}
	if currentPIN == newPIN {
		return store.ErrSamePIN
	}
	var se sessionRec
	if err := s.get(ctx, "SESS#"+store.HashSecret(currentSession), "SESS", &se); err != nil {
		return store.ErrUnauthorized
	}
	st, err := s.staff(ctx, se.ShopID, staffID)
	if err != nil {
		return store.ErrBadPIN
	}
	if err := s.checkPIN(ctx, st, currentPIN); err != nil {
		return err
	}
	hash, err := store.HashPIN(newPIN)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			if st, err = s.staff(ctx, se.ShopID, staffID); err != nil {
				return err
			}
		}
		now := s.now()
		next := st
		next.PinHash, next.PinSetAt, next.FailedAttempts, next.LockedUntil = hash, &now, 0, nil
		next.SessEpoch++
		// Keep this device signed in: move its session to the new epoch.
		kept := se
		kept.Epoch = next.SessEpoch
		_, err = s.transact(ctx, s.staffSwap(st, next), s.put(kept, "attribute_exists(PK)", nil, nil))
		if errors.Is(err, errConflict) {
			continue
		}
		return err
	}
	return errBusy
}

// RemoveStaff deactivates someone: they are signed out and their links stop working.
// Nobody can remove themselves, and the last owner can't be removed.
func (s *Store) RemoveStaff(ctx context.Context, shopID, staffID, byStaffID string) error {
	if staffID == byStaffID {
		return store.ErrSelf
	}
	for attempt := 0; attempt < maxAttempts; attempt++ {
		st, err := s.staff(ctx, shopID, staffID)
		if err != nil {
			return err
		}
		if !st.Active {
			return store.ErrNotFound
		}
		now := s.now()
		next := st
		next.Active, next.RemovedAt = false, &now
		next.SessEpoch++
		next.LinkEpoch++
		items := []types.TransactWriteItem{s.staffSwap(st, next), s.del(st.PK, nameSK(st.Name), "")}
		if st.Role == "owner" {
			items = append(items, s.ownerDelta(shopID, -1))
		}
		i, err := s.transact(ctx, items...)
		if errors.Is(err, errConflict) && i == 2 {
			return store.ErrLastOwner
		}
		if errors.Is(err, errConflict) {
			continue
		}
		return err
	}
	return errBusy
}
