package graph

import (
	"context"
	"fmt"
	"log"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Client wraps Memgraph connection using Neo4j driver (Bolt protocol)
type Client struct {
	driver neo4j.DriverWithContext
}

// NewClient creates a new Memgraph client
func NewClient(uri, username, password string) (*Client, error) {
	driver, err := neo4j.NewDriverWithContext(
		uri,
		neo4j.BasicAuth(username, password, ""),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create driver: %w", err)
	}

	// Verify connectivity
	ctx := context.Background()
	if err := driver.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("failed to verify connectivity: %w", err)
	}

	log.Println("✅ Connected to Memgraph")

	return &Client{driver: driver}, nil
}

// Close closes the Memgraph connection
func (c *Client) Close(ctx context.Context) error {
	return c.driver.Close(ctx)
}

// ExecuteWrite executes a write query
func (c *Client) ExecuteWrite(ctx context.Context, cypher string, params map[string]interface{}) error {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, cypher, params)
		if err != nil {
			return nil, err
		}
		return result.Consume(ctx)
	})

	return err
}

// ExecuteRead executes a read query and returns results
func (c *Client) ExecuteRead(ctx context.Context, cypher string, params map[string]interface{}) ([]map[string]interface{}, error) {
	session := c.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (interface{}, error) {
		result, err := tx.Run(ctx, cypher, params)
		if err != nil {
			return nil, err
		}

		records, err := result.Collect(ctx)
		if err != nil {
			return nil, err
		}

		var results []map[string]interface{}
		for _, record := range records {
			recordMap := make(map[string]interface{})
			for _, key := range record.Keys {
				recordMap[key] = record.Values[record.Keys[0]]
			}
			results = append(results, recordMap)
		}

		return results, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]map[string]interface{}), nil
}


