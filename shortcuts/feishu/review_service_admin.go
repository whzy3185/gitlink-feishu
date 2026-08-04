package feishu

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	reviewAdminListSchema  = "review.admin-list/v1"
	reviewAdminErrorSchema = "review.admin-error/v1"
)

type reviewAdminListResponse struct {
	SchemaVersion string                   `json:"schema_version"`
	GeneratedAt   string                   `json:"generated_at"`
	Items         []map[string]interface{} `json:"items"`
	NextCursor    string                   `json:"next_cursor"`
}

type reviewAdminListSpec struct {
	table          string
	idExpression   string
	statusExpr     string
	allowedFilters map[string]string
}

var reviewAdminListSpecs = map[string]reviewAdminListSpec{
	"service-instances":         {"review_service_instances", "instance_id", "status", map[string]string{"status": "status"}},
	"installations":             {"gitlink_installations", "installation_id", "CAST(enabled AS TEXT)", map[string]string{"installation": "installation_id"}},
	"bindings":                  {"chat_repository_bindings", "chat_id || ':' || repository", "CAST(enabled AS TEXT)", map[string]string{"installation": "installation_id", "repository": "repository"}},
	"subscriptions":             {"review_chat_subscriptions", "subscription_id", "CAST(enabled AS TEXT)", map[string]string{"installation": "installation_id", "repository": "repository"}},
	"config-revisions":          {"review_configuration_revisions", "revision_id", "'applied'", map[string]string{}},
	"events":                    {"review_event_inbox", "event_id", "status", map[string]string{"status": "status", "installation": "installation_id", "repository": "repository"}},
	"event-routes":              {"review_event_routes", "event_id || ':' || subscription_id", "route_status", map[string]string{"status": "route_status", "installation": "installation_id", "repository": "repository"}},
	"jobs":                      {"review_gateway_jobs", "job_id", "status", map[string]string{"status": "status", "repository": "repository", "queue_class": "queue_class"}},
	"job-consumers":             {"review_job_consumers", "consumer_id", "status", map[string]string{"status": "status"}},
	"operation-attempts":        {"review_operation_attempts", "attempt_id", "mutation_status", map[string]string{"status": "mutation_status"}},
	"dead-letters":              {"review_dead_letters", "dead_letter_id", "status", map[string]string{"status": "status", "repository": "repository", "queue_class": "queue_class"}},
	"operation-reconciliations": {"review_operation_reconciliation_tasks", "reconciliation_id", "status", map[string]string{"status": "status", "resource_type": "resource_type"}},
	"resource-migrations":       {"review_resource_migrations", "migration_id", "status", map[string]string{"status": "status", "installation": "installation_id", "repository": "repository", "resource_type": "resource_type"}},
	"projection-status":         {"review_resource_projection_status", "projection_key", "status", map[string]string{"status": "status", "installation": "installation_id", "repository": "repository", "resource_type": "resource_type"}},
	"maintenance-runs":          {"review_maintenance_runs", "run_id", "status", map[string]string{"status": "status"}},
	"audit":                     {"review_gateway_configuration_audit", "audit_id", "'applied'", map[string]string{}},
}

func (s *ReviewService) handleMetrics(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !s.authorizeAdmin(request) {
		writeReviewAdminError(writer, http.StatusUnauthorized, "unauthorized", "administration bearer token is required")
		return
	}
	if s.Metrics == nil || !s.Config.MetricsEnabled {
		writeReviewAdminError(writer, http.StatusNotFound, "metrics_disabled", "metrics are disabled")
		return
	}
	text, err := s.Metrics.Render(request.Context())
	if err != nil {
		writeReviewAdminError(writer, http.StatusServiceUnavailable, "metrics_unavailable", "metrics are temporarily unavailable")
		return
	}
	writer.Header().Set("Content-Type", "text/plain; version=0.0.4")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte(text))
}

func (s *ReviewService) handleAdmin(writer http.ResponseWriter, request *http.Request) {
	if !s.authorizeAdmin(request) {
		writeReviewAdminError(writer, http.StatusUnauthorized, "unauthorized", "administration bearer token is required")
		return
	}
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		writeReviewAdminError(writer, http.StatusMethodNotAllowed, "method_not_allowed", "administration API is read-only")
		return
	}
	endpoint := strings.Trim(strings.TrimPrefix(request.URL.Path, "/admin/v1/"), "/")
	switch endpoint {
	case "summary":
		s.handleAdminSummary(writer, request)
	case "components":
		s.handleAdminComponents(writer, request)
	case "operations":
		s.handleAdminOperations(writer, request)
	default:
		spec, ok := reviewAdminListSpecs[endpoint]
		if !ok {
			writeReviewAdminError(writer, http.StatusNotFound, "not_found", "administration resource not found")
			return
		}
		response, err := s.queryAdminList(request.Context(), request, spec)
		if err != nil {
			s.writeAdminQueryError(writer, err)
			return
		}
		writeReviewServiceJSON(writer, http.StatusOK, response)
	}
}

func (s *ReviewService) authorizeAdmin(request *http.Request) bool {
	if s == nil || len(s.adminToken) == 0 {
		return false
	}
	header := strings.TrimSpace(request.Header.Get("Authorization"))
	if !strings.HasPrefix(header, "Bearer ") {
		return false
	}
	return reviewAdminTokenEqual([]byte(strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))), s.adminToken)
}

func reviewAdminTokenEqual(provided, expected []byte) bool {
	if len(provided) != len(expected) || len(expected) == 0 {
		return false
	}
	return subtle.ConstantTimeCompare(provided, expected) == 1
}

func (s *ReviewService) handleAdminSummary(writer http.ResponseWriter, request *http.Request) {
	snapshot := s.Status.Snapshot()
	schema, _ := s.Store.CurrentSchemaVersion(request.Context())
	s.mu.Lock()
	revision := s.configRevision
	s.mu.Unlock()
	counts := map[string]int{}
	for name, query := range map[string]string{
		"jobs":            `SELECT COUNT(*) FROM review_gateway_jobs WHERE status IN ('queued','running','retry_scheduled')`,
		"operations":      `SELECT COUNT(*) FROM review_operations WHERE status IN ('pending','leased','writing','retry_scheduled','unknown','needs_reconciliation')`,
		"dead_letters":    `SELECT COUNT(*) FROM review_dead_letters WHERE status='open'`,
		"reconciliations": `SELECT COUNT(*) FROM review_operation_reconciliation_tasks WHERE status IN ('pending','checking','manual_required')`,
	} {
		var count int
		if err := s.Store.db.QueryRowContext(request.Context(), query).Scan(&count); err == nil {
			counts[name] = count
		}
	}
	writeReviewServiceJSON(writer, http.StatusOK, map[string]interface{}{"schema_version": "review.admin-summary/v1", "generated_at": reviewGatewayTimestamp(s.now()), "service_state": snapshot.State, "admission": snapshot.Admission, "schema_revision": schema, "config_revision": revision, "components": len(snapshot.Components), "counts": counts})
}

func (s *ReviewService) handleAdminComponents(writer http.ResponseWriter, request *http.Request) {
	limit, offset, err := parseReviewAdminPagination(request)
	if err != nil {
		s.writeAdminQueryError(writer, err)
		return
	}
	components := s.Status.Snapshot().Components
	end := offset + limit
	if end > len(components) {
		end = len(components)
	}
	items := []map[string]interface{}{}
	if offset < len(components) {
		for _, component := range components[offset:end] {
			items = append(items, map[string]interface{}{"component": component.Component, "worker_index": component.WorkerIndex, "required": component.Required, "status": component.Status, "last_heartbeat_at": component.LastHeartbeatAt, "last_error_summary": component.LastErrorSummary})
		}
	}
	next := ""
	if end < len(components) {
		next = encodeReviewAdminCursor(end)
	}
	writeReviewServiceJSON(writer, http.StatusOK, newReviewAdminList(items, next, s.now()))
}

func (s *ReviewService) handleAdminOperations(writer http.ResponseWriter, request *http.Request) {
	allowed := map[string]string{"status": "status", "installation": "installation_id", "repository": "repository", "resource_type": "resource_type", "queue_class": "queue_class"}
	limit, offset, where, args, err := parseReviewAdminQuery(request, allowed)
	if err != nil {
		s.writeAdminQueryError(writer, err)
		return
	}
	query := `SELECT operation_id,operation_kind,queue_class,resource_type,status,error_class,error_code,
		remote_id,requires_reconciliation,attempt_count,created_at,updated_at FROM review_operations` + where + ` ORDER BY created_at,operation_id LIMIT ? OFFSET ?`
	args = append(args, limit+1, offset)
	rows, err := s.Store.db.QueryContext(request.Context(), query, args...)
	if err != nil {
		s.writeAdminQueryError(writer, err)
		return
	}
	defer rows.Close()
	items := []map[string]interface{}{}
	for rows.Next() {
		var operationID, kind, queueClass, resourceType, status, errorClass, errorCode, remoteID, createdAt, updatedAt string
		var reconciliation, attempts int
		if err := rows.Scan(&operationID, &kind, &queueClass, &resourceType, &status, &errorClass, &errorCode, &remoteID, &reconciliation, &attempts, &createdAt, &updatedAt); err != nil {
			s.writeAdminQueryError(writer, err)
			return
		}
		items = append(items, map[string]interface{}{"operation_id_hash": reviewGatewayHashIdentifier(operationID), "operation_kind": kind, "queue_class": queueClass, "resource_type": resourceType, "status": status, "error_class": errorClass, "error_code": errorCode, "remote_id_hash": reviewGatewayHashIdentifier(remoteID), "requires_reconciliation": reconciliation != 0, "attempt_count": attempts, "created_at": createdAt, "updated_at": updatedAt})
	}
	next := ""
	if len(items) > limit {
		items = items[:limit]
		next = encodeReviewAdminCursor(offset + limit)
	}
	writeReviewServiceJSON(writer, http.StatusOK, newReviewAdminList(items, next, s.now()))
}

func (s *ReviewService) queryAdminList(ctx context.Context, request *http.Request, spec reviewAdminListSpec) (reviewAdminListResponse, error) {
	limit, offset, where, args, err := parseReviewAdminQuery(request, spec.allowedFilters)
	if err != nil {
		return reviewAdminListResponse{}, err
	}
	query := `SELECT ` + spec.idExpression + `,` + spec.statusExpr + ` FROM ` + spec.table + where + ` ORDER BY 1 LIMIT ? OFFSET ?`
	args = append(args, limit+1, offset)
	rows, err := s.Store.db.QueryContext(ctx, query, args...)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return newReviewAdminList(nil, "", s.now()), nil
		}
		return reviewAdminListResponse{}, err
	}
	defer rows.Close()
	items := []map[string]interface{}{}
	for rows.Next() {
		var identifier, status string
		if err := rows.Scan(&identifier, &status); err != nil {
			return reviewAdminListResponse{}, err
		}
		items = append(items, map[string]interface{}{"id_hash": reviewGatewayHashIdentifier(identifier), "status": status})
	}
	next := ""
	if len(items) > limit {
		items = items[:limit]
		next = encodeReviewAdminCursor(offset + limit)
	}
	return newReviewAdminList(items, next, s.now()), rows.Err()
}

func parseReviewAdminPagination(request *http.Request) (int, int, error) {
	limit, offset, _, _, err := parseReviewAdminQuery(request, map[string]string{})
	return limit, offset, err
}

func parseReviewAdminQuery(request *http.Request, allowed map[string]string) (int, int, string, []interface{}, error) {
	for key := range request.URL.Query() {
		if key != "limit" && key != "cursor" {
			if _, ok := allowed[key]; !ok {
				return 0, 0, "", nil, fmt.Errorf("unsupported_filter:%s", key)
			}
		}
	}
	limit := 50
	if raw := strings.TrimSpace(request.URL.Query().Get("limit")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > 200 {
			return 0, 0, "", nil, fmt.Errorf("invalid_limit")
		}
		limit = value
	}
	offset := 0
	if raw := strings.TrimSpace(request.URL.Query().Get("cursor")); raw != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(raw)
		if err != nil {
			return 0, 0, "", nil, fmt.Errorf("invalid_cursor")
		}
		value, err := strconv.Atoi(string(decoded))
		if err != nil || value < 0 {
			return 0, 0, "", nil, fmt.Errorf("invalid_cursor")
		}
		offset = value
	}
	whereParts := []string{}
	args := []interface{}{}
	for _, key := range []string{"status", "installation", "repository", "resource_type", "queue_class"} {
		column, ok := allowed[key]
		value := strings.TrimSpace(request.URL.Query().Get(key))
		if !ok || value == "" {
			continue
		}
		whereParts = append(whereParts, column+"=?")
		args = append(args, value)
	}
	where := ""
	if len(whereParts) > 0 {
		where = " WHERE " + strings.Join(whereParts, " AND ")
	}
	return limit, offset, where, args, nil
}

func encodeReviewAdminCursor(offset int) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

func newReviewAdminList(items []map[string]interface{}, cursor string, now time.Time) reviewAdminListResponse {
	if items == nil {
		items = []map[string]interface{}{}
	}
	return reviewAdminListResponse{SchemaVersion: reviewAdminListSchema, GeneratedAt: reviewGatewayTimestamp(now), Items: items, NextCursor: cursor}
}

func (s *ReviewService) writeAdminQueryError(writer http.ResponseWriter, err error) {
	code, message, status := "admin_query_failed", "administration query failed", http.StatusInternalServerError
	if strings.Contains(err.Error(), "invalid_cursor") {
		code, message, status = "invalid_cursor", "invalid pagination cursor", http.StatusBadRequest
	} else if strings.Contains(err.Error(), "invalid_limit") {
		code, message, status = "invalid_limit", "invalid pagination limit", http.StatusBadRequest
	} else if strings.Contains(err.Error(), "unsupported_filter") {
		code, message, status = "unsupported_filter", "filter is not supported for this resource", http.StatusBadRequest
	} else if err == sql.ErrNoRows {
		code, message, status = "not_found", "administration resource not found", http.StatusNotFound
	}
	writeReviewAdminError(writer, status, code, message)
}

func writeReviewAdminError(writer http.ResponseWriter, status int, code, message string) {
	writeReviewServiceJSON(writer, status, map[string]interface{}{"schema_version": reviewAdminErrorSchema, "code": code, "message": message})
}
