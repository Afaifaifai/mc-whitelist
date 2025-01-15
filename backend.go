package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

type Response struct {
	Status   string                   `json:"status"`
	Messages []map[string]interface{} `json:"messages,omitempty"` // 使用omitempty來在錯誤情況下省略此欄位
	Error    string                   `json:"error,omitempty"`
}

func Start_listening() {

	// 設定 HTTP 伺服器
	mux := http.NewServeMux()
	mux.Handle("/api/", enable_CORS(http.HandlerFunc(api_handler)))

	server := &http.Server{
		Addr:    LISTEN_PORT,
		Handler: mux,
	}

	// 啟動伺服器於獨立的協程中
	go func() {
		log.Printf("Server is starting, listening on port %s", LISTEN_PORT)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// 建立一個接收系統信號的通道
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// 阻塞直到接收到信號
	<-stop
	log.Println("Signal received. Shutting down gracefully...")

	// 設定一個 5 秒的超時上下文，用於優雅關閉伺服器
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	// 優雅關閉伺服器
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server shutdown complete.")
	// 資料庫連線將在 defer 中關閉
}

// CORS middleware to allow all origins
func enable_CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 設置 CORS 標頭
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")

		// 處理預檢請求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 繼續處理下一個處理器
		next.ServeHTTP(w, r)
	})
}

// 通用 API 處理器
func api_handler(w http.ResponseWriter, r *http.Request) {
	clientIP := strings.Split(r.RemoteAddr, ":")[0] // 分割 RemoteAddr，取得 IP 部分
	if clientIP != ALLOW_IP {                       // 只允許 IPv4 和 IPv6 本地 IP
		log.Printf("Access denied: Client IP %s is not localhost", clientIP)
		return // 不進一步處理，也不回應攻擊者
	}

	// Get the API path, for example, /api/fruits -> fruits
	fmt.Println(r.URL.Path)
	pathParts := strings.Split(r.URL.Path, "/")
	fmt.Println(pathParts)

	api_method := pathParts[len(pathParts)-1]
	// fmt.Printf("%s %s\n", api_table, api_method)

	// Read the request body
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Unable to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// If the Content-Type is JSON, parse and print
	// log.Println(r.Header.Get("Content-Type"))
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		err := process(body, r.Method, api_method)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized) // 設置 HTTP 狀態碼為 401
			response := Response{
				Status: "error",
				Error:  err.Error(),
			}
			json.NewEncoder(w).Encode(response)
		} else {
			// log.Println("\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\", message)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK) // 設置 HTTP 狀態碼為 200
			response := Response{
				Status:   "success",
				Messages: nil,
			}
			json.NewEncoder(w).Encode(response)
		}
	} else {
		// Otherwise, print as plain text
		// log.Printf("Received non-JSON request resource: %s %s, Content:\n%s\n", api_table, api_method, string(body))
		log.Printf("Received non-JSON request resource: %s\n", api_method)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized) // 設置 HTTP 狀態碼為 400
		response := Response{
			Status: "error",
			Error:  "Invalid content type",
		}
		json.NewEncoder(w).Encode(response)
	}
}

func process(body []byte, request_method string, api_method string) error {
	if api_method == "auth" {
		var json_data map[string]string
		if err := json.Unmarshal(body, &json_data); err != nil && request_method != http.MethodPost {
			log.Printf("Received invalid JSON request: %s", err)
			return err
		} else {
			json.MarshalIndent(json_data, "", "  ")
			log.Printf("Received JSON request resource: %s %s\n", request_method, api_method)
			for key, value := range json_data {
				log.Printf("Data - %s: %s", key, value)
			}
			err := auth(json_data)
			if err != nil {
				return err
			} else {
				var player_id string = json_data["player_id"]
				cmd := exec.Command(SCRIPT_PATH, player_id)
				output, err := cmd.CombinedOutput()
				if err != nil {
					fmt.Printf("Error executing script: %s\n", err)
				} else {
					fmt.Printf("Script output:\n%s\n", output)
				}
				return nil
			}
		}
	} else {
		return fmt.Errorf("error: not expected operations")
	}
}

func auth(json_data map[string]string) error {
	// log.Println(json_data)
	if json_data["auth_password"] == AUTH_PASSWORD {
		log.Println("Authentication success.")
		return nil
	} else {
		log.Println("error: received incorrect password")
		return fmt.Errorf("error: incorrect password")
	}
}
