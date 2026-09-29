// Package wsgw is the live-update side of the AWS deployment (ADR-001): the API Gateway WebSocket
// routes ($connect, $disconnect, $default) and the push step that fans DynamoDB stream changes out
// to open connections. It keeps the same contract as the in-process SSE hub: topics job:<id> and
// shop:<id>, and pointer events ({"type":"job.updated","data":{...}}) that make clients refetch.
//
// Connections live in their own small table (default "cd-connections"):
//
//	T#<topic>   C#<connectionId>   who is listening to a topic (push reads this)
//	C#<conn>    T#<topic>          reverse entry, so $disconnect can clean up without a scan
//
// Both expire by TTL shortly after API Gateway's 2-hour connection limit, in case a $disconnect is lost.
package wsgw

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"counter-drop/api/internal/realtime"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// ConnectionTTL outlives API Gateway's maximum connection duration (2 h).
const ConnectionTTL = 2*time.Hour + 10*time.Minute

// Connections records which WebSocket connections listen to which topic.
type Connections struct {
	DB    *dynamodb.Client
	Table string
	Now   func() time.Time
}

func (c *Connections) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func key(pk, sk string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: pk}, "SK": &types.AttributeValueMemberS{Value: sk}}
}

func item(pk, sk string, ttl int64) map[string]types.AttributeValue {
	m := key(pk, sk)
	m["ttl"] = &types.AttributeValueMemberN{Value: fmt.Sprint(ttl)}
	return m
}

// Add registers a connection for a topic.
func (c *Connections) Add(ctx context.Context, connID, topic string) error {
	ttl := c.now().Add(ConnectionTTL).Unix()
	_, err := c.DB.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: []types.TransactWriteItem{
		{Put: &types.Put{TableName: &c.Table, Item: item("T#"+topic, "C#"+connID, ttl)}},
		{Put: &types.Put{TableName: &c.Table, Item: item("C#"+connID, "T#"+topic, ttl)}},
	}})
	return err
}

// Remove forgets a connection (every topic it listened to).
func (c *Connections) Remove(ctx context.Context, connID string) error {
	res, err := c.DB.Query(ctx, &dynamodb.QueryInput{
		TableName: &c.Table, KeyConditionExpression: aws.String("PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{":pk": &types.AttributeValueMemberS{Value: "C#" + connID}},
	})
	if err != nil {
		return err
	}
	for _, it := range res.Items {
		topic := strings.TrimPrefix(it["SK"].(*types.AttributeValueMemberS).Value, "T#")
		for _, k := range []map[string]types.AttributeValue{key("T#"+topic, "C#"+connID), key("C#"+connID, "T#"+topic)} {
			if _, err := c.DB.DeleteItem(ctx, &dynamodb.DeleteItemInput{TableName: &c.Table, Key: k}); err != nil {
				return err
			}
		}
	}
	return nil
}

// List returns the connections listening to a topic.
func (c *Connections) List(ctx context.Context, topic string) ([]string, error) {
	var out []string
	p := dynamodb.NewQueryPaginator(c.DB, &dynamodb.QueryInput{
		TableName: &c.Table, KeyConditionExpression: aws.String("PK = :pk"),
		ExpressionAttributeValues: map[string]types.AttributeValue{":pk": &types.AttributeValueMemberS{Value: "T#" + topic}},
	})
	now := c.now().Unix()
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, it := range page.Items {
			// TTL deletion can lag; skip entries that have already expired.
			if n, ok := it["ttl"].(*types.AttributeValueMemberN); ok {
				var ttl int64
				fmt.Sscan(n.Value, &ttl)
				if ttl < now {
					continue
				}
			}
			out = append(out, strings.TrimPrefix(it["SK"].(*types.AttributeValueMemberS).Value, "C#"))
		}
	}
	return out, nil
}

// CreateTableInput is the connections table (CDK creates the same; tests and dev use this).
func CreateTableInput(table string) *dynamodb.CreateTableInput {
	return &dynamodb.CreateTableInput{
		TableName:   aws.String(table),
		BillingMode: types.BillingModePayPerRequest,
		AttributeDefinitions: []types.AttributeDefinition{
			{AttributeName: aws.String("PK"), AttributeType: types.ScalarAttributeTypeS},
			{AttributeName: aws.String("SK"), AttributeType: types.ScalarAttributeTypeS},
		},
		KeySchema: []types.KeySchemaElement{
			{AttributeName: aws.String("PK"), KeyType: types.KeyTypeHash},
			{AttributeName: aws.String("SK"), KeyType: types.KeyTypeRange},
		},
	}
}

// --- sending ---------------------------------------------------------------------------------

// ErrGone means the connection no longer exists (API Gateway returns 410).
var ErrGone = errors.New("connection gone")

// Poster sends one message to one connection.
type Poster interface {
	Post(ctx context.Context, connectionID string, data []byte) error
}

// Gateway handles the WebSocket routes and pushes events.
type Gateway struct {
	Conns  *Connections
	Key    []byte // same key the API signs tickets with (CD_REALTIME_KEY)
	Poster Poster
	Logger *slog.Logger
	Now    func() time.Time
}

func (g *Gateway) now() time.Time {
	if g.Now != nil {
		return g.Now()
	}
	return time.Now()
}

// Connect accepts a connection only with a valid ticket, and subscribes it to the ticket's topic.
func (g *Gateway) Connect(ctx context.Context, connID, ticket string) error {
	topic, err := realtime.VerifyTicket(g.Key, ticket, g.now())
	if err != nil {
		return err
	}
	return g.Conns.Add(ctx, connID, topic)
}

func (g *Gateway) Disconnect(ctx context.Context, connID string) error {
	return g.Conns.Remove(ctx, connID)
}

// Push delivers messages to every connection on their topics. Connections that are gone are removed;
// other failures are logged and skipped so one bad connection never blocks the stream.
func (g *Gateway) Push(ctx context.Context, msgs []realtime.Message) error {
	for _, m := range msgs {
		body, err := json.Marshal(m.Event)
		if err != nil {
			return err
		}
		conns, err := g.Conns.List(ctx, m.Topic)
		if err != nil {
			return err
		}
		for _, c := range conns {
			err := g.Poster.Post(ctx, c, body)
			switch {
			case errors.Is(err, ErrGone):
				if err := g.Conns.Remove(ctx, c); err != nil && g.Logger != nil {
					g.Logger.Warn("remove gone connection", "connection", c, "error", err)
				}
			case err != nil && g.Logger != nil:
				g.Logger.Warn("push failed", "connection", c, "topic", m.Topic, "error", err)
			}
		}
	}
	return nil
}
