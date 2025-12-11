package web

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"time"

	"L3.1/producer"
	"L3.1/store"
	"L3.1/types"

	"github.com/google/uuid"
	"github.com/wb-go/wbf/rabbitmq"
)

// Server инкапсулирует настройки HTTP-сервера, шаблоны, RabbitMQ-публикатор и хранилище уведомлений.
type Server struct {
	Port  string
	mux   *http.ServeMux
	Tpl   *template.Template
	Prod  *rabbitmq.Publisher
	Store store.Store
}

// NewServer создаёт и возвращает новый экземпляр веб-сервера.
func NewServer(port string, tpl *template.Template, prod *rabbitmq.Publisher, st store.Store) *Server {
	mux := http.NewServeMux()
	return &Server{
		Port:  port,
		mux:   mux,
		Tpl:   tpl,
		Prod:  prod,
		Store: st,
	}
}

// generateID генерирует уникальный идентификатор уведомления.
func generateID() string {
	return uuid.New().String()
}

// handleNotifyCreate обрабатывает POST-запрос на создание уведомления.
// — парсит входные данные
// — валидирует время
// — сохраняет запись в хранилище
// — отправляет сообщение в RabbitMQ с задержкой until SendAt.
func (s *Server) handleNotifyCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var ntf types.Notification
	if err := json.NewDecoder(r.Body).Decode(&ntf); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	targetTime, err := time.Parse(time.RFC3339, ntf.SendAt)
	if err != nil {
		log.Printf("Ошибка парсинга времени: %v", err)
		http.Error(w, "invalid time format (expect RFC3339)", http.StatusBadRequest)
		return
	}

	now := time.Now()
	if targetTime.Before(now) {
		log.Printf("Время в прошлом. Target: %s, Now: %s", targetTime, now)
		http.Error(w, "time must be in the future", http.StatusBadRequest)
		return
	}

	id := generateID()

	// Сохраняем в хранилище
	if err := s.Store.Save(id, ntf); err != nil {
		http.Error(w, "failed to save", http.StatusInternalServerError)
		return
	}

	envelope := types.NotificationEnvelope{
		ID:      id,
		Message: ntf.Message,
		SendAt:  ntf.SendAt,
	}

	msg, _ := json.Marshal(envelope)

	delay := time.Until(targetTime)

	log.Printf("[WEB] New notification created:\n"+
		"  ID              : %s\n"+
		"  Message         : %s\n"+
		"  SendAt (raw)    : %s\n"+
		"  TargetTime      : %s\n"+
		"  Now             : %s\n"+
		"  Delay           : %s\n"+
		"  RabbitMQ payload: %s\n",
		id,
		ntf.Message,
		ntf.SendAt,
		targetTime.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
		delay,
		string(msg),
	)

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	// Публикуем сообщение в RabbitMQ
	if err := s.Prod.Publish(ctx, msg, "notify.delay", producer.WithTTL(delay)); err != nil {
		http.Error(w, "failed to publish", http.StatusBadGateway)
		return
	}

	// Возвращаем ID созданного уведомления
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{"id": id}); err != nil {
		log.Printf("[WEB] encode error: %v", err)
	}

}

// handleNotifyGet получает уведомление по ID (GET /notify/{id}).
func (s *Server) handleNotifyGet(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/notify/"):]

	ntf, err := s.Store.Get(id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ntf); err != nil {
		log.Printf("[WEB] encode error: %v", err)
	}

}

// handleNotifyCancel отменяет уведомление (DELETE /notify/{id}).
// Помечает уведомление как canceled в хранилище.
func (s *Server) handleNotifyCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Path[len("/notify/"):]
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	if err := s.Store.Cancel(id); err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// IndexHandler рендерит HTML-страницу интерфейса (index.html).
func (s *Server) IndexHandler(w http.ResponseWriter, r *http.Request) {
	if s.Tpl == nil {
		http.Error(w, "template not set", http.StatusInternalServerError)
		return
	}
	if err := s.Tpl.Execute(w, nil); err != nil {
		log.Fatalln(err)
	}
}

// Start запускает HTTP-сервер:
// — регистрирует маршруты
// — обслуживает статику
// — начинает слушать порт
func (s *Server) Start() error {
	fs := http.FileServer(http.Dir("static"))
	s.mux.Handle("/static/", http.StripPrefix("/static/", fs))

	s.mux.HandleFunc("/", s.IndexHandler)

	s.mux.HandleFunc("POST /notify", s.handleNotifyCreate)
	s.mux.HandleFunc("GET /notify/{id}", s.handleNotifyGet)
	s.mux.HandleFunc("DELETE /notify/{id}", s.handleNotifyCancel)

	log.Printf("[WEB] listening on %s\nhttp://localhost:%s/", s.Port, s.Port)
	return http.ListenAndServe(":"+s.Port, s.mux)
}
