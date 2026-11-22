package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/buggregator/server/v2/modules/sentry"
	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type queueJobsInput struct {
	ID      string `json:"id,omitempty" jsonschema:"A job's message id (or transaction id): returns that job with every attempt, the errors of each attempt and the request that dispatched it. Omit to list jobs."`
	Queue   string `json:"queue,omitempty" jsonschema:"Only jobs of this queue"`
	Job     string `json:"job,omitempty" jsonschema:"Only jobs of this class, exact name (App\\Jobs\\SendInvoice)"`
	Status  string `json:"status,omitempty" jsonschema:"ok, failed (last attempt failed) or retried (more than one attempt)"`
	Window  string `json:"window,omitempty" jsonschema:"Only jobs whose last attempt ended within this period: 15m, 1h, 24h, 7d"`
	Project string `json:"project,omitempty" jsonschema:"Project key"`
	Limit   int    `json:"limit,omitempty" jsonschema:"Jobs to return (default 50, max 500)"`
}

func registerQueueTools(server *sdkmcp.Server, db *sql.DB) {
	sdkmcp.AddTool(server, &sdkmcp.Tool{
		Name: "queue_jobs",
		Description: "Queue jobs seen through Sentry tracing (sentry-laravel QueueIntegration: one transaction per attempt, op queue.process). " +
			"Without id: jobs, newest first, each with its class, queue, last status, attempts and duration. With id: one job's attempts in order, " +
			"the exception of each failed attempt and the request that dispatched it. Running jobs are not visible (Sentry sends an attempt when it ends).",
	}, func(ctx context.Context, req *sdkmcp.CallToolRequest, in queueJobsInput) (*sdkmcp.CallToolResult, any, error) {
		var (
			res map[string]any
			err error
		)
		if in.ID != "" {
			res, err = sentry.QueueJob(db, in.ID)
			if err == nil && res == nil {
				return nil, nil, fmt.Errorf("job not found: %s", in.ID)
			}
		} else {
			q := url.Values{}
			for k, v := range map[string]string{"queue": in.Queue, "job": in.Job, "status": in.Status, "window": in.Window, "project": in.Project} {
				if v != "" {
					q.Set(k, v)
				}
			}
			if in.Limit > 0 {
				q.Set("limit", strconv.Itoa(in.Limit))
			}
			res, err = sentry.QueueJobs(db, q)
		}
		if err != nil {
			return nil, nil, err
		}
		data, _ := json.MarshalIndent(res, "", "  ")
		return &sdkmcp.CallToolResult{
			Content: []sdkmcp.Content{&sdkmcp.TextContent{Text: string(data)}},
		}, nil, nil
	})
}
