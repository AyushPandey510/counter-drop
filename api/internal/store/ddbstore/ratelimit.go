package ddbstore

import (
	"context"
	"strconv"
	"time"

	"counter-drop/api/internal/ratelimit"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var _ ratelimit.Shared = (*Store)(nil)

// Hit counts one request in the current one-minute window: item RL#<key> / <window start>, removed by TTL.
func (s *Store) Hit(ctx context.Context, key string, now time.Time) (int, error) {
	start := ratelimit.WindowStart(now)
	res, err := s.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &s.table, Key: map[string]types.AttributeValue{"PK": sv("RL#" + key), "SK": sv(start.Format("2006-01-02T15:04"))},
		UpdateExpression:          aws.String("ADD #h :one SET #t = :ttl"),
		ExpressionAttributeNames:  map[string]string{"#h": "Hits", "#t": "ttl"},
		ExpressionAttributeValues: map[string]types.AttributeValue{":one": nv(1), ":ttl": nv(start.Add(2 * ratelimit.Window).Unix())},
		ReturnValues:              types.ReturnValueUpdatedNew,
	})
	if err != nil {
		return 0, err
	}
	n, _ := res.Attributes["Hits"].(*types.AttributeValueMemberN)
	if n == nil {
		return 0, nil
	}
	return strconv.Atoi(n.Value)
}
