package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fxrates/internal/domain"
	"fxrates/internal/rate"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockValidator struct{ mock.Mock }

func (m *MockValidator) ValidateCodes(base, quote string) error {
	args := m.Called(base, quote)
	return args.Error(0)
}

func (m *MockValidator) SupportedCodes() []string {
	args := m.Called()
	codes, _ := args.Get(0).([]string)
	return codes
}

type MockService struct{ mock.Mock }

func (m *MockService) ScheduleUpdate(ctx context.Context, base, quote string) (uuid.UUID, error) {
	args := m.Called(ctx, base, quote)
	id, _ := args.Get(0).(uuid.UUID)
	return id, args.Error(1)
}

func (m *MockService) GetByUpdateID(ctx context.Context, id uuid.UUID) (rate.View, error) {
	args := m.Called(ctx, id)
	v, _ := args.Get(0).(rate.View)
	return v, args.Error(1)
}

func (m *MockService) GetByCodes(ctx context.Context, base, quote string) (rate.View, error) {
	args := m.Called(ctx, base, quote)
	v, _ := args.Get(0).(rate.View)
	return v, args.Error(1)
}

type errorJSON struct {
	Error string `json:"error"`
}

// --- GetByCodes ---

func TestHandler_GetByCodes_ValidationErrors(t *testing.T) {
	cases := []struct {
		name         string
		validatorErr error
		wantMsg      string
	}{
		{name: "base required", validatorErr: rate.ErrBaseRequired, wantMsg: rate.ErrBaseRequired.Error()},
		{name: "quote required", validatorErr: rate.ErrQuoteRequired, wantMsg: rate.ErrQuoteRequired.Error()},
		{name: "same codes", validatorErr: rate.ErrSameCodes, wantMsg: rate.ErrSameCodes.Error()},
		{name: "base unsupported", validatorErr: rate.ErrBaseUnsupported, wantMsg: rate.ErrBaseUnsupported.Error()},
		{name: "quote unsupported", validatorErr: rate.ErrQuoteUnsupported, wantMsg: rate.ErrQuoteUnsupported.Error()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockValidator := new(MockValidator)
			mockService := new(MockService)
			h := NewRateHandler(mockValidator, mockService)

			req := httptest.NewRequest(http.MethodGet, "/rates/usd/eur", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("base", " usd ")
			rctx.URLParams.Add("quote", " eur")
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			rr := httptest.NewRecorder()

			mockValidator.On("ValidateCodes", "USD", "EUR").Return(tc.validatorErr).Once()

			h.GetByCodes(rr, req)

			require.Equal(t, http.StatusBadRequest, rr.Code)
			require.Equal(t, "application/json", rr.Header().Get("Content-Type"))
			var ej errorJSON
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &ej))
			require.Equal(t, tc.wantMsg, ej.Error)

			mockService.AssertNotCalled(t, "GetByCodes", mock.Anything, mock.Anything, mock.Anything)
			mockValidator.AssertExpectations(t)
		})
	}
}

func TestHandler_GetByCodes_NotFound(t *testing.T) {
	mockValidator := new(MockValidator)
	mockService := new(MockService)
	h := NewRateHandler(mockValidator, mockService)

	req := httptest.NewRequest(http.MethodGet, "/rates/usd/eur", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("base", "usd")
	rctx.URLParams.Add("quote", "eur")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	mockValidator.On("ValidateCodes", "USD", "EUR").Return(nil).Once()
	mockService.On("GetByCodes", mock.Anything, "USD", "EUR").Return(rate.View{}, domain.ErrRateNotFound).Once()

	h.GetByCodes(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
	require.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	var ej errorJSON
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &ej))
	require.Equal(t, "rate not found", ej.Error)
	mockValidator.AssertExpectations(t)
	mockService.AssertExpectations(t)
}

func TestHandler_GetByCodes_InternalError(t *testing.T) {
	mockValidator := new(MockValidator)
	mockService := new(MockService)
	h := NewRateHandler(mockValidator, mockService)

	req := httptest.NewRequest(http.MethodGet, "/rates/usd/eur", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("base", "usd")
	rctx.URLParams.Add("quote", "eur")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	mockValidator.On("ValidateCodes", "USD", "EUR").Return(nil).Once()
	mockService.On("GetByCodes", mock.Anything, "USD", "EUR").Return(rate.View{}, errors.New("boom")).Once()

	h.GetByCodes(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
	require.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	var ej errorJSON
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &ej))
	require.Equal(t, "ups, couldn't get rate by codes this time", ej.Error)
	mockValidator.AssertExpectations(t)
	mockService.AssertExpectations(t)
}

func TestHandler_GetByCodes_Success(t *testing.T) {
	mockValidator := new(MockValidator)
	mockService := new(MockService)
	h := NewRateHandler(mockValidator, mockService)

	req := httptest.NewRequest(http.MethodGet, "/rates/usd/eur", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("base", " usd ")
	rctx.URLParams.Add("quote", " eur ")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	now := time.Date(2025, 1, 2, 15, 4, 5, 0, time.UTC)
	val := 0.9231
	view := rate.View{Base: "USD", Quote: "EUR", Value: &val, UpdatedAt: &now}

	mockValidator.On("ValidateCodes", "USD", "EUR").Return(nil).Once()
	mockService.On("GetByCodes", mock.Anything, "USD", "EUR").Return(view, nil).Once()

	h.GetByCodes(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	var res GetByCodesResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &res))
	require.Equal(t, "USD", res.Base)
	require.Equal(t, "EUR", res.Quote)
	require.InDelta(t, 0.9231, res.Value, 1e-9)
	require.True(t, res.UpdatedAt.Equal(now))
	mockValidator.AssertExpectations(t)
	mockService.AssertExpectations(t)
}

// --- GetByUpdateID ---

func TestHandler_GetByUpdateID_InvalidID(t *testing.T) {
	mockValidator := new(MockValidator)
	mockService := new(MockService)
	h := NewRateHandler(mockValidator, mockService)

	req := httptest.NewRequest(http.MethodGet, "/rates/updates/not-a-uuid", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "not-a-uuid")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	h.GetByUpdateID(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	var ej errorJSON
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &ej))
	require.Equal(t, "invalid update ID format", ej.Error)
	mockService.AssertNotCalled(t, "GetByUpdateID", mock.Anything, mock.Anything)
}

func TestHandler_GetByUpdateID_NotFound(t *testing.T) {
	mockValidator := new(MockValidator)
	mockService := new(MockService)
	h := NewRateHandler(mockValidator, mockService)

	updateID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/rates/updates/"+updateID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", updateID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	mockService.On("GetByUpdateID", mock.Anything, updateID).Return(rate.View{}, domain.ErrRateNotFound).Once()

	h.GetByUpdateID(rr, req)

	require.Equal(t, http.StatusNotFound, rr.Code)
	var ej errorJSON
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &ej))
	require.Equal(t, "rate update not found", ej.Error)
	mockService.AssertExpectations(t)
}

func TestHandler_GetByUpdateID_InternalError(t *testing.T) {
	mockService := new(MockService)
	h := NewRateHandler(new(MockValidator), mockService)

	updateID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/rates/updates/"+updateID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", updateID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	mockService.On("GetByUpdateID", mock.Anything, updateID).Return(rate.View{}, errors.New("db failed")).Once()

	h.GetByUpdateID(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
	var ej errorJSON
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &ej))
	require.Equal(t, "ups, couldn't get rate by update id this time", ej.Error)
	mockService.AssertExpectations(t)
}

func TestHandler_GetByUpdateID_Pending(t *testing.T) {
	mockService := new(MockService)
	h := NewRateHandler(new(MockValidator), mockService)

	updateID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/rates/updates/"+updateID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", updateID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	view := rate.View{Base: "USD", Quote: "EUR", Status: domain.StatusPending}
	mockService.On("GetByUpdateID", mock.Anything, updateID).Return(view, nil).Once()

	h.GetByUpdateID(rr, req)

	require.Equal(t, http.StatusAccepted, rr.Code)
	require.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	var res GetByUpdateIDPending
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &res))
	require.Equal(t, updateID.String(), res.UpdateID)
	require.Equal(t, "USD", res.Base)
	require.Equal(t, "EUR", res.Quote)
	require.Equal(t, domain.StatusPending, res.Status)
	mockService.AssertExpectations(t)
}

func TestHandler_GetByUpdateID_Applied(t *testing.T) {
	mockService := new(MockService)
	h := NewRateHandler(new(MockValidator), mockService)

	updateID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/rates/updates/"+updateID.String(), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", updateID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rr := httptest.NewRecorder()

	val := 1.01
	now := time.Date(2025, 1, 2, 15, 4, 5, 0, time.UTC)
	view := rate.View{Base: "USD", Quote: "EUR", Status: domain.StatusApplied, Value: &val, UpdatedAt: &now}
	mockService.On("GetByUpdateID", mock.Anything, updateID).Return(view, nil).Once()

	h.GetByUpdateID(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	var res GetByUpdateIDApplied
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &res))
	require.Equal(t, updateID.String(), res.UpdateID)
	require.Equal(t, "USD", res.Base)
	require.Equal(t, "EUR", res.Quote)
	require.Equal(t, domain.StatusApplied, res.Status)
	require.InDelta(t, 1.01, res.Value, 1e-9)
	require.True(t, res.UpdatedAt.Equal(now))
	mockService.AssertExpectations(t)
}

// --- ScheduleUpdate ---

func TestHandler_ScheduleUpdate_InvalidJSON(t *testing.T) {
	mockValidator := new(MockValidator)
	mockService := new(MockService)
	h := NewRateHandler(mockValidator, mockService)

	req := httptest.NewRequest(http.MethodPost, "/rates/updates", bytes.NewBufferString("{"))
	rr := httptest.NewRecorder()

	h.ScheduleUpdate(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	var ej errorJSON
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &ej))
	require.Equal(t, "invalid request body", ej.Error)
	mockValidator.AssertNotCalled(t, "ValidateCodes", mock.Anything, mock.Anything)
	mockService.AssertNotCalled(t, "ScheduleUpdate", mock.Anything, mock.Anything, mock.Anything)
}

func TestHandler_ScheduleUpdate_UnknownField(t *testing.T) {
	mockValidator := new(MockValidator)
	mockService := new(MockService)
	h := NewRateHandler(mockValidator, mockService)

	body := `{"base":"USD","quote":"EUR","extra":1}`
	req := httptest.NewRequest(http.MethodPost, "/rates/updates", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	h.ScheduleUpdate(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	var ej errorJSON
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &ej))
	require.Equal(t, "invalid request body", ej.Error)
	mockValidator.AssertNotCalled(t, "ValidateCodes", mock.Anything, mock.Anything)
	mockService.AssertNotCalled(t, "ScheduleUpdate", mock.Anything, mock.Anything, mock.Anything)
}

func TestHandler_ScheduleUpdate_BodyTooLarge(t *testing.T) {
	mockValidator := new(MockValidator)
	mockService := new(MockService)
	h := NewRateHandler(mockValidator, mockService)

	// Build a single JSON object whose size exceeds 256 bytes
	longBase := make([]byte, 270)
	for i := range longBase {
		longBase[i] = 'A'
	}
	body := `{"base":"` + string(longBase) + `","quote":"EUR"}`
	req := httptest.NewRequest(http.MethodPost, "/rates/updates", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	h.ScheduleUpdate(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code)
	var ej errorJSON
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &ej))
	require.Equal(t, "invalid request body", ej.Error)
	mockValidator.AssertNotCalled(t, "ValidateCodes", mock.Anything, mock.Anything)
	mockService.AssertNotCalled(t, "ScheduleUpdate", mock.Anything, mock.Anything, mock.Anything)
}

func TestHandler_ScheduleUpdate_ValidationErrors(t *testing.T) {
	cases := []struct {
		name         string
		validatorErr error
		wantMsg      string
	}{
		{name: "base required", validatorErr: rate.ErrBaseRequired, wantMsg: rate.ErrBaseRequired.Error()},
		{name: "quote required", validatorErr: rate.ErrQuoteRequired, wantMsg: rate.ErrQuoteRequired.Error()},
		{name: "same codes", validatorErr: rate.ErrSameCodes, wantMsg: rate.ErrSameCodes.Error()},
		{name: "base unsupported", validatorErr: rate.ErrBaseUnsupported, wantMsg: rate.ErrBaseUnsupported.Error()},
		{name: "quote unsupported", validatorErr: rate.ErrQuoteUnsupported, wantMsg: rate.ErrQuoteUnsupported.Error()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockValidator := new(MockValidator)
			mockService := new(MockService)
			h := NewRateHandler(mockValidator, mockService)

			body := `{"base":" usd ","quote":" eur"}`
			req := httptest.NewRequest(http.MethodPost, "/rates/updates", bytes.NewBufferString(body))
			rr := httptest.NewRecorder()

			mockValidator.On("ValidateCodes", "USD", "EUR").Return(tc.validatorErr).Once()

			h.ScheduleUpdate(rr, req)

			require.Equal(t, http.StatusBadRequest, rr.Code)
			var ej errorJSON
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &ej))
			require.Equal(t, tc.wantMsg, ej.Error)
			mockService.AssertNotCalled(t, "ScheduleUpdate", mock.Anything, mock.Anything, mock.Anything)
			mockValidator.AssertExpectations(t)
		})
	}
}

func TestHandler_ScheduleUpdate_ServiceError(t *testing.T) {
	mockValidator := new(MockValidator)
	mockService := new(MockService)
	h := NewRateHandler(mockValidator, mockService)

	body := `{"base":" usd ","quote":" eur"}`
	req := httptest.NewRequest(http.MethodPost, "/rates/updates", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	mockValidator.On("ValidateCodes", "USD", "EUR").Return(nil).Once()
	mockService.On("ScheduleUpdate", mock.Anything, "USD", "EUR").Return(uuid.Nil, errors.New("failed")).Once()

	h.ScheduleUpdate(rr, req)

	require.Equal(t, http.StatusInternalServerError, rr.Code)
	require.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	var ej errorJSON
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &ej))
	require.Equal(t, "failed to schedule rate update", ej.Error)
	mockValidator.AssertExpectations(t)
	mockService.AssertExpectations(t)
}

func TestHandler_ScheduleUpdate_Success(t *testing.T) {
	mockValidator := new(MockValidator)
	mockService := new(MockService)
	h := NewRateHandler(mockValidator, mockService)

	body := `{"base":" usd ","quote":" eur"}`
	req := httptest.NewRequest(http.MethodPost, "/rates/updates", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()

	updateID := uuid.New()
	mockValidator.On("ValidateCodes", "USD", "EUR").Return(nil).Once()
	mockService.On("ScheduleUpdate", mock.Anything, "USD", "EUR").Return(updateID, nil).Once()

	h.ScheduleUpdate(rr, req)

	require.Equal(t, http.StatusAccepted, rr.Code)
	require.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	var res ScheduleUpdateResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &res))
	require.Equal(t, updateID.String(), res.UpdateID)
	mockValidator.AssertExpectations(t)
	mockService.AssertExpectations(t)
}

func TestHandler_GetSupportedCodes(t *testing.T) {
	mockValidator := new(MockValidator)
	mockService := new(MockService)
	h := NewRateHandler(mockValidator, mockService)

	mockValidator.On("SupportedCodes").Return([]string{"USD", "EUR"}).Once()

	req := httptest.NewRequest(http.MethodGet, "/rates/supported-currencies", nil)
	rr := httptest.NewRecorder()

	h.GetSupportedCodes(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	require.Equal(t, "application/json", rr.Header().Get("Content-Type"))
	var resp GetSupportedCodesResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	require.Equal(t, []string{"USD", "EUR"}, resp.Codes)

	mockValidator.AssertExpectations(t)
	mockService.AssertExpectations(t)
}
