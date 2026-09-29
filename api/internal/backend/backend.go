// Package backend opens the DynamoDB store from the configuration.
package backend

import (
	"context"

	"counter-drop/api/internal/config"
	"counter-drop/api/internal/domain"
	"counter-drop/api/internal/store"
	"counter-drop/api/internal/store/ddbstore"
)

// Open connects to the DynamoDB table. When pointed at DynamoDB Local (CD_DYNAMODB_ENDPOINT) it also
// creates the table if missing; on AWS the table is created by the CDK stack.
func Open(ctx context.Context, cfg config.Config, policy domain.Policy) (store.Repository, error) {
	st, err := ddbstore.New(ctx, ddbstore.Config{Table: cfg.Dynamo.Table, Region: cfg.Dynamo.Region, Endpoint: cfg.Dynamo.Endpoint}, policy)
	if err != nil {
		return nil, err
	}
	if cfg.Dynamo.Endpoint != "" {
		if err := st.EnsureTable(ctx); err != nil {
			return nil, err
		}
	}
	return st, nil
}
