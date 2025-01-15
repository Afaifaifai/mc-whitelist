package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var DB *sql.DB // Initialize in the fucnction Init()

func main() {
	Start_listening()

	// 建立 context 和信號監聽
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // 確保所有資源釋放

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// 啟動工作協程
	go func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				log.Println("Worker shutting down...")
				return
			default:
				log.Println("Worker is processing...")
				time.Sleep(1 * time.Second)
			}
		}
	}(ctx)

	// 等待信號
	<-stop
	log.Println("Signal received. Initiating shutdown...")

	// 取消 context 並釋放資源
	cancel()
	if DB != nil {
		log.Println("Closing database connection...")
		DB.Close()
		log.Println("Database connection closed.")
	}
}
