package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"subscription-aggregator/internal/model"
	"subscription-aggregator/internal/repository"

	"github.com/google/uuid"
)

type SubscriptionHandler struct {
	repo repository.SubscriptionRepository
}

func NewSubscriptionHandler(repo repository.SubscriptionRepository) *SubscriptionHandler {
	return &SubscriptionHandler{repo: repo}
}

func (h *SubscriptionHandler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	var req model.Subscription
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if err := ValidateSubscriptionInput(req.ServiceName, req.Price, req.UserID, req.StartDate); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err.Error()), http.StatusBadRequest)
		return
	}

	if req.EndDate != nil {
		if err := ValidatePeriodDate(*req.EndDate); err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "invalid end_date: %s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		if !isEndDateAfterOrEqual(req.StartDate, *req.EndDate) {
			http.Error(w, `{"error": "end_date must be >= start_date"}`, http.StatusBadRequest)
			return
		}
	}

	if err := h.repo.Create(r.Context(), &req); err != nil {
		slog.Error("Create subscription failed", "error", err)
		http.Error(w, `{"error": "failed to create subscription"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(req); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *SubscriptionHandler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/subscriptions/")
	if id == "" {
		http.Error(w, `{"error": "subscription ID is required"}`, http.StatusBadRequest)
		return
	}

	if _, err := uuid.Parse(id); err != nil {
		http.Error(w, `{"error": "invalid subscription ID format"}`, http.StatusBadRequest)
		return
	}

	sub, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if err.Error() == "subscription not found" {
			http.Error(w, `{"error": "subscription not found"}`, http.StatusNotFound)
			return
		}
		slog.Error("Get subscription failed", "id", id, "error", err)
		http.Error(w, `{"error": "internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(sub); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *SubscriptionHandler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, `{"error": "user_id query parameter is required"}`, http.StatusBadRequest)
		return
	}

	if _, err := uuid.Parse(userID); err != nil {
		http.Error(w, `{"error": "user_id must be a valid UUID"}`, http.StatusBadRequest)
		return
	}

	subs, err := h.repo.ListByUserID(r.Context(), userID)
	if err != nil {
		slog.Error("List subscriptions failed", "user_id", userID, "error", err)
		http.Error(w, `{"error": "failed to list subscriptions"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(subs); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *SubscriptionHandler) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/subscriptions/")
	if id == "" {
		http.Error(w, `{"error": "subscription ID is required"}`, http.StatusBadRequest)
		return
	}

	var req model.Subscription
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if err := ValidateSubscriptionInput(req.ServiceName, req.Price, req.UserID, req.StartDate); err != nil {
		http.Error(w, fmt.Sprintf(`{"error": %q}`, err.Error()), http.StatusBadRequest)
		return
	}

	if req.EndDate != nil {
		if err := ValidatePeriodDate(*req.EndDate); err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "invalid end_date: %s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		if !isEndDateAfterOrEqual(req.StartDate, *req.EndDate) {
			http.Error(w, `{"error": "end_date must be >= start_date"}`, http.StatusBadRequest)
			return
		}
	}

	req.ID = id

	if err := h.repo.Update(r.Context(), id, &req); err != nil {
		if err.Error() == "subscription not found" {
			http.Error(w, `{"error": "subscription not found"}`, http.StatusNotFound)
			return
		}
		slog.Error("Update subscription failed", "id", id, "error", err)
		http.Error(w, `{"error": "failed to update subscription"}`, http.StatusInternalServerError)
		return
	}

	updated, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		slog.Warn("Updated subscription not found after update", "id", id)
		http.Error(w, `{"error": "subscription updated but retrieval failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(updated); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *SubscriptionHandler) DeleteSubscription(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/subscriptions/")
	if id == "" {
		http.Error(w, `{"error": "subscription ID is required"}`, http.StatusBadRequest)
		return
	}

	if _, err := uuid.Parse(id); err != nil {
		http.Error(w, `{"error": "invalid subscription ID format"}`, http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		if err.Error() == "subscription not found" {
			http.Error(w, `{"error": "subscription not found"}`, http.StatusNotFound)
			return
		}
		slog.Error("Delete subscription failed", "id", id, "error", err)
		http.Error(w, `{"error": "failed to delete subscription"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *SubscriptionHandler) GetTotalCost(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	serviceName := r.URL.Query().Get("service_name")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if from == "" || to == "" {
		http.Error(w, `{"error": "'from' and 'to' query parameters are required"}`, http.StatusBadRequest)
		return
	}
	if userID == "" {
		http.Error(w, `{"error": "'user_id' is required"}`, http.StatusBadRequest)
		return
	}

	total, err := h.repo.TotalCost(r.Context(), userID, serviceName, from, to)
	if err != nil {
		if strings.Contains(err.Error(), "invalid") {
			http.Error(w, fmt.Sprintf(`{"error": %q}`, err.Error()), http.StatusBadRequest)
			return
		}
		slog.Error("Total cost calculation failed", "user_id", userID, "error", err)
		http.Error(w, `{"error": "failed to calculate total cost"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]int{"total": total}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
