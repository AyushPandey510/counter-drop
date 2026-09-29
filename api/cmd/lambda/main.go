// Command lambda is the AWS Lambda binary (ADR-001). One binary, several handlers, chosen by CD_RUNTIME:
//
//	lambda-ws     API Gateway WebSocket routes: $connect (ticket check), $disconnect, $default (heartbeat)
//	lambda-push   DynamoDB stream of the main table → live-update messages to open connections
//
// Phase 3 adds lambda-api (the HTTP API) and lambda-sweeper (the deletion worker) here.
//
// Build for Lambda (arm64, provided.al2023): GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags lambda.norpc -o bootstrap ./cmd/lambda
package main

import (
	"context"
	"log/slog"
	"os"

	"counter-drop/api/internal/config"
	"counter-drop/api/internal/realtime/wsgw"
	"counter-drop/api/internal/store/ddbstore"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	ctx := context.Background()
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		logger.Error("aws config", "error", err)
		os.Exit(1)
	}
	gw := &wsgw.Gateway{
		Conns:  &wsgw.Connections{DB: dynamodb.NewFromConfig(awsCfg), Table: cfg.Realtime.ConnectionsTable},
		Key:    []byte(cfg.Realtime.Key),
		Logger: logger,
	}
	switch rt := os.Getenv("CD_RUNTIME"); rt {
	case "lambda-ws":
		if len(gw.Key) < 32 {
			logger.Error("CD_REALTIME_KEY must be at least 32 characters")
			os.Exit(1)
		}
		lambda.Start(wsHandler(gw, logger))
	case "lambda-push":
		if cfg.Realtime.ManagementEndpoint == "" {
			logger.Error("CD_WS_MANAGEMENT_ENDPOINT is required")
			os.Exit(1)
		}
		gw.Poster = wsgw.NewAPIGatewayPoster(awsCfg, cfg.Realtime.ManagementEndpoint)
		lambda.Start(pushHandler(gw, logger))
	default:
		logger.Error("unknown CD_RUNTIME", "runtime", rt)
		os.Exit(1)
	}
}

func wsHandler(gw *wsgw.Gateway, logger *slog.Logger) func(context.Context, events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	return func(ctx context.Context, req events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
		conn := req.RequestContext.ConnectionID
		switch req.RequestContext.RouteKey {
		case "$connect":
			if err := gw.Connect(ctx, conn, req.QueryStringParameters["t"]); err != nil {
				logger.Info("websocket refused", "connection", conn, "error", err)
				return events.APIGatewayProxyResponse{StatusCode: 401}, nil
			}
		case "$disconnect":
			if err := gw.Disconnect(ctx, conn); err != nil {
				logger.Warn("websocket disconnect cleanup", "connection", conn, "error", err)
			}
		default:
			// Heartbeats ({"type":"ping"}) only keep the connection inside API Gateway's 10-minute idle limit.
		}
		return events.APIGatewayProxyResponse{StatusCode: 200}, nil
	}
}

// pushHandler never fails the batch: a missed live update is repaired by the client's refetch on
// reconnect and its slow fallback poll, while a failed batch would block the stream shard.
func pushHandler(gw *wsgw.Gateway, logger *slog.Logger) func(context.Context, events.DynamoDBEvent) error {
	return func(ctx context.Context, ev events.DynamoDBEvent) error {
		for _, rec := range ev.Records {
			msgs, err := ddbstore.ChangeEvents(toAV(rec.Change.OldImage), toAV(rec.Change.NewImage))
			if err != nil {
				logger.Warn("decode stream record", "event_id", rec.EventID, "error", err)
				continue
			}
			if err := gw.Push(ctx, msgs); err != nil {
				logger.Warn("push", "event_id", rec.EventID, "error", err)
			}
		}
		return nil
	}
}

// toAV converts the Lambda event's attribute values to the SDK's types.
func toAV(m map[string]events.DynamoDBAttributeValue) map[string]types.AttributeValue {
	if m == nil {
		return nil
	}
	out := make(map[string]types.AttributeValue, len(m))
	for k, v := range m {
		out[k] = convert(v)
	}
	return out
}

func convert(v events.DynamoDBAttributeValue) types.AttributeValue {
	switch v.DataType() {
	case events.DataTypeString:
		return &types.AttributeValueMemberS{Value: v.String()}
	case events.DataTypeNumber:
		return &types.AttributeValueMemberN{Value: v.Number()}
	case events.DataTypeBoolean:
		return &types.AttributeValueMemberBOOL{Value: v.Boolean()}
	case events.DataTypeBinary:
		return &types.AttributeValueMemberB{Value: v.Binary()}
	case events.DataTypeStringSet:
		return &types.AttributeValueMemberSS{Value: v.StringSet()}
	case events.DataTypeNumberSet:
		return &types.AttributeValueMemberNS{Value: v.NumberSet()}
	case events.DataTypeBinarySet:
		return &types.AttributeValueMemberBS{Value: v.BinarySet()}
	case events.DataTypeList:
		var l []types.AttributeValue
		for _, x := range v.List() {
			l = append(l, convert(x))
		}
		return &types.AttributeValueMemberL{Value: l}
	case events.DataTypeMap:
		return &types.AttributeValueMemberM{Value: toAV(v.Map())}
	default:
		return &types.AttributeValueMemberNULL{Value: true}
	}
}
