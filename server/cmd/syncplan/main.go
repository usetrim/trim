package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/usetrim/trim/server/internal/billing/catalogsync"
	"github.com/usetrim/trim/server/internal/billing/paddleapi"
)

func main() {
	_ = godotenv.Load(".env")
	dbURL := os.Getenv("DATABASE_URL")
	key := os.Getenv("PADDLE_API_KEY")
	env := os.Getenv("PADDLE_ENV")
	if dbURL == "" || key == "" {
		log.Fatal("DATABASE_URL and PADDLE_API_KEY required")
	}
	timeout, _ := strconv.Atoi(os.Getenv("PADDLE_HTTP_TIMEOUT_SEC"))
	if timeout < 1 {
		timeout = 30
	}
	maxPages, _ := strconv.Atoi(os.Getenv("PADDLE_LIST_MAX_PAGES"))
	if maxPages < 1 {
		maxPages = 100
	}
	perPage, _ := strconv.Atoi(os.Getenv("PADDLE_LIST_PER_PAGE"))
	if perPage < 1 {
		perPage = 50
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	client, err := paddleapi.New(key, env, timeout, maxPages, perPage)
	if err != nil {
		log.Fatal(err)
	}
	syncer := catalogsync.New(pool, client)
	res, err := syncer.SyncPlan(ctx, "enterprise")
	if err != nil {
		log.Fatalf("sync: %v", err)
	}
	fmt.Printf("%+v\n", res)
}
