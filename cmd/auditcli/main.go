package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: auditcli run|status <job_id>")
		return
	}
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	ctx := context.Background()
	switch os.Args[1] {
	case "run":
		jobID := uuid.New().String()
		jb, _ := json.Marshal(struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		}{jobID, "audit_export"})
		_ = rdb.RPush(ctx, "jobs", jb).Err()
		fmt.Println(jobID)
	case "status":
		if len(os.Args) < 3 {
			fmt.Println("job id required")
			return
		}
		val, err := rdb.Get(ctx, "audit_export:"+os.Args[2]).Result()
		if err != nil {
			fmt.Println("error:", err)
			return
		}
		fmt.Println(val)
	default:
		fmt.Println("unknown command")
	}
}
