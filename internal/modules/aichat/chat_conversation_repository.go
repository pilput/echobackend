package aichat

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

type ChatConversationRepository interface {
	CreateConversation(ctx context.Context, conversation *ChatConversation) (*ChatConversation, error)
	GetConversationByID(ctx context.Context, id string) (*ChatConversation, error)
	GetConversationsByUserID(ctx context.Context, userID string, offset int, limit int) ([]*ChatConversation, int64, error)
	UpdateConversation(ctx context.Context, id string, updates map[string]any) (*ChatConversation, error)
	DeleteConversation(ctx context.Context, id string) error
	GetUserConversations(ctx context.Context, userID string, offset int, limit int) ([]*ChatConversation, int64, error)
	CreateMessage(ctx context.Context, message *ChatMessage) (*ChatMessage, error)
	GetMessagesByConversationID(ctx context.Context, conversationID, userID string) ([]*ChatMessage, error)
	GetMessagesByConversationIDAsc(ctx context.Context, conversationID, userID string) ([]*ChatMessage, error)
	GetMessageByID(ctx context.Context, id, userID string) (*ChatMessage, error)
	DeleteMessage(ctx context.Context, id, userID string) error
}

type chatConversationRepository struct {
	db *gorm.DB
}

func NewChatConversationRepository(db *gorm.DB) ChatConversationRepository {
	return &chatConversationRepository{db: db}
}

func (r *chatConversationRepository) CreateConversation(ctx context.Context, conversation *ChatConversation) (*ChatConversation, error) {
	err := r.db.WithContext(ctx).Create(conversation).Error
	if err != nil {
		return nil, fmt.Errorf("failed to create chat conversation: %w", err)
	}
	return conversation, nil
}

func (r *chatConversationRepository) GetConversationByID(ctx context.Context, id string) (*ChatConversation, error) {
	var conversation ChatConversation
	err := r.db.WithContext(ctx).
		Preload("Messages", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		First(&conversation, "id = ?", id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrChatConversationNotFound
		}
		return nil, fmt.Errorf("failed to get chat conversation: %w", err)
	}
	return &conversation, nil
}

func (r *chatConversationRepository) GetConversationsByUserID(ctx context.Context, userID string, offset int, limit int) ([]*ChatConversation, int64, error) {
	return r.GetUserConversations(ctx, userID, offset, limit)
}

func (r *chatConversationRepository) GetUserConversations(ctx context.Context, userID string, offset int, limit int) ([]*ChatConversation, int64, error) {
	var conversations []*ChatConversation
	var count int64

	err := r.db.WithContext(ctx).Model(&ChatConversation{}).Where("user_id = ?", userID).Count(&count).Error
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count chat conversations: %w", err)
	}

	err = r.db.WithContext(ctx).
		Preload("Messages").
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&conversations).Error

	if err != nil {
		return nil, 0, fmt.Errorf("failed to get chat conversations: %w", err)
	}

	return conversations, count, nil
}

func (r *chatConversationRepository) UpdateConversation(ctx context.Context, id string, updates map[string]any) (*ChatConversation, error) {
	var existingConversation ChatConversation
	err := r.db.WithContext(ctx).First(&existingConversation, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrChatConversationNotFound
		}
		return nil, fmt.Errorf("error checking conversation existence: %w", err)
	}

	if len(updates) == 0 {
		return &existingConversation, nil
	}

	result := r.db.WithContext(ctx).Model(&ChatConversation{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return nil, fmt.Errorf("failed to update chat conversation: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, ErrChatConversationNotFound
	}

	err = r.db.WithContext(ctx).
		Preload("Messages").
		First(&existingConversation, "id = ?", id).Error

	if err != nil {
		return nil, fmt.Errorf("conversation updated, but failed to retrieve updated record: %w", err)
	}

	return &existingConversation, nil
}

func (r *chatConversationRepository) DeleteConversation(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&ChatConversation{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete chat conversation: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrChatConversationNotFound
	}
	return nil
}

func (r *chatConversationRepository) CreateMessage(ctx context.Context, message *ChatMessage) (*ChatMessage, error) {
	if err := r.db.WithContext(ctx).Create(message).Error; err != nil {
		return nil, fmt.Errorf("failed to create chat message: %w", err)
	}
	return message, nil
}

func (r *chatConversationRepository) GetMessagesByConversationID(ctx context.Context, conversationID, userID string) ([]*ChatMessage, error) {
	var messages []*ChatMessage
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Order("created_at DESC").
		Find(&messages).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get chat messages: %w", err)
	}
	return messages, nil
}

func (r *chatConversationRepository) GetMessagesByConversationIDAsc(ctx context.Context, conversationID, userID string) ([]*ChatMessage, error) {
	var messages []*ChatMessage
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Order("created_at ASC").
		Find(&messages).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get chat messages: %w", err)
	}
	return messages, nil
}

func (r *chatConversationRepository) GetMessageByID(ctx context.Context, id, userID string) (*ChatMessage, error) {
	var message ChatMessage
	err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&message).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrChatMessageNotFound
		}
		return nil, fmt.Errorf("failed to get chat message: %w", err)
	}
	return &message, nil
}

func (r *chatConversationRepository) DeleteMessage(ctx context.Context, id, userID string) error {
	result := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&ChatMessage{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete chat message: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrChatMessageNotFound
	}
	return nil
}
