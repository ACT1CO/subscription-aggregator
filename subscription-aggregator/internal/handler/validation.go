package handler

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

var monthYearRegex = regexp.MustCompile(`^(0[1-9]|1[0-2])-\d{4}$`)

func ValidateSubscriptionInput(serviceName string, price int, userID, startDate string) error {
	if serviceName == "" {
		return fmt.Errorf("service_name is required")
	}
	if price <= 0 {
		return fmt.Errorf("price must be a positive integer")
	}
	if _, err := uuid.Parse(userID); err != nil {
		return fmt.Errorf("user_id must be a valid UUID")
	}
	if !monthYearRegex.MatchString(startDate) {
		return fmt.Errorf("start_date must be in MM-YYYY format (e.g., 07-2025)")
	}
	return nil
}

func ValidatePeriodDate(dateStr string) error {
	if !monthYearRegex.MatchString(dateStr) {
		return fmt.Errorf("date must be in MM-YYYY format")
	}
	return nil
}

func isEndDateAfterOrEqual(start, end string) bool {
	startParts := strings.Split(start, "-")
	endParts := strings.Split(end, "-")

	startYear, _ := strconv.Atoi(startParts[1])
	startMonth, _ := strconv.Atoi(startParts[0])
	endYear, _ := strconv.Atoi(endParts[1])
	endMonth, _ := strconv.Atoi(endParts[0])

	if endYear > startYear {
		return true
	}
	if endYear == startYear && endMonth >= startMonth {
		return true
	}
	return false
}
