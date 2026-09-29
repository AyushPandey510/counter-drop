// Package ddbstore is the DynamoDB implementation of store.Repository (ADR-001).
//
// One table, keys PK/SK, four sparse GSIs, TTL attribute "ttl":
//
//	S#<shop>           PROFILE                shop (profile, hours, prices, lanes, ownerCount)
//	SLUG#<slug>        SLUG                   slug guard → shop id (unique link names)
//	S#<shop>           STAFF#<id>             staff member (PIN hash, lockout, session/link epochs)
//	S#<shop>           NAME#<lower name>      active-name guard (unique names among current staff)
//	S#<shop>           TOK#<day>#<lane>       daily token counter
//	SESS#<hash>        SESS                   staff session
//	LINK#<hash>        LINK                   one-time setup link
//	J#<job>            JOB                    job with its files embedded
//	J#<job>            EVT#<ts>#<n>           job event (audit/timeline, 1 year)
//	STATS#DEL          <hour>                 files deleted per hour (deletion health)
//
//	GSI1 board  S#<shop>#OPEN    | <queuedAt>#<job>   queued, claimed, ready
//	GSI2 day    S#<shop>#D#<day> | <queuedAt>#<job>   every submitted job, by business day
//	GSI3 work   DRAFT | COPIES#<shop> | SHOPS         drafts, finished jobs with copies to delete, shop list
//	GSI4 due    DUE              | <nextDue>#<job>    jobs with a file scheduled for deletion
//
// Every change to a job is one read-modify-write of the job item, conditioned on its version and
// written together with its event in one transaction, so two counters can never act on the same job at once.
package ddbstore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"counter-drop/api/internal/domain"
	"counter-drop/api/internal/store"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Config struct {
	Table    string
	Region   string
	Endpoint string // DynamoDB Local / moto in dev and tests; empty on AWS
}

type Store struct {
	db     *dynamodb.Client
	table  string
	now    func() time.Time
	policy domain.Policy
}

var _ store.Repository = (*Store)(nil)

// New connects to DynamoDB. On AWS, credentials come from the Lambda or task role.
func New(ctx context.Context, cfg Config, policy domain.Policy) (*Store, error) {
	opts := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(cfg.Region)}
	if cfg.Endpoint != "" {
		// Local emulators accept any credentials; don't require real ones in dev.
		opts = append(opts, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("local", "local", "")))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	db := dynamodb.NewFromConfig(awsCfg, func(o *dynamodb.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
	})
	return NewWithClient(db, cfg.Table, policy), nil
}

func NewWithClient(db *dynamodb.Client, table string, policy domain.Policy) *Store {
	return &Store{db: db, table: table, policy: policy, now: func() time.Time { return time.Now().UTC() }}
}

func (s *Store) SetClock(now func() time.Time) { s.now = now }
func (s *Store) Close()                        {}

func (s *Store) Ping(ctx context.Context) error {
	_, err := s.db.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: &s.table})
	return err
}

// --- table definition -------------------------------------------------------------------------

var indexes = []string{"GSI1", "GSI2", "GSI3", "GSI4"}

// CreateTableInput is the table as CDK must create it (kept here so tests and dev use the same shape).
func CreateTableInput(table string) *dynamodb.CreateTableInput {
	attr := func(n string) types.AttributeDefinition {
		return types.AttributeDefinition{AttributeName: aws.String(n), AttributeType: types.ScalarAttributeTypeS}
	}
	defs := []types.AttributeDefinition{attr("PK"), attr("SK")}
	var gsis []types.GlobalSecondaryIndex
	for _, ix := range indexes {
		defs = append(defs, attr(ix+"PK"), attr(ix+"SK"))
		gsis = append(gsis, types.GlobalSecondaryIndex{
			IndexName: aws.String(ix),
			KeySchema: []types.KeySchemaElement{
				{AttributeName: aws.String(ix + "PK"), KeyType: types.KeyTypeHash},
				{AttributeName: aws.String(ix + "SK"), KeyType: types.KeyTypeRange},
			},
			Projection: &types.Projection{ProjectionType: types.ProjectionTypeAll},
		})
	}
	return &dynamodb.CreateTableInput{
		TableName:              aws.String(table),
		BillingMode:            types.BillingModePayPerRequest,
		AttributeDefinitions:   defs,
		KeySchema:              []types.KeySchemaElement{{AttributeName: aws.String("PK"), KeyType: types.KeyTypeHash}, {AttributeName: aws.String("SK"), KeyType: types.KeyTypeRange}},
		GlobalSecondaryIndexes: gsis,
		StreamSpecification:    &types.StreamSpecification{StreamEnabled: aws.Bool(true), StreamViewType: types.StreamViewTypeNewAndOldImages},
	}
}

// EnsureTable creates the table if it doesn't exist (dev and tests; production tables come from CDK).
func (s *Store) EnsureTable(ctx context.Context) error {
	if _, err := s.db.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: &s.table}); err == nil {
		return nil
	}
	if _, err := s.db.CreateTable(ctx, CreateTableInput(s.table)); err != nil {
		var inUse *types.ResourceInUseException
		if !errors.As(err, &inUse) {
			return fmt.Errorf("create table %s: %w", s.table, err)
		}
	}
	w := dynamodb.NewTableExistsWaiter(s.db)
	return w.Wait(ctx, &dynamodb.DescribeTableInput{TableName: &s.table}, 2*time.Minute)
}

// PITRDays is how long point-in-time backups are kept (ADR-001 decision 2). Backups still hold
// customer names for this long after the app clears them; the privacy page says so.
const PITRDays = 7

// Configure turns on what the table needs in production: TTL on "ttl" (sessions, links, counters and
// events expire on their own) and point-in-time recovery. CDK sets the same; this is for manual setup.
func (s *Store) Configure(ctx context.Context) error {
	_, err := s.db.UpdateTimeToLive(ctx, &dynamodb.UpdateTimeToLiveInput{
		TableName:               &s.table,
		TimeToLiveSpecification: &types.TimeToLiveSpecification{AttributeName: aws.String("ttl"), Enabled: aws.Bool(true)},
	})
	if err != nil && !strings.Contains(err.Error(), "already enabled") {
		return fmt.Errorf("enable TTL: %w", err)
	}
	_, err = s.db.UpdateContinuousBackups(ctx, &dynamodb.UpdateContinuousBackupsInput{
		TableName: &s.table,
		PointInTimeRecoverySpecification: &types.PointInTimeRecoverySpecification{
			PointInTimeRecoveryEnabled: aws.Bool(true), RecoveryPeriodInDays: aws.Int32(PITRDays),
		},
	})
	if err != nil {
		return fmt.Errorf("enable point-in-time recovery: %w", err)
	}
	return nil
}

// DeleteTable drops the table (tests).
func (s *Store) DeleteTable(ctx context.Context) error {
	_, err := s.db.DeleteTable(ctx, &dynamodb.DeleteTableInput{TableName: &s.table})
	return err
}

// --- low-level helpers ---------------------------------------------------------------------------

// ts formats a time so that string order is time order (fixed width, UTC, nanoseconds).
func ts(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000000000Z") }

func key(pk, sk string) map[string]types.AttributeValue {
	return map[string]types.AttributeValue{"PK": &types.AttributeValueMemberS{Value: pk}, "SK": &types.AttributeValueMemberS{Value: sk}}
}

func sv(s string) types.AttributeValue { return &types.AttributeValueMemberS{Value: s} }
func nv(n int64) types.AttributeValue  { return &types.AttributeValueMemberN{Value: fmt.Sprint(n)} }

// get reads one item with a strongly consistent read. It returns store.ErrNotFound when absent.
func (s *Store) get(ctx context.Context, pk, sk string, out any) error {
	res, err := s.db.GetItem(ctx, &dynamodb.GetItemInput{TableName: &s.table, Key: key(pk, sk), ConsistentRead: aws.Bool(true)})
	if err != nil {
		return err
	}
	if res.Item == nil {
		return store.ErrNotFound
	}
	return attributevalue.UnmarshalMap(res.Item, out)
}

func marshal(v any) map[string]types.AttributeValue {
	m, err := attributevalue.MarshalMap(v)
	if err != nil {
		panic(fmt.Sprintf("marshal %T: %v", v, err)) // records are plain structs; this is a programming error
	}
	return m
}

func (s *Store) put(item any, cond string, names map[string]string, values map[string]types.AttributeValue) types.TransactWriteItem {
	p := &types.Put{TableName: &s.table, Item: marshal(item)}
	if cond != "" {
		p.ConditionExpression = aws.String(cond)
		if len(names) > 0 {
			p.ExpressionAttributeNames = names
		}
		if len(values) > 0 {
			p.ExpressionAttributeValues = values
		}
	}
	return types.TransactWriteItem{Put: p}
}

func (s *Store) del(pk, sk, cond string) types.TransactWriteItem {
	d := &types.Delete{TableName: &s.table, Key: key(pk, sk)}
	if cond != "" {
		d.ConditionExpression = aws.String(cond)
	}
	return types.TransactWriteItem{Delete: d}
}

// errConflict means a condition failed: another writer changed the item first (retry), or a
// uniqueness guard already exists (report).
var errConflict = errors.New("ddb: condition failed")

// transact writes all items atomically. A failed condition on any item returns errConflict with the
// index of the first failing item.
func (s *Store) transact(ctx context.Context, items ...types.TransactWriteItem) (int, error) {
	if len(items) == 1 && items[0].Put != nil {
		p := items[0].Put
		_, err := s.db.PutItem(ctx, &dynamodb.PutItemInput{TableName: p.TableName, Item: p.Item, ConditionExpression: p.ConditionExpression,
			ExpressionAttributeNames: p.ExpressionAttributeNames, ExpressionAttributeValues: p.ExpressionAttributeValues})
		var ccf *types.ConditionalCheckFailedException
		if errors.As(err, &ccf) {
			return 0, errConflict
		}
		return -1, err
	}
	_, err := s.db.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{TransactItems: items})
	var tce *types.TransactionCanceledException
	if errors.As(err, &tce) {
		for i, r := range tce.CancellationReasons {
			if aws.ToString(r.Code) == "ConditionalCheckFailed" {
				return i, errConflict
			}
		}
		// TransactionConflict: another transaction touched the same item at the same moment — retry.
		for _, r := range tce.CancellationReasons {
			if aws.ToString(r.Code) == "TransactionConflict" {
				return 0, errConflict
			}
		}
		if strings.Contains(tce.ErrorMessage(), "ConditionalCheckFailed") {
			return 0, errConflict
		}
	}
	return -1, err
}

// query runs a query on the table or an index and decodes every page into out (a pointer to a slice).
type queryOpts struct {
	index      string
	pk         string
	skOp       string // "", "<", "<=", ">", ">=", "begins_with"
	sk         string
	descending bool
	limit      int32
	consistent bool
}

func (s *Store) query(ctx context.Context, o queryOpts, each func(map[string]types.AttributeValue) (bool, error)) error {
	pkName, skName := "PK", "SK"
	if o.index != "" {
		pkName, skName = o.index+"PK", o.index+"SK"
	}
	cond := "#pk = :pk"
	names := map[string]string{"#pk": pkName}
	values := map[string]types.AttributeValue{":pk": sv(o.pk)}
	switch o.skOp {
	case "":
	case "begins_with":
		cond += " AND begins_with(#sk, :sk)"
	default:
		cond += " AND #sk " + o.skOp + " :sk"
	}
	if o.skOp != "" {
		names["#sk"] = skName
		values[":sk"] = sv(o.sk)
	}
	in := &dynamodb.QueryInput{
		TableName: &s.table, KeyConditionExpression: aws.String(cond),
		ExpressionAttributeNames: names, ExpressionAttributeValues: values,
		ScanIndexForward: aws.Bool(!o.descending),
	}
	if o.index != "" {
		in.IndexName = aws.String(o.index)
	} else if o.consistent {
		in.ConsistentRead = aws.Bool(true)
	}
	if o.limit > 0 {
		in.Limit = aws.Int32(o.limit)
	}
	p := dynamodb.NewQueryPaginator(s.db, in)
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return err
		}
		for _, item := range page.Items {
			more, err := each(item)
			if err != nil || !more {
				return err
			}
		}
		if o.limit > 0 { // Limit caps a single page; the caller asked for at most that many.
			return nil
		}
	}
	return nil
}

// ttlAfter returns a TTL attribute value (epoch seconds) for items DynamoDB may expire on its own.
// Expiry can lag by up to 48 hours, so readers always check their own timestamps too.
func ttlAfter(t time.Time, d time.Duration) int64 { return t.Add(d).Unix() }
