package wsgw

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"counter-drop/api/internal/realtime"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type fakePoster struct {
	mu   sync.Mutex
	sent map[string][]string
	gone map[string]bool
}

func (f *fakePoster) Post(_ context.Context, conn string, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.gone[conn] {
		return ErrGone
	}
	f.sent[conn] = append(f.sent[conn], string(data))
	return nil
}

func newGateway(t *testing.T) (*Gateway, *fakePoster) {
	t.Helper()
	endpoint := os.Getenv("CD_TEST_DYNAMODB_ENDPOINT")
	if endpoint == "" {
		t.Skip("CD_TEST_DYNAMODB_ENDPOINT not set")
	}
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("ap-south-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("local", "local", "")))
	if err != nil {
		t.Fatal(err)
	}
	db := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) { o.BaseEndpoint = aws.String(endpoint) })
	table := fmt.Sprintf("cd-conn-test-%d", time.Now().UnixNano())
	if _, err := db.CreateTable(ctx, CreateTableInput(table)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.DeleteTable(context.Background(), &dynamodb.DeleteTableInput{TableName: &table}) })
	p := &fakePoster{sent: map[string][]string{}, gone: map[string]bool{}}
	now := func() time.Time { return time.Date(2026, 10, 1, 5, 30, 0, 0, time.UTC) }
	return &Gateway{Conns: &Connections{DB: db, Table: table, Now: now}, Key: []byte("test-key-test-key-test-key-12345"), Poster: p, Now: now}, p
}

func TestConnectPushDisconnect(t *testing.T) {
	g, p := newGateway(t)
	ctx := context.Background()
	jobTicket, _ := realtime.SignTicket(g.Key, "job:job_1", g.Now())
	shopTicket, _ := realtime.SignTicket(g.Key, "shop:shop_1", g.Now())

	if err := g.Connect(ctx, "c-customer", jobTicket); err != nil {
		t.Fatal(err)
	}
	for _, c := range []string{"c-board1", "c-board2"} {
		if err := g.Connect(ctx, c, shopTicket); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.Connect(ctx, "c-intruder", "not-a-ticket"); err == nil {
		t.Fatal("connection without a valid ticket accepted")
	}
	expired, _ := realtime.SignTicket(g.Key, "job:job_1", g.Now().Add(-2*realtime.TicketTTL))
	if err := g.Connect(ctx, "c-late", expired); err == nil {
		t.Fatal("expired ticket accepted")
	}

	p.gone["c-board2"] = true // this board closed its tab without a clean disconnect
	msgs := []realtime.Message{
		{Topic: "job:job_1", Event: realtime.Event{Type: "job.updated", Data: map[string]any{"id": "job_1"}}},
		{Topic: "shop:shop_1", Event: realtime.Event{Type: "queue.changed", Data: map[string]string{"jobId": "job_1"}}},
	}
	if err := g.Push(ctx, msgs); err != nil {
		t.Fatal(err)
	}
	if got := p.sent["c-customer"]; len(got) != 1 || got[0] != `{"type":"job.updated","data":{"id":"job_1"}}` {
		t.Fatalf("customer got %v", got)
	}
	if got := p.sent["c-board1"]; len(got) != 1 {
		t.Fatalf("board got %v", got)
	}
	if len(p.sent["c-intruder"]) != 0 {
		t.Fatal("unauthenticated connection received data")
	}

	// The gone connection was cleaned up; the live one stays.
	conns, _ := g.Conns.List(ctx, "shop:shop_1")
	sort.Strings(conns)
	if len(conns) != 1 || conns[0] != "c-board1" {
		t.Fatalf("shop connections after push: %v", conns)
	}

	if err := g.Disconnect(ctx, "c-customer"); err != nil {
		t.Fatal(err)
	}
	if conns, _ := g.Conns.List(ctx, "job:job_1"); len(conns) != 0 {
		t.Fatalf("connection still listed after disconnect: %v", conns)
	}
	// Disconnecting an unknown connection is harmless ($disconnect can follow a refused $connect).
	if err := g.Disconnect(ctx, "c-intruder"); err != nil {
		t.Fatal(err)
	}
}

func TestExpiredConnectionsAreSkipped(t *testing.T) {
	g, p := newGateway(t)
	ctx := context.Background()
	tk, _ := realtime.SignTicket(g.Key, "shop:shop_1", g.Now())
	if err := g.Connect(ctx, "c-old", tk); err != nil {
		t.Fatal(err)
	}
	later := g.Now().Add(ConnectionTTL + time.Minute)
	g.Conns.Now = func() time.Time { return later }
	if err := g.Push(ctx, []realtime.Message{{Topic: "shop:shop_1", Event: realtime.Event{Type: "queue.changed"}}}); err != nil {
		t.Fatal(err)
	}
	if len(p.sent["c-old"]) != 0 {
		t.Fatal("pushed to a connection past its TTL")
	}
}
