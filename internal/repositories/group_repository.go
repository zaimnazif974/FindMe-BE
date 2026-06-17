package repositories

import (
	"context"
	"fmt"
	"strings"

	"findme/backend/internal/apperror"
	groupdto "findme/backend/internal/dto/groups"
	"findme/backend/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GroupRepository struct {
	db *gorm.DB
}

func NewGroupRepository(db *gorm.DB) *GroupRepository {
	return &GroupRepository{db: db}
}

func (r *GroupRepository) Create(ctx context.Context, userID uuid.UUID, name string, description *string, inviteCode string) (groupdto.Group, error) {
	var group groupdto.Group
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?::text, 0))`, userID).Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Table("group_members").Where("user_id = ?", userID).Count(&count).Error; err != nil {
			return err
		}
		if count >= 5 {
			return apperror.ErrGroupLimit
		}
		if err := tx.Raw(`
			INSERT INTO groups (name, description, invite_code, created_by)
			VALUES (?, ?, ?, ?)
			RETURNING id, name, description, invite_code, created_by, created_at, updated_at
		`, strings.TrimSpace(name), description, inviteCode, userID).Scan(&group).Error; err != nil {
			return err
		}
		return tx.Table("group_members").Create(map[string]any{
			"group_id": group.ID, "user_id": userID, "role": "admin",
		}).Error
	})
	if err != nil {
		return groupdto.Group{}, err
	}
	group.Role, group.MemberCount = "admin", 1
	return group, nil
}

func (r *GroupRepository) Join(ctx context.Context, userID uuid.UUID, inviteCode string) (groupdto.Group, error) {
	var group groupdto.Group
	var members int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`SELECT pg_advisory_xact_lock(hashtextextended(?::text, 0))`, userID).Error; err != nil {
			return err
		}
		err := tx.Table("groups").Where("invite_code = upper(?)", strings.TrimSpace(inviteCode)).
			Clauses(clause.Locking{Strength: "UPDATE"}).Take(&group).Error
		if err != nil {
			return utils.DatabaseError(err)
		}
		var userGroups int64
		if err := tx.Table("group_members").Where("user_id = ?", userID).Count(&userGroups).Error; err != nil {
			return err
		}
		if userGroups >= 5 {
			return apperror.ErrGroupLimit
		}
		if err := tx.Table("group_members").Where("group_id = ?", group.ID).Count(&members).Error; err != nil {
			return err
		}
		if members >= 10 {
			return apperror.ErrMemberLimit
		}
		result := tx.Exec(`
			INSERT INTO group_members (group_id, user_id, role)
			VALUES (?, ?, 'member') ON CONFLICT DO NOTHING
		`, group.ID, userID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return apperror.ErrConflict
		}
		return nil
	})
	if err != nil {
		return groupdto.Group{}, err
	}
	group.Role, group.MemberCount = "member", int(members)+1
	return group, nil
}

func (r *GroupRepository) List(ctx context.Context, userID uuid.UUID) ([]groupdto.Group, error) {
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT g.id, g.name, g.description, g.invite_code, g.created_by, gm.role,
		       (SELECT count(*) FROM group_members x WHERE x.group_id = g.id) AS member_count,
		       g.created_at, g.updated_at
		FROM groups g JOIN group_members gm ON gm.group_id = g.id
		WHERE gm.user_id = ? ORDER BY g.updated_at DESC
	`, userID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []groupdto.Group{}
	for rows.Next() {
		var group groupdto.Group
		if err := rows.Scan(&group.ID, &group.Name, &group.Description, &group.InviteCode, &group.CreatedBy, &group.Role, &group.MemberCount, &group.CreatedAt, &group.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, group)
	}
	return result, rows.Err()
}

func (r *GroupRepository) Get(ctx context.Context, groupID, userID uuid.UUID) (groupdto.Group, error) {
	var group groupdto.Group
	err := r.db.WithContext(ctx).Raw(`
		SELECT g.id, g.name, g.description, g.invite_code, g.created_by, gm.role,
		       (SELECT count(*) FROM group_members x WHERE x.group_id = g.id) AS member_count,
		       g.created_at, g.updated_at
		FROM groups g JOIN group_members gm ON gm.group_id = g.id
		WHERE g.id = ? AND gm.user_id = ?
	`, groupID, userID).Scan(&group).Error
	if err != nil {
		return groupdto.Group{}, utils.DatabaseError(err)
	}
	if group.ID == uuid.Nil {
		return groupdto.Group{}, apperror.ErrNotFound
	}
	return group, nil
}

func (r *GroupRepository) Members(ctx context.Context, groupID uuid.UUID) ([]groupdto.Member, error) {
	rows, err := r.db.WithContext(ctx).Raw(`
		SELECT u.id AS user_id, u.name, u.email, u.avatar_url, gm.role, gm.joined_at
		FROM group_members gm JOIN users u ON u.id = gm.user_id
		WHERE gm.group_id = ? ORDER BY gm.joined_at
	`, groupID).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []groupdto.Member{}
	for rows.Next() {
		var member groupdto.Member
		if err := rows.Scan(&member.UserID, &member.Name, &member.Email, &member.AvatarURL, &member.Role, &member.JoinedAt); err != nil {
			return nil, err
		}
		result = append(result, member)
	}
	return result, rows.Err()
}

func (r *GroupRepository) IsMember(ctx context.Context, groupID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.WithContext(ctx).Raw(`SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = ? AND user_id = ?)`, groupID, userID).Scan(&exists).Error
	return exists, err
}

func (r *GroupRepository) IsAdmin(ctx context.Context, groupID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.WithContext(ctx).Raw(`SELECT EXISTS(SELECT 1 FROM group_members WHERE group_id = ? AND user_id = ? AND role = 'admin')`, groupID, userID).Scan(&exists).Error
	return exists, err
}

func (r *GroupRepository) Update(ctx context.Context, groupID uuid.UUID, name string, description *string) (groupdto.Group, error) {
	var group groupdto.Group
	err := r.db.WithContext(ctx).Raw(`
		UPDATE groups SET name = ?, description = ?, updated_at = now()
		WHERE id = ?
		RETURNING id, name, description, invite_code, created_by, created_at, updated_at
	`, strings.TrimSpace(name), description, groupID).Scan(&group).Error
	return group, utils.DatabaseError(err)
}

func (r *GroupRepository) Delete(ctx context.Context, groupID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec("DELETE FROM groups WHERE id = ?", groupID).Error
}

func (r *GroupRepository) Leave(ctx context.Context, groupID, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).Exec("DELETE FROM group_members WHERE group_id = ? AND user_id = ?", groupID, userID)
	if result.Error == nil && result.RowsAffected == 0 {
		return apperror.ErrNotFound
	}
	return result.Error
}

func (r *GroupRepository) RemoveMember(ctx context.Context, groupID, memberID uuid.UUID) error {
	result := r.db.WithContext(ctx).Exec(
		"DELETE FROM group_members WHERE group_id = ? AND user_id = ? AND role <> 'admin'",
		groupID, memberID,
	)
	if result.Error == nil && result.RowsAffected == 0 {
		return apperror.ErrNotFound
	}
	return result.Error
}

func (r *GroupRepository) RegenerateInviteCode(ctx context.Context, groupID uuid.UUID, code string) (string, error) {
	var result string
	err := r.db.WithContext(ctx).Raw(`UPDATE groups SET invite_code = ?, updated_at = now() WHERE id = ? RETURNING invite_code`, code, groupID).Scan(&result).Error
	return result, err
}

func (r *GroupRepository) AssertMember(ctx context.Context, groupID, userID uuid.UUID) error {
	ok, err := r.IsMember(ctx, groupID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: group membership required", apperror.ErrForbidden)
	}
	return nil
}
