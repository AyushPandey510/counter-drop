package wsgw

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi"
	"github.com/aws/aws-sdk-go-v2/service/apigatewaymanagementapi/types"
)

// APIGatewayPoster posts to connections through the API Gateway management API
// (endpoint https://<api-id>.execute-api.<region>.amazonaws.com/<stage>, CD_WS_MANAGEMENT_ENDPOINT).
type APIGatewayPoster struct {
	Client *apigatewaymanagementapi.Client
}

func NewAPIGatewayPoster(cfg aws.Config, endpoint string) *APIGatewayPoster {
	return &APIGatewayPoster{Client: apigatewaymanagementapi.NewFromConfig(cfg, func(o *apigatewaymanagementapi.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})}
}

func (p *APIGatewayPoster) Post(ctx context.Context, connectionID string, data []byte) error {
	_, err := p.Client.PostToConnection(ctx, &apigatewaymanagementapi.PostToConnectionInput{ConnectionId: &connectionID, Data: data})
	var gone *types.GoneException
	if errors.As(err, &gone) {
		return ErrGone
	}
	return err
}
