package mongo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Rrens/text-to-sql/internal/mcp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Adapter struct {
	client *mongo.Client
	db     *mongo.Database
	config mcp.ConnectionConfig
}

func NewAdapter() mcp.Adapter {
	return &Adapter{}
}

func (a *Adapter) DatabaseType() string {
	return "mongodb"
}

func (a *Adapter) SQLDialect() string {
	return "mongodb"
}

func (a *Adapter) Connect(ctx context.Context, config mcp.ConnectionConfig) error {
	a.config = config

	// Build connection string
	// Default: mongodb://user:pass@host:port
	uri := fmt.Sprintf("mongodb://%s:%d", config.Host, config.Port)
	if config.Username != "" && config.Password != "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%d", config.Username, config.Password, config.Host, config.Port)
	}

	clientOpts := options.Client().ApplyURI(uri)
	if config.TimeoutSeconds > 0 {
		clientOpts.SetConnectTimeout(time.Duration(config.TimeoutSeconds) * time.Second)
	}

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping: %w", err)
	}

	a.client = client
	a.db = client.Database(config.Database)

	return nil
}

func (a *Adapter) Close() error {
	if a.client != nil {
		return a.client.Disconnect(context.Background())
	}
	return nil
}

func (a *Adapter) HealthCheck(ctx context.Context) error {
	if a.client == nil {
		return fmt.Errorf("not connected")
	}
	return a.client.Ping(ctx, nil)
}

func (a *Adapter) ListTables(ctx context.Context) ([]string, error) {
	if a.db == nil {
		return nil, fmt.Errorf("no database selected")
	}
	return a.db.ListCollectionNames(ctx, bson.D{})
}

func (a *Adapter) DescribeTable(ctx context.Context, tableName string) (*mcp.TableInfo, error) {
	// For MongoDB, we don't have a rigid schema.
	// We'll return a generic "document" column.

	// Optionally we could sample a document, but for now we keep it simple.
	return &mcp.TableInfo{
		Name: tableName,
		Columns: []mcp.ColumnInfo{
			{Name: "_id", DataType: "ObjectId", PrimaryKey: true},
			{Name: "document", DataType: "JSON", Description: "Full document content"},
		},
	}, nil
}

func (a *Adapter) GetSchemaDDL(ctx context.Context) (string, error) {
	collections, err := a.ListTables(ctx)
	if err != nil {
		return "", err
	}

	// Represent schema as a list of collections
	schema := map[string]interface{}{
		"database":    a.config.Database,
		"collections": collections,
		"note":        "NoSQL database - schema is flexible",
	}

	bytes, _ := json.MarshalIndent(schema, "", "  ")
	return string(bytes), nil
}

func (a *Adapter) ValidateQuery(sql string) error {
	var cmd bson.D
	if err := bson.UnmarshalExtJSON([]byte(sql), true, &cmd); err != nil {
		return fmt.Errorf("invalid mongodb query: expected JSON object: %w", err)
	}

	if len(cmd) == 0 {
		return fmt.Errorf("empty command")
	}

	// MongoDB commands are order-sensitive; the first key is the command.
	commandName := cmd[0].Key

	// Allowlist of read-only commands
	allowedCommands := map[string]bool{
		"find":            true,
		"aggregate":       true,
		"count":           true,
		"distinct":        true,
		"listCollections": true,
		"buildInfo":       true,
		"collStats":       true,
		"dbStats":         true,
		"ping":            true,
	}

	if !allowedCommands[commandName] {
		return fmt.Errorf("command '%s' is not allowed (read-only mode enabled)", commandName)
	}

	// Deep strict check for aggregation to prevent writing via $out or $merge
	if commandName == "aggregate" {
		for _, elem := range cmd {
			if elem.Key == "pipeline" {
				if pipeline, ok := elem.Value.(bson.A); ok {
					for _, stage := range pipeline {
						if stageDoc, ok := stage.(bson.D); ok {
							for _, stageElem := range stageDoc {
								if stageElem.Key == "$out" || stageElem.Key == "$merge" {
									return fmt.Errorf("aggregation stage '%s' is not allowed", stageElem.Key)
								}
							}
						}
					}
				}
			}
		}
	}

	return nil
}

func (a *Adapter) ExecuteQuery(ctx context.Context, sql string, opts mcp.QueryOptions) (*mcp.QueryResult, error) {
	if err := a.ValidateQuery(sql); err != nil {
		return nil, err
	}

	// Expecting JSON: {"collection": "users", "filter": {...}, "operation": "find"}
	// Or simpler: {"find": "users", "filter": {...}} matching runCommand style somewhat

	var cmd bson.D
	if err := bson.UnmarshalExtJSON([]byte(sql), true, &cmd); err != nil {
		return nil, fmt.Errorf("failed to parse query JSON: %w", err)
	}

	// This is a simplified execution that runs specific commands
	// Ideally we'd map "sql" to true MongoDB commands.
	// For now, let's assume raw runCommand for flexibility if the user knows what they are doing.

	res := a.db.RunCommand(ctx, cmd)
	if err := res.Err(); err != nil {
		return nil, fmt.Errorf("execution error: %w", err)
	}

	var raw bson.M
	if err := res.Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode error: %w", err)
	}

	// Convert result to "rows"
	// If it's a cursor result (like from 'find')
	rows := [][]any{}
	columns := []string{"result"} // Default single column for raw JSON

	// Handle 'cursor' response for find/aggregate
	if cursor, ok := raw["cursor"].(bson.M); ok {
		if firstBatch, ok := cursor["firstBatch"].(bson.A); ok {
			columns = []string{"json_document"}
			for _, doc := range firstBatch {
				jsonBytes, _ := json.Marshal(doc)
				rows = append(rows, []any{string(jsonBytes)})
			}
		}
	} else {
		// Generic command response
		jsonBytes, _ := json.Marshal(raw)
		rows = append(rows, []any{string(jsonBytes)})
	}

	return &mcp.QueryResult{
		Columns:  columns,
		Rows:     rows,
		RowCount: len(rows),
	}, nil
}
