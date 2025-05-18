// main.go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type AlertManagerPayload struct {
	Alerts []Alert `json:"alerts"`
}

type Alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
}

var (
	defaultReceivers = strings.Split(os.Getenv("DEFAULT_RECEPIENTS"), ",")
	botToken         = os.Getenv("TELEGRAM_BOT_TOKEN")
	grafanaURL       = os.Getenv("GRAFANA_BASE_URL")
	promURL          = os.Getenv("PROM_BASE_URL")
	amURL            = os.Getenv("AM_BASE_URL")

	alertCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "am_telegram_alerts_total",
			Help: "Total alerts received",
		},
		[]string{"status", "alertname"},
	)

	sendCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "am_telegram_alerts_send",
			Help: "Total alerts sent",
		},
		[]string{"receiver"},
	)

	sendErrorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "am_telegram_alerts_send_errors",
			Help: "Total alert send errors",
		},
		[]string{"receiver"},
	)
)

func init() {
	prometheus.MustRegister(alertCounter)
	prometheus.MustRegister(sendCounter)
	prometheus.MustRegister(sendErrorCounter)
}

func logJSON(severity, message string) {
	logEntry := map[string]string{
		"severity": severity,
		"message":  message,
	}
	b, _ := json.Marshal(logEntry)
	fmt.Println(string(b))
}

func emojiFor(alert Alert) string {
	if alert.Status == "resolved" {
		return "🟢"
	}
	switch alert.Labels["severity"] {
	case "info":
		return "🔵"
	case "warning":
		return "🟡"
	case "critical":
		return "🔴"
	default:
		return "⚪️"
	}
}

func inWorkingHours() bool {
	hour := time.Now().Hour()
	return hour >= 8 && hour <= 22
}

func parseChatIDs(base []string, overrides string) []string {
	ids := append([]string{}, base...)
	if overrides != "" {
		ids = []string{} // override
		ids = append(ids, strings.Split(overrides, ",")...)
	}
	return ids
}

func handleAlert(w http.ResponseWriter, r *http.Request) {
	var payload AlertManagerPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		logJSON("ERROR", fmt.Sprintf("Error decoding alert payload: %v", err))
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	logJSON("NORMAL", fmt.Sprintf("Received %d alerts", len(payload.Alerts)))

	for _, alert := range payload.Alerts {
		alertname := alert.Labels["alertname"]
		alertCounter.WithLabelValues(alert.Status, alertname).Inc()
		logJSON("NORMAL", fmt.Sprintf("Processing alert: %s [%s]", alertname, alert.Status))

		ann := alert.Annotations
		if alert.Status == "resolved" && ann["do_not_send_resolved"] == "true" {
			logJSON("NORMAL", fmt.Sprintf("Skipping resolved alert %s due to do_not_send_resolved", alertname))
			continue
		}
		if ann["only_working_hours"] == "true" && !inWorkingHours() {
			logJSON("NORMAL", fmt.Sprintf("Skipping alert %s due to only_working_hours", alertname))
			continue
		}

		receivers := parseChatIDs(defaultReceivers, ann["override_receivers"])
		if ann["additional_receivers"] != "" {
			receivers = append(receivers, strings.Split(ann["additional_receivers"], ",")...)
		}

		emoji := emojiFor(alert)
		title := ann["title"]
		desc := ann["description"]
		msg := fmt.Sprintf("%s *%s*\n%s\n[Query](%s) / [Mute](%s) / [Grafana](%s)",
			emoji,
			title,
			desc,
			alert.GeneratorURL,
			amURL,
			grafanaURL,
		)

		for _, rcpt := range receivers {
			sendToTelegram(rcpt, msg)
			sendCounter.WithLabelValues(rcpt).Inc()
			logJSON("NORMAL", fmt.Sprintf("Alert sent to receiver: %s", rcpt))
		}
	}
}

func sendToTelegram(chatID, text string) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	body := map[string]string{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}
	b, _ := json.Marshal(body)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(b))
	if err != nil || (resp != nil && resp.StatusCode >= 300) {
		errMsg := fmt.Sprintf("Failed to send alert to %s: %v", chatID, err)
		if resp != nil {
			errMsg = fmt.Sprintf("Failed to send alert to %s: HTTP %d", chatID, resp.StatusCode)
		}
		logJSON("ERROR", errMsg)
		sendErrorCounter.WithLabelValues(chatID).Inc()
	}
	if resp != nil {
		resp.Body.Close()
	}
}

func main() {
	listen := os.Getenv("LISTEN_ADDR")
	if listen == "" {
		listen = ":8080"
	}
	logJSON("NORMAL", fmt.Sprintf("Starting server on %s", listen))
	http.HandleFunc("/alert", handleAlert)
	http.HandleFunc("/health-check", func(w http.ResponseWriter, _ *http.Request) {
		logJSON("NORMAL", "Health check OK")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(listen, nil); err != nil {
		logJSON("ERROR", fmt.Sprintf("Server error: %v", err))
	}
}
