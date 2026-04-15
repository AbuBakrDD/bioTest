package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

// 🔐 Читаем токены из переменных окружения (или .env при локальном запуске)
var (
	botToken = getEnv("TG_BOT_TOKEN", "ТВОЙ_ТОКЕН")
	chatID   = getEnv("TG_CHAT_ID", "ТВОЙ_CHAT_ID")
)

type Result struct {
	TestName string `json:"testName"`
	Student  string `json:"student"`
	Score    string `json:"score"`
	Variant  string `json:"variant"`
	Errors   string `json:"errors"`
}

// 🌐 CORS + обработчик
func handler(w http.ResponseWriter, r *http.Request) {
	// ✅ Разрешаем запросы с любого источника (для МВП)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// ✅ Обрабатываем preflight
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// ✅ Только POST
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// ✅ Читаем тело запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var res Result
	if err := json.Unmarshal(body, &res); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// ✅ Формируем сообщение для Telegram
	emoji := "🎉"
	if res.Errors != "" {
		emoji = "📝"
	}
	text := "📊 *" + res.TestName + "*\n" +
		"👤 *Ученик:* " + res.Student + "\n" +
		"📈 *Баллы:* " + res.Score + " " + emoji + "\n" +
		"🔢 *Вариант:* " + res.Variant
	if res.Errors != "" {
		text += "\n\n❌ *Ошибки:*\n" + res.Errors
	}

	// ✅ Отправляем в Telegram
	tgURL := "https://api.telegram.org/bot" + botToken + "/sendMessage"
	payload, _ := json.Marshal(map[string]string{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	})

	resp, err := http.Post(tgURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("Telegram error: %v", err)
		http.Error(w, "Failed to send to Telegram", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// ✅ Ответ фронту
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// 🔄 Хелпер для .env
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	// Пробуем прочитать из .env (для локальной разработки)
	if data, err := os.ReadFile(".env"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, key+"=") {
				return strings.TrimPrefix(line, key+"=")
			}
		}
	}
	return fallback
}

func main() {
	http.HandleFunc("/send", handler)
	log.Println("🚀 Сервер запущен на :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
