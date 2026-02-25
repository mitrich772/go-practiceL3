package handlers

import (
	"bytes"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"contracts/model"
	storageMocks "image-processor/internal/handlers/mocks"
	"image-processor/internal/repo"
	repoMocks "image-processor/internal/repo/mocks"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupHandler(t *testing.T) (*Handler, *repoMocks.MockImageRepo, *storageMocks.MockAPIStorage, *gomock.Controller) {
	t.Helper()
	ctrl := gomock.NewController(t)

	mockRepo := repoMocks.NewMockImageRepo(ctrl)
	mockStorage := storageMocks.NewMockAPIStorage(ctrl)

	h := &Handler{
		Logger:         nil, // будет использовать slog.Default()
		Producer:       nil, // для тестов, не использующих Kafka
		DB:             mockRepo,
		Storage:        mockStorage,
		MaxUploadBytes: 10 << 20,
	}
	// Устанавливаем slog.Default() через New, но без producer
	// Вместо этого просто используем nil logger
	if h.Logger == nil {
		h.Logger = newDefaultLogger()
	}

	return h, mockRepo, mockStorage, ctrl
}

func newDefaultLogger() *slog.Logger {
	return slog.Default().With(
		slog.String("layer", "http"),
		slog.String("component", "handlers_test"),
	)
}

// --- GetImage tests ---

func TestGetImage_NotFound(t *testing.T) {
	h, mockRepo, _, ctrl := setupHandler(t)
	defer ctrl.Finish()

	mockRepo.EXPECT().Get(gomock.Any(), "unknown-id").Return(model.Image{}, repo.ErrNotFound)

	r := chi.NewRouter()
	r.Get("/image/{id}", h.GetImage)

	req := httptest.NewRequest(http.MethodGet, "/image/unknown-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetImage_Processing(t *testing.T) {
	h, mockRepo, _, ctrl := setupHandler(t)
	defer ctrl.Finish()

	mockRepo.EXPECT().Get(gomock.Any(), "proc-id").Return(model.Image{
		ID:     "proc-id",
		Status: model.StatusProcessing,
	}, nil)

	r := chi.NewRouter()
	r.Get("/image/{id}", h.GetImage)

	req := httptest.NewRequest(http.MethodGet, "/image/proc-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), `"processing"`)
}

func TestGetImage_Failed(t *testing.T) {
	h, mockRepo, _, ctrl := setupHandler(t)
	defer ctrl.Finish()

	mockRepo.EXPECT().Get(gomock.Any(), "fail-id").Return(model.Image{
		ID:     "fail-id",
		Status: model.StatusFailed,
	}, nil)

	r := chi.NewRouter()
	r.Get("/image/{id}", h.GetImage)

	req := httptest.NewRequest(http.MethodGet, "/image/fail-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"failed"`)
}

func TestGetImage_Ready(t *testing.T) {
	h, mockRepo, mockStorage, ctrl := setupHandler(t)
	defer ctrl.Finish()

	processedPath := "/processed/ready-id.jpg"
	mockRepo.EXPECT().Get(gomock.Any(), "ready-id").Return(model.Image{
		ID:            "ready-id",
		Status:        model.StatusReady,
		ProcessedPath: &processedPath,
	}, nil)

	fileContent := []byte("fake jpeg data")
	mockStorage.EXPECT().OpenProcessed(gomock.Any(), processedPath).Return(
		io.NopCloser(bytes.NewReader(fileContent)), nil,
	)

	r := chi.NewRouter()
	r.Get("/image/{id}", h.GetImage)

	req := httptest.NewRequest(http.MethodGet, "/image/ready-id", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, fileContent, w.Body.Bytes())
}

// --- DeleteImage tests ---

func TestDeleteImage_NotFound(t *testing.T) {
	h, mockRepo, _, ctrl := setupHandler(t)
	defer ctrl.Finish()

	mockRepo.EXPECT().Get(gomock.Any(), "del-404").Return(model.Image{}, repo.ErrNotFound)

	r := chi.NewRouter()
	r.Delete("/image/{id}", h.DeleteImage)

	req := httptest.NewRequest(http.MethodDelete, "/image/del-404", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteImage_Success(t *testing.T) {
	h, mockRepo, mockStorage, ctrl := setupHandler(t)
	defer ctrl.Finish()

	processedPath := "/processed/del-ok.jpg"
	img := model.Image{
		ID:            "del-ok",
		Status:        model.StatusReady,
		OriginalPath:  "/originals/del-ok.jpg",
		ProcessedPath: &processedPath,
	}

	mockRepo.EXPECT().Get(gomock.Any(), "del-ok").Return(img, nil)
	mockStorage.EXPECT().DeleteOriginal(gomock.Any(), img.OriginalPath).Return(nil)
	mockStorage.EXPECT().DeleteProcessed(gomock.Any(), processedPath).Return(nil)
	mockRepo.EXPECT().Delete(gomock.Any(), "del-ok").Return(nil)

	r := chi.NewRouter()
	r.Delete("/image/{id}", h.DeleteImage)

	req := httptest.NewRequest(http.MethodDelete, "/image/del-ok", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// --- Upload tests (только валидация, без Kafka) ---

func TestUpload_NoFile(t *testing.T) {
	h, _, _, ctrl := setupHandler(t)
	defer ctrl.Finish()

	r := chi.NewRouter()
	r.Post("/image", h.Upload)

	// пустое тело без multipart
	req := httptest.NewRequest(http.MethodPost, "/image", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpload_InvalidMode(t *testing.T) {
	h, _, _, ctrl := setupHandler(t)
	defer ctrl.Finish()

	// Формируем multipart с файлом, но mode=invalid
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.jpg")
	require.NoError(t, err)
	_, _ = part.Write([]byte("fake image"))
	_ = writer.WriteField("mode", "invalid_mode")
	_ = writer.Close()

	r := chi.NewRouter()
	r.Post("/image", h.Upload)

	req := httptest.NewRequest(http.MethodPost, "/image", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpload_InvalidWidth(t *testing.T) {
	h, _, _, ctrl := setupHandler(t)
	defer ctrl.Finish()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.jpg")
	require.NoError(t, err)
	_, _ = part.Write([]byte("fake image"))
	_ = writer.WriteField("mode", "resize")
	_ = writer.WriteField("width", "-5")
	_ = writer.Close()

	r := chi.NewRouter()
	r.Post("/image", h.Upload)

	req := httptest.NewRequest(http.MethodPost, "/image", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// End of tests
