package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/clario360/platform/internal/acta/model"
)

func (s *Store) CreateCommittee(ctx context.Context, q DBTX, committee *model.Committee) error {
	metadata, err := marshalJSON(committee.Metadata)
	if err != nil {
		return fmt.Errorf("marshal committee metadata: %w", err)
	}
	_, err = q.Exec(ctx, `
		INSERT INTO committees (
			id, tenant_id, name, type, description, chair_user_id, vice_chair_user_id,
			secretary_user_id, meeting_frequency, quorum_percentage, quorum_type,
			quorum_fixed_count, charter, established_date, dissolution_date,
			status, tags, metadata, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11,
			$12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21
		)`,
		committee.ID,
		committee.TenantID,
		committee.Name,
		committee.Type,
		committee.Description,
		committee.ChairUserID,
		nullableUUID(committee.ViceChairUserID),
		nullableUUID(committee.SecretaryUserID),
		committee.MeetingFrequency,
		committee.QuorumPercentage,
		committee.QuorumType,
		committee.QuorumFixedCount,
		committee.Charter,
		dateOnly(committee.EstablishedDate),
		dateOnly(committee.DissolutionDate),
		committee.Status,
		committee.Tags,
		metadata,
		committee.CreatedBy,
		committee.CreatedAt,
		committee.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create committee: %w", err)
	}
	return nil
}

func (s *Store) UpdateCommittee(ctx context.Context, q DBTX, committee *model.Committee) error {
	metadata, err := marshalJSON(committee.Metadata)
	if err != nil {
		return fmt.Errorf("marshal committee metadata: %w", err)
	}
	tag, err := q.Exec(ctx, `
		UPDATE committees
		SET name = $3,
		    type = $4,
		    description = $5,
		    chair_user_id = $6,
		    vice_chair_user_id = $7,
		    secretary_user_id = $8,
		    meeting_frequency = $9,
		    quorum_percentage = $10,
		    quorum_type = $11,
		    quorum_fixed_count = $12,
		    charter = $13,
		    established_date = $14,
		    dissolution_date = $15,
		    status = $16,
		    tags = $17,
		    metadata = $18,
		    updated_at = $19
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		committee.TenantID,
		committee.ID,
		committee.Name,
		committee.Type,
		committee.Description,
		committee.ChairUserID,
		nullableUUID(committee.ViceChairUserID),
		nullableUUID(committee.SecretaryUserID),
		committee.MeetingFrequency,
		committee.QuorumPercentage,
		committee.QuorumType,
		committee.QuorumFixedCount,
		committee.Charter,
		dateOnly(committee.EstablishedDate),
		dateOnly(committee.DissolutionDate),
		committee.Status,
		committee.Tags,
		metadata,
		committee.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update committee: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return notFoundError("committee", committee.ID)
	}
	return nil
}

func (s *Store) SoftDeleteCommittee(ctx context.Context, tenantID, committeeID uuid.UUID, deletedAt time.Time) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE committees
		SET deleted_at = $3, updated_at = $3
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, committeeID, deletedAt,
	)
	if err != nil {
		return fmt.Errorf("soft delete committee: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return notFoundError("committee", committeeID)
	}
	return nil
}

func (s *Store) GetCommittee(ctx context.Context, tenantID, committeeID uuid.UUID) (*model.Committee, error) {
	row := s.db.QueryRow(ctx, `
		SELECT id, tenant_id, name, type, description, chair_user_id, vice_chair_user_id,
		       secretary_user_id, meeting_frequency, quorum_percentage, quorum_type,
		       quorum_fixed_count, charter, established_date, dissolution_date,
		       status, tags, metadata, created_by, created_at, updated_at, deleted_at
		FROM committees
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, committeeID,
	)
	return scanCommittee(row)
}

func (s *Store) GetCommitteeByName(ctx context.Context, tenantID uuid.UUID, name string) (*model.Committee, error) {
	row := s.db.QueryRow(ctx, `
		SELECT id, tenant_id, name, type, description, chair_user_id, vice_chair_user_id,
		       secretary_user_id, meeting_frequency, quorum_percentage, quorum_type,
		       quorum_fixed_count, charter, established_date, dissolution_date,
		       status, tags, metadata, created_by, created_at, updated_at, deleted_at
		FROM committees
		WHERE tenant_id = $1 AND lower(name) = lower($2) AND deleted_at IS NULL`,
		tenantID, name,
	)
	return scanCommittee(row)
}

func (s *Store) ListCommittees(ctx context.Context, tenantID uuid.UUID, search string, page, perPage int) ([]model.Committee, int, error) {
	offset := (page - 1) * perPage
	search = strings.TrimSpace(search)
	countQuery := `SELECT COUNT(*) FROM committees WHERE tenant_id = $1 AND deleted_at IS NULL`
	args := []any{tenantID}
	if search != "" {
		countQuery += ` AND (name ILIKE $2 OR description ILIKE $2)`
		args = append(args, "%"+search+"%")
	}
	var total int
	if err := s.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count committees: %w", err)
	}

	dataQuery := `
		SELECT c.id, c.tenant_id, c.name, c.type, c.description, c.chair_user_id, c.vice_chair_user_id,
		       c.secretary_user_id, c.meeting_frequency, c.quorum_percentage, c.quorum_type,
		       c.quorum_fixed_count, c.charter, c.established_date, c.dissolution_date,
		       c.status, c.tags, c.metadata, c.created_by, c.created_at, c.updated_at, c.deleted_at,
		       COALESCE(member_counts.active_members, 0)
		FROM committees c
		LEFT JOIN (
			SELECT committee_id, COUNT(*) FILTER (WHERE active = true AND role != 'observer') AS active_members
			FROM committee_members
			GROUP BY committee_id
		) member_counts ON member_counts.committee_id = c.id
		WHERE c.tenant_id = $1 AND c.deleted_at IS NULL`
	if search != "" {
		dataQuery += ` AND (c.name ILIKE $2 OR c.description ILIKE $2)`
	}
	dataQuery += fmt.Sprintf(" ORDER BY c.name ASC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, perPage, offset)

	rows, err := s.db.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list committees: %w", err)
	}
	defer rows.Close()

	items := make([]model.Committee, 0, perPage)
	for rows.Next() {
		committee, memberCount, err := scanCommitteeWithMemberCount(rows)
		if err != nil {
			return nil, 0, err
		}
		committee.Stats = &model.CommitteeStats{ActiveMembers: memberCount}
		items = append(items, *committee)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate committees: %w", err)
	}
	return items, total, nil
}

func (s *Store) ListActiveCommittees(ctx context.Context, tenantID *uuid.UUID) ([]model.Committee, error) {
	query := `
		SELECT id, tenant_id, name, type, description, chair_user_id, vice_chair_user_id,
		       secretary_user_id, meeting_frequency, quorum_percentage, quorum_type,
		       quorum_fixed_count, charter, established_date, dissolution_date,
		       status, tags, metadata, created_by, created_at, updated_at, deleted_at
		FROM committees
		WHERE status = 'active' AND deleted_at IS NULL`
	args := []any{}
	if tenantID != nil {
		query += ` AND tenant_id = $1`
		args = append(args, *tenantID)
	}
	query += ` ORDER BY name ASC`
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list active committees: %w", err)
	}
	defer rows.Close()
	out := make([]model.Committee, 0)
	for rows.Next() {
		item, err := scanCommittee(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (s *Store) ListTenantIDs(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := s.db.Query(ctx, `SELECT DISTINCT tenant_id FROM committees WHERE deleted_at IS NULL ORDER BY tenant_id`)
	if err != nil {
		return nil, fmt.Errorf("list tenant ids: %w", err)
	}
	defer rows.Close()
	out := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan tenant id: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Store) CommitteeHasPendingMeetings(ctx context.Context, tenantID, committeeID uuid.UUID) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM meetings
			WHERE tenant_id = $1
			  AND committee_id = $2
			  AND deleted_at IS NULL
			  AND status IN ('draft', 'scheduled', 'in_progress', 'postponed')
		)`,
		tenantID, committeeID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check committee pending meetings: %w", err)
	}
	return exists, nil
}

func (s *Store) ListCommitteeMembers(ctx context.Context, tenantID, committeeID uuid.UUID, activeOnly bool) ([]model.CommitteeMember, error) {
	query := `
		SELECT id, tenant_id, committee_id, user_id, user_name, user_email, role, joined_at, left_at, active, created_at, updated_at
		FROM committee_members
		WHERE tenant_id = $1 AND committee_id = $2`
	if activeOnly {
		query += ` AND active = true`
	}
	query += ` ORDER BY role ASC, user_name ASC`
	rows, err := s.db.Query(ctx, query, tenantID, committeeID)
	if err != nil {
		return nil, fmt.Errorf("list committee members: %w", err)
	}
	defer rows.Close()
	out := make([]model.CommitteeMember, 0)
	for rows.Next() {
		item, err := scanCommitteeMember(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (s *Store) GetCommitteeMember(ctx context.Context, tenantID, committeeID, userID uuid.UUID) (*model.CommitteeMember, error) {
	row := s.db.QueryRow(ctx, `
		SELECT id, tenant_id, committee_id, user_id, user_name, user_email, role, joined_at, left_at, active, created_at, updated_at
		FROM committee_members
		WHERE tenant_id = $1 AND committee_id = $2 AND user_id = $3 AND active = true`,
		tenantID, committeeID, userID,
	)
	return scanCommitteeMember(row)
}

func (s *Store) UpsertCommitteeMember(ctx context.Context, q DBTX, member *model.CommitteeMember) error {
	tag, err := q.Exec(ctx, `
		UPDATE committee_members
		SET user_name = $4,
		    user_email = $5,
		    role = $6,
		    active = true,
		    left_at = NULL,
		    updated_at = $7
		WHERE tenant_id = $1 AND committee_id = $2 AND user_id = $3`,
		member.TenantID,
		member.CommitteeID,
		member.UserID,
		member.UserName,
		member.UserEmail,
		member.Role,
		member.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update committee member: %w", err)
	}
	if tag.RowsAffected() > 0 {
		return nil
	}
	_, err = q.Exec(ctx, `
		INSERT INTO committee_members (
			id, tenant_id, committee_id, user_id, user_name, user_email, role, joined_at, active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true, $9, $10)`,
		member.ID,
		member.TenantID,
		member.CommitteeID,
		member.UserID,
		member.UserName,
		member.UserEmail,
		member.Role,
		member.JoinedAt,
		member.CreatedAt,
		member.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert committee member: %w", err)
	}
	return nil
}

func (s *Store) DeactivateCommitteeMember(ctx context.Context, q DBTX, tenantID, committeeID, userID uuid.UUID, leftAt time.Time) error {
	tag, err := q.Exec(ctx, `
		UPDATE committee_members
		SET active = false,
		    left_at = $4,
		    updated_at = $4
		WHERE tenant_id = $1 AND committee_id = $2 AND user_id = $3 AND active = true`,
		tenantID, committeeID, userID, leftAt,
	)
	if err != nil {
		return fmt.Errorf("deactivate committee member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return notFoundError("committee_member", userID)
	}
	return nil
}

func (s *Store) UserIsCommitteeMember(ctx context.Context, tenantID, committeeID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM committee_members
			WHERE tenant_id = $1 AND committee_id = $2 AND user_id = $3 AND active = true
		)`,
		tenantID, committeeID, userID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check committee membership: %w", err)
	}
	return exists, nil
}

func (s *Store) CountQuorumEligibleMembers(ctx context.Context, q DBTX, tenantID, committeeID uuid.UUID) (int, error) {
	var count int
	err := q.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM committee_members
		WHERE tenant_id = $1
		  AND committee_id = $2
		  AND active = true
		  AND role != 'observer'`,
		tenantID, committeeID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count committee members: %w", err)
	}
	return count, nil
}

func (s *Store) GetCommitteeStats(ctx context.Context, tenantID, committeeID uuid.UUID) (*model.CommitteeStats, error) {
	stats := &model.CommitteeStats{}
	err := s.db.QueryRow(ctx, `
		SELECT
			COALESCE((SELECT COUNT(*) FROM committee_members WHERE tenant_id = $1 AND committee_id = $2 AND active = true AND role != 'observer'), 0),
			COALESCE((SELECT COUNT(*) FROM meetings WHERE tenant_id = $1 AND committee_id = $2 AND deleted_at IS NULL AND status IN ('draft', 'scheduled', 'postponed')), 0),
			COALESCE((SELECT COUNT(*) FROM meetings WHERE tenant_id = $1 AND committee_id = $2 AND deleted_at IS NULL AND status = 'completed'), 0),
			COALESCE((SELECT COUNT(*) FROM action_items WHERE tenant_id = $1 AND committee_id = $2 AND status IN ('pending', 'in_progress', 'deferred', 'overdue')), 0),
			COALESCE((SELECT COUNT(*) FROM action_items WHERE tenant_id = $1 AND committee_id = $2 AND status = 'overdue'), 0),
			COALESCE((SELECT COUNT(*) FROM meetings WHERE tenant_id = $1 AND committee_id = $2 AND has_minutes = true AND minutes_status IN ('draft', 'review', 'revision_requested')), 0)`,
		tenantID, committeeID,
	).Scan(
		&stats.ActiveMembers,
		&stats.UpcomingMeetings,
		&stats.CompletedMeetings,
		&stats.OpenActionItems,
		&stats.OverdueActionItems,
		&stats.PendingMinutesApproval,
	)
	if err != nil {
		return nil, fmt.Errorf("get committee stats: %w", err)
	}
	return stats, nil
}

func (s *Store) RefreshCommitteeLeadership(ctx context.Context, q DBTX, tenantID, committeeID uuid.UUID) error {
	var chairID *uuid.UUID
	var viceChairID *uuid.UUID
	var secretaryID *uuid.UUID
	rows, err := q.Query(ctx, `
		SELECT user_id, role
		FROM committee_members
		WHERE tenant_id = $1 AND committee_id = $2 AND active = true AND role IN ('chair', 'vice_chair', 'secretary')`,
		tenantID, committeeID,
	)
	if err != nil {
		return fmt.Errorf("query committee leadership: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var userID uuid.UUID
		var role string
		if err := rows.Scan(&userID, &role); err != nil {
			return fmt.Errorf("scan committee leadership: %w", err)
		}
		switch role {
		case "chair":
			chairID = ptr(userID)
		case "vice_chair":
			viceChairID = ptr(userID)
		case "secretary":
			secretaryID = ptr(userID)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate committee leadership: %w", err)
	}
	if chairID == nil {
		return fmt.Errorf("committee %s must retain a chair", committeeID)
	}
	_, err = q.Exec(ctx, `
		UPDATE committees
		SET chair_user_id = $3,
		    vice_chair_user_id = $4,
		    secretary_user_id = $5,
		    updated_at = now()
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`,
		tenantID, committeeID, *chairID, nullableUUID(viceChairID), nullableUUID(secretaryID),
	)
	if err != nil {
		return fmt.Errorf("refresh committee leadership: %w", err)
	}
	return nil
}

func scanCommittee(scanner rowScanner) (*model.Committee, error) {
	var (
		item            model.Committee
		metadataRaw      []byte
		viceChairUserID  *uuid.UUID
		secretaryUserID  *uuid.UUID
		quorumFixedCount *int
		charter          *string
		establishedDate  *time.Time
		dissolutionDate  *time.Time
		deletedAt        *time.Time
	)
	err := scanner.Scan(
		&item.ID,
		&item.TenantID,
		&item.Name,
		&item.Type,
		&item.Description,
		&item.ChairUserID,
		&viceChairUserID,
		&secretaryUserID,
		&item.MeetingFrequency,
		&item.QuorumPercentage,
		&item.QuorumType,
		&quorumFixedCount,
		&charter,
		&establishedDate,
		&dissolutionDate,
		&item.Status,
		&item.Tags,
		&metadataRaw,
		&item.CreatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
		&deletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, err
		}
		return nil, fmt.Errorf("scan committee: %w", err)
	}
	item.ViceChairUserID = viceChairUserID
	item.SecretaryUserID = secretaryUserID
	item.QuorumFixedCount = quorumFixedCount
	item.Charter = charter
	item.EstablishedDate = establishedDate
	item.DissolutionDate = dissolutionDate
	item.DeletedAt = deletedAt
	item.Metadata, err = decodeJSONMap(metadataRaw)
	if err != nil {
		return nil, fmt.Errorf("decode committee metadata: %w", err)
	}
	return &item, nil
}

func scanCommitteeMember(scanner rowScanner) (*model.CommitteeMember, error) {
	var item model.CommitteeMember
	if err := scanner.Scan(
		&item.ID,
		&item.TenantID,
		&item.CommitteeID,
		&item.UserID,
		&item.UserName,
		&item.UserEmail,
		&item.Role,
		&item.JoinedAt,
		&item.LeftAt,
		&item.Active,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan committee member: %w", err)
	}
	return &item, nil
}

func scanCommitteeWithMemberCount(scanner rowScanner) (*model.Committee, int, error) {
	var (
		item            model.Committee
		metadataRaw      []byte
		viceChairUserID  *uuid.UUID
		secretaryUserID  *uuid.UUID
		quorumFixedCount *int
		charter          *string
		establishedDate  *time.Time
		dissolutionDate  *time.Time
		deletedAt        *time.Time
		memberCount      int
	)
	err := scanner.Scan(
		&item.ID,
		&item.TenantID,
		&item.Name,
		&item.Type,
		&item.Description,
		&item.ChairUserID,
		&viceChairUserID,
		&secretaryUserID,
		&item.MeetingFrequency,
		&item.QuorumPercentage,
		&item.QuorumType,
		&quorumFixedCount,
		&charter,
		&establishedDate,
		&dissolutionDate,
		&item.Status,
		&item.Tags,
		&metadataRaw,
		&item.CreatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
		&deletedAt,
		&memberCount,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("scan committee with member count: %w", err)
	}
	item.ViceChairUserID = viceChairUserID
	item.SecretaryUserID = secretaryUserID
	item.QuorumFixedCount = quorumFixedCount
	item.Charter = charter
	item.EstablishedDate = establishedDate
	item.DissolutionDate = dissolutionDate
	item.DeletedAt = deletedAt
	item.Metadata, err = decodeJSONMap(metadataRaw)
	if err != nil {
		return nil, 0, fmt.Errorf("decode committee metadata: %w", err)
	}
	return &item, memberCount, nil
}

func dateOnly(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format("2006-01-02")
}
