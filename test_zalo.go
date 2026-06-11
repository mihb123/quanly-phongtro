package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

const (
	botToken       = "605648857092198064:pPBVulAZkVNeDFqIPIoKTzLHPqtnJZQoGOfATdOqDeHfmfIFvTPHytfbWvJURtmI"
	secretToken    = "0ece04ba-96d"
	sendMessageURL = "https://bot-api.zaloplatforms.com/bot%s/sendMessage"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\nĐang khởi động server tại cổng 8080 để nhận webhook...")
	fmt.Println("Vui lòng dùng ngrok (ví dụ: ngrok http 8080) và cấu hình Webhook URL tới: https://<ngrok-id>.ngrok-free.app/webhooks")
	fmt.Println("Sau đó, hãy nhắn 1 tin nhắn bất kỳ cho Bot.")

	chatIdChan := make(chan string)

	http.HandleFunc("/api/v1/zalo/webhooks/94c0e326-d2af-4ff2-8c1e-578cfb62114b", func(w http.ResponseWriter, r *http.Request) {
		// Kiểm tra Secret Token nếu người dùng có cấu hình
		if secretToken != "" {
			reqToken := r.Header.Get("X-Bot-Api-Secret-Token")
			if reqToken != secretToken {
				fmt.Println("❌ Webhook bị từ chối: Secret Token không khớp (X-Bot-Api-Secret-Token).")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{"message": "Unauthorized"})
				return
			}
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Println("Lỗi đọc body webhook:", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var payload map[string]interface{}
		if err := json.Unmarshal(body, &payload); err != nil {
			fmt.Println("Lỗi parse JSON:", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		fmt.Printf("\n📩 Nhận được webhook payload:\n%s\n", string(body))

		var chatID string

		// Theo tài liệu Zalo Bot Platform (bot.zapps.me), payload có dạng { "ok": true, "result": { "message": { ... } } }
		var result map[string]interface{}
		if res, ok := payload["result"].(map[string]interface{}); ok {
			result = res
		} else {
			// Dự phòng nếu payload gửi trực tiếp message ở root (cấu trúc cũ/khác)
			result = payload
		}

		if messageInfo, ok := result["message"].(map[string]interface{}); ok {
			// Lấy từ chat.id (hỗ trợ cả PRIVATE và GROUP)
			if chatInfo, ok := messageInfo["chat"].(map[string]interface{}); ok {
				if id, ok := chatInfo["id"].(string); ok {
					chatID = id
				} else if idFloat, ok := chatInfo["id"].(float64); ok {
					chatID = fmt.Sprintf("%.0f", idFloat)
				}
			}

			// Nếu không có chat.id, thử lấy từ from.id
			if chatID == "" {
				if fromInfo, ok := messageInfo["from"].(map[string]interface{}); ok {
					if id, ok := fromInfo["id"].(string); ok {
						chatID = id
					} else if idFloat, ok := fromInfo["id"].(float64); ok {
						chatID = fmt.Sprintf("%.0f", idFloat)
					}
				}
			}
		}

		// Nếu vẫn không có chatID thì thông báo lỗi
		if chatID == "" {
			fmt.Println("⚠️ Không tìm thấy chat.id trong payload. Không thể lấy chatId.")
		} else {
			fmt.Printf("\n✅ Đã lấy được chatId (từ webhook): %s\n", chatID)
			// Tránh block nếu có nhiều webhook tới
			select {
			case chatIdChan <- chatID:
			default:
			}
		}

		// Trả về HTTP 200 OK
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Success"})
	})

	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			fmt.Println("Lỗi khởi động server:", err)
			os.Exit(1)
		}
	}()

	chatID := <-chatIdChan
	fmt.Printf("\n🚀 Sẵn sàng gửi tin nhắn tới chatId: %s\n", chatID)
	fmt.Print("Nhập nội dung tin nhắn muốn gửi (hoặc Enter để gửi tin nhắn mặc định): ")
	messageText, _ := reader.ReadString('\n')
	messageText = strings.TrimSpace(messageText)
	if messageText == "" {
		messageText = "Xin chào! Đây là tin nhắn test từ hệ thống."
	}

	err := sendMessage(botToken, chatID, messageText)
	if err != nil {
		fmt.Println("❌ Gửi tin nhắn thất bại:", err)
	} else {
		fmt.Println("✅ Gửi tin nhắn thành công!")
	}
	os.Exit(0)
}

func sendMessage(botToken, chatID, text string) error {
	url := fmt.Sprintf(sendMessageURL, botToken)

	body := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}

	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("zalo api error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}

	if ok, val := res["ok"].(bool); ok && !val {
		return fmt.Errorf("zalo api returned error: %+v", res)
	}

	return nil
}
