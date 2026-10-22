package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"github.com/vionarion/go-projects/url-shortener/internal/api"
	"github.com/vionarion/go-projects/url-shortener/internal/store"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	n, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return def
	}
	return n
}

func main() {
	// .env is optional: in Docker the settings come from the environment.
	_ = godotenv.Load()

	rdb := redis.NewClient(&redis.Options{
		Addr:     env("REDIS_ADDR", "localhost:6379"),
		Password: os.Getenv("REDIS_PASSWORD"),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Do not exit if Redis is down: the client reconnects on its own and /health keeps the
	// instance out of the load balancer until Redis is back (exiting here caused CrashLoopBackOff).
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("redis not reachable yet: %v", err)
	}

	app := api.New(api.Config{
		Domain:     os.Getenv("DOMAIN"),
		Quota:      envInt("API_QUOTA", 10),
		Window:     time.Duration(envInt("API_QUOTA_WINDOW_MIN", 30)) * time.Minute,
		DefaultTTL: time.Duration(envInt("DEFAULT_EXPIRY_HOURS", 24)) * time.Hour,
		MaxTTL:     time.Duration(envInt("MAX_EXPIRY_HOURS", 24*30)) * time.Hour,
		CodeLength: envInt("CODE_LENGTH", 7),
		TrustProxy: os.Getenv("TRUST_PROXY") == "true",
	}, store.NewRedis(rdb))

	go func() {
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		log.Println("shutting down")
		_ = app.ShutdownWithTimeout(10 * time.Second)
	}()

	if err := app.Listen(env("APP_PORT", ":3000")); err != nil {
		log.Fatal(err)
	}
}
