package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"ai_chat/internal/domain"

	_ "github.com/go-sql-driver/mysql"
)

var ErrNotFound = domain.ErrNotFound

type MySQLStore struct {
	db *sql.DB
}

type MySQLConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Database string
}

func OpenMySQL(dsn string) (*MySQLStore, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &MySQLStore{db: db}, nil
}

func EnsureMySQLDatabase(ctx context.Context, cfg MySQLConfig) error {
	if err := validateDatabaseName(cfg.Database); err != nil {
		return err
	}
	db, err := sql.Open("mysql", cfg.AdminDSN())
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", cfg.Database))
	return err
}

func (s *MySQLStore) Close() error {
	return s.db.Close()
}

func (s *MySQLStore) ClearAllTables(ctx context.Context) error {
	tables := []string{
		"token_usages",
		"tool_executions",
		"chat_files",
		"messages",
		"roles",
		"chats",
		"model_configs",
		"sessions",
		"users",
	}
	for _, table := range tables {
		if _, err := s.db.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return err
		}
	}
	return nil
}

func (s *MySQLStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *MySQLStore) Migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY uniq_users_email (email)
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token_hash CHAR(64) PRIMARY KEY,
			user_id BIGINT NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_sessions_user_id (user_id),
			INDEX idx_sessions_expires_at (expires_at),
			CONSTRAINT fk_sessions_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS chats (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			user_id BIGINT NULL,
			name VARCHAR(255) NOT NULL,
			topic VARCHAR(500) NOT NULL DEFAULT '',
			ai_review_enabled BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_chats_user_id (user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS roles (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			chat_id BIGINT NOT NULL,
			model_config_id BIGINT NULL,
			name VARCHAR(255) NOT NULL,
			avatar VARCHAR(1024) NOT NULL DEFAULT '',
			persona TEXT NOT NULL,
			reply_style TEXT NOT NULL,
			model VARCHAR(255) NOT NULL,
			reasoning_effort VARCHAR(32) NOT NULL DEFAULT '',
			can_speak BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_roles_chat_id (chat_id),
			CONSTRAINT fk_roles_chat_id FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS model_configs (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			user_id BIGINT NULL,
			name VARCHAR(255) NOT NULL DEFAULT 'Default API',
			provider VARCHAR(255) NOT NULL,
			base_url VARCHAR(1024) NOT NULL,
			api_key TEXT NOT NULL,
			default_model VARCHAR(255) NOT NULL,
			models TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_model_configs_user_id (user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS messages (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			chat_id BIGINT NOT NULL,
			sender_type VARCHAR(32) NOT NULL,
			sender_name VARCHAR(255) NOT NULL,
			role_id BIGINT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_messages_chat_created (chat_id, created_at, id),
			CONSTRAINT fk_messages_chat_id FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE,
			CONSTRAINT fk_messages_role_id FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE SET NULL
		)`,
		`CREATE TABLE IF NOT EXISTS token_usages (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			user_id BIGINT NOT NULL,
			chat_id BIGINT NOT NULL,
			message_id BIGINT NOT NULL,
			role_id BIGINT NULL,
			model_config_id BIGINT NULL,
			model VARCHAR(255) NOT NULL,
			prompt_tokens INT NOT NULL DEFAULT 0,
			completion_tokens INT NOT NULL DEFAULT 0,
			total_tokens INT NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_token_usages_user_created (user_id, created_at),
			INDEX idx_token_usages_chat_id (chat_id),
			CONSTRAINT fk_token_usages_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			CONSTRAINT fk_token_usages_chat_id FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE,
			CONSTRAINT fk_token_usages_message_id FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
			CONSTRAINT fk_token_usages_role_id FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE SET NULL
		)`,
		`CREATE TABLE IF NOT EXISTS chat_files (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			user_id BIGINT NOT NULL,
			chat_id BIGINT NOT NULL,
			original_name VARCHAR(255) NOT NULL,
			storage_path VARCHAR(1024) NOT NULL,
			content_type VARCHAR(255) NOT NULL,
			size_bytes BIGINT NOT NULL,
			extracted_text MEDIUMTEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_chat_files_user_chat_created (user_id, chat_id, created_at),
			CONSTRAINT fk_chat_files_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			CONSTRAINT fk_chat_files_chat_id FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS tool_executions (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			user_id BIGINT NOT NULL,
			chat_id BIGINT NOT NULL,
			message_id BIGINT NOT NULL,
			tool_name VARCHAR(255) NOT NULL,
			input TEXT NOT NULL,
			status VARCHAR(32) NOT NULL,
			result TEXT NOT NULL,
			error TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_tool_executions_user_chat_created (user_id, chat_id, created_at),
			CONSTRAINT fk_tool_executions_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			CONSTRAINT fk_tool_executions_chat_id FOREIGN KEY (chat_id) REFERENCES chats(id) ON DELETE CASCADE,
			CONSTRAINT fk_tool_executions_message_id FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
		)`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if err := s.addColumnIfMissing(ctx, "model_configs", "models", "TEXT NOT NULL"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "model_configs", "user_id", "BIGINT NULL"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "model_configs", "name", "VARCHAR(255) NOT NULL DEFAULT 'Default API'"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "chats", "ai_review_enabled", "BOOLEAN NOT NULL DEFAULT FALSE"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "chats", "user_id", "BIGINT NULL"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "chats", "topic", "VARCHAR(500) NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "roles", "can_speak", "BOOLEAN NOT NULL DEFAULT TRUE"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "roles", "model_config_id", "BIGINT NULL"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "roles", "avatar", "VARCHAR(1024) NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.addColumnIfMissing(ctx, "roles", "reasoning_effort", "VARCHAR(32) NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := s.assignLegacyRoleModelConfigs(ctx); err != nil {
		return err
	}
	return nil
}

func (s *MySQLStore) addColumnIfMissing(ctx context.Context, table, column, definition string) error {
	var count int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?`,
		table,
		column,
	).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err = s.db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN `%s` %s", table, column, definition))
	return err
}

func (s *MySQLStore) CreateUser(ctx context.Context, email, passwordHash string) (domain.User, error) {
	res, err := s.db.ExecContext(ctx, `INSERT INTO users (email, password_hash) VALUES (?, ?)`, email, passwordHash)
	if err != nil {
		return domain.User{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.User{}, err
	}
	if err := s.assignLegacyDataToFirstUser(ctx, id); err != nil {
		return domain.User{}, err
	}
	return s.GetUser(ctx, id)
}

func (s *MySQLStore) assignLegacyDataToFirstUser(ctx context.Context, userID int64) error {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return err
	}
	if count != 1 {
		return nil
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE chats SET user_id = ? WHERE user_id IS NULL`, userID); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `UPDATE model_configs SET user_id = ? WHERE user_id IS NULL`, userID)
	return err
}

func (s *MySQLStore) assignLegacyRoleModelConfigs(ctx context.Context) error {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT r.id, c.user_id
		 FROM roles r
		 JOIN chats c ON c.id = r.chat_id
		 WHERE r.model_config_id IS NULL AND c.user_id IS NOT NULL`,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	type legacyRole struct {
		roleID int64
		userID int64
	}
	var roles []legacyRole
	for rows.Next() {
		var role legacyRole
		if err := rows.Scan(&role.roleID, &role.userID); err != nil {
			return err
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, role := range roles {
		var configID int64
		err := s.db.QueryRowContext(ctx, `SELECT id FROM model_configs WHERE user_id = ? ORDER BY id ASC LIMIT 1`, role.userID).Scan(&configID)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return err
		}
		if _, err := s.db.ExecContext(ctx, `UPDATE roles SET model_config_id = ? WHERE id = ? AND model_config_id IS NULL`, configID, role.roleID); err != nil {
			return err
		}
	}
	return nil
}

func (s *MySQLStore) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	var user domain.User
	err := s.db.QueryRowContext(ctx, `SELECT id, email, password_hash, created_at, updated_at FROM users WHERE email = ?`, email).
		Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, ErrNotFound
	}
	return user, err
}

func (s *MySQLStore) GetUser(ctx context.Context, userID int64) (domain.User, error) {
	var user domain.User
	err := s.db.QueryRowContext(ctx, `SELECT id, email, password_hash, created_at, updated_at FROM users WHERE id = ?`, userID).
		Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, ErrNotFound
	}
	return user, err
}

func (s *MySQLStore) CreateSession(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO sessions (token_hash, user_id, expires_at) VALUES (?, ?, ?)`, tokenHash, userID, expiresAt)
	return err
}

func (s *MySQLStore) GetSessionUser(ctx context.Context, tokenHash string, now time.Time) (domain.User, error) {
	var user domain.User
	err := s.db.QueryRowContext(
		ctx,
		`SELECT u.id, u.email, u.password_hash, u.created_at, u.updated_at
		 FROM sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.token_hash = ? AND s.expires_at > ?`,
		tokenHash,
		now,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.User{}, ErrNotFound
	}
	return user, err
}

func (s *MySQLStore) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, tokenHash)
	return err
}

func (s *MySQLStore) CreateChat(ctx context.Context, name string) (domain.Chat, error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return domain.Chat{}, err
	}
	res, err := s.db.ExecContext(ctx, `INSERT INTO chats (user_id, name) VALUES (?, ?)`, userID, name)
	if err != nil {
		return domain.Chat{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Chat{}, err
	}
	return s.GetChat(ctx, id)
}

func (s *MySQLStore) ListChats(ctx context.Context) ([]domain.Chat, error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, user_id, name, topic, ai_review_enabled, created_at, updated_at FROM chats WHERE user_id = ? ORDER BY updated_at DESC, id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chats []domain.Chat
	for rows.Next() {
		var chat domain.Chat
		if err := rows.Scan(&chat.ID, &chat.UserID, &chat.Name, &chat.Topic, &chat.AIReviewEnabled, &chat.CreatedAt, &chat.UpdatedAt); err != nil {
			return nil, err
		}
		chats = append(chats, chat)
	}
	return chats, rows.Err()
}

func (s *MySQLStore) DeleteChat(ctx context.Context, chatID int64) error {
	userID, err := requireUserID(ctx)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM chats WHERE id = ? AND user_id = ?`, chatID, userID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *MySQLStore) GetChat(ctx context.Context, chatID int64) (domain.Chat, error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return domain.Chat{}, err
	}
	var chat domain.Chat
	err = s.db.QueryRowContext(ctx, `SELECT id, user_id, name, topic, ai_review_enabled, created_at, updated_at FROM chats WHERE id = ? AND user_id = ?`, chatID, userID).
		Scan(&chat.ID, &chat.UserID, &chat.Name, &chat.Topic, &chat.AIReviewEnabled, &chat.CreatedAt, &chat.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Chat{}, ErrNotFound
	}
	return chat, err
}

func (s *MySQLStore) UpdateChatAIReview(ctx context.Context, chatID int64, enabled bool) (domain.Chat, error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return domain.Chat{}, err
	}
	res, err := s.db.ExecContext(ctx, `UPDATE chats SET ai_review_enabled = ? WHERE id = ? AND user_id = ?`, enabled, chatID, userID)
	if err != nil {
		return domain.Chat{}, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return domain.Chat{}, err
	}
	if affected == 0 {
		return domain.Chat{}, ErrNotFound
	}
	return s.GetChat(ctx, chatID)
}

func (s *MySQLStore) UpdateChatTopic(ctx context.Context, chatID int64, topic string) (domain.Chat, error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return domain.Chat{}, err
	}
	res, err := s.db.ExecContext(ctx, `UPDATE chats SET topic = ? WHERE id = ? AND user_id = ?`, topic, chatID, userID)
	if err != nil {
		return domain.Chat{}, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return domain.Chat{}, err
	}
	if affected == 0 {
		return domain.Chat{}, ErrNotFound
	}
	return s.GetChat(ctx, chatID)
}

func (s *MySQLStore) CreateChatFile(ctx context.Context, file domain.ChatFile) (domain.ChatFile, error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return domain.ChatFile{}, err
	}
	res, err := s.db.ExecContext(
		ctx,
		`INSERT INTO chat_files (user_id, chat_id, original_name, storage_path, content_type, size_bytes, extracted_text)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID,
		file.ChatID,
		file.OriginalName,
		file.StoragePath,
		file.ContentType,
		file.SizeBytes,
		file.ExtractedText,
	)
	if err != nil {
		return domain.ChatFile{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.ChatFile{}, err
	}
	return s.getChatFile(ctx, id)
}

func (s *MySQLStore) ListChatFiles(ctx context.Context, chatID int64) ([]domain.ChatFile, error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, user_id, chat_id, original_name, storage_path, content_type, size_bytes, extracted_text, created_at
		 FROM chat_files
		 WHERE user_id = ? AND chat_id = ?
		 ORDER BY created_at DESC, id DESC`,
		userID,
		chatID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := []domain.ChatFile{}
	for rows.Next() {
		file, err := scanChatFile(rows)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, rows.Err()
}

func (s *MySQLStore) getChatFile(ctx context.Context, id int64) (domain.ChatFile, error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return domain.ChatFile{}, err
	}
	file, err := scanChatFile(s.db.QueryRowContext(
		ctx,
		`SELECT id, user_id, chat_id, original_name, storage_path, content_type, size_bytes, extracted_text, created_at
		 FROM chat_files
		 WHERE id = ? AND user_id = ?`,
		id,
		userID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ChatFile{}, ErrNotFound
	}
	return file, err
}

func (s *MySQLStore) CreateRole(ctx context.Context, role domain.Role) (domain.Role, error) {
	res, err := s.db.ExecContext(
		ctx,
		`INSERT INTO roles (chat_id, model_config_id, name, avatar, persona, reply_style, model, reasoning_effort, can_speak) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		role.ChatID,
		nullInt64(role.ModelConfigID),
		role.Name,
		role.Avatar,
		role.Persona,
		role.ReplyStyle,
		role.Model,
		role.ReasoningEffort,
		role.CanSpeak,
	)
	if err != nil {
		return domain.Role{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Role{}, err
	}
	roles, err := s.ListRoles(ctx, role.ChatID)
	if err != nil {
		return domain.Role{}, err
	}
	for _, created := range roles {
		if created.ID == id {
			return created, nil
		}
	}
	return domain.Role{}, ErrNotFound
}

func (s *MySQLStore) ListRoles(ctx context.Context, chatID int64) ([]domain.Role, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, chat_id, COALESCE(model_config_id, 0), name, avatar, persona, reply_style, model, reasoning_effort, can_speak, created_at, updated_at FROM roles WHERE chat_id = ? ORDER BY id ASC`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []domain.Role
	for rows.Next() {
		var role domain.Role
		if err := rows.Scan(&role.ID, &role.ChatID, &role.ModelConfigID, &role.Name, &role.Avatar, &role.Persona, &role.ReplyStyle, &role.Model, &role.ReasoningEffort, &role.CanSpeak, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (s *MySQLStore) GetRole(ctx context.Context, chatID, roleID int64) (domain.Role, error) {
	var role domain.Role
	err := s.db.QueryRowContext(ctx, `SELECT id, chat_id, COALESCE(model_config_id, 0), name, avatar, persona, reply_style, model, reasoning_effort, can_speak, created_at, updated_at FROM roles WHERE id = ? AND chat_id = ?`, roleID, chatID).
		Scan(&role.ID, &role.ChatID, &role.ModelConfigID, &role.Name, &role.Avatar, &role.Persona, &role.ReplyStyle, &role.Model, &role.ReasoningEffort, &role.CanSpeak, &role.CreatedAt, &role.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Role{}, ErrNotFound
	}
	return role, err
}

func (s *MySQLStore) UpdateRole(ctx context.Context, role domain.Role) (domain.Role, error) {
	res, err := s.db.ExecContext(
		ctx,
		`UPDATE roles SET model_config_id = ?, name = ?, avatar = ?, persona = ?, reply_style = ?, model = ?, reasoning_effort = ?, can_speak = ? WHERE id = ? AND chat_id = ?`,
		nullInt64(role.ModelConfigID),
		role.Name,
		role.Avatar,
		role.Persona,
		role.ReplyStyle,
		role.Model,
		role.ReasoningEffort,
		role.CanSpeak,
		role.ID,
		role.ChatID,
	)
	if err != nil {
		return domain.Role{}, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return domain.Role{}, err
	}
	if affected == 0 {
		return domain.Role{}, ErrNotFound
	}
	return s.GetRole(ctx, role.ChatID, role.ID)
}

func (s *MySQLStore) DeleteRole(ctx context.Context, chatID, roleID int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM roles WHERE id = ? AND chat_id = ?`, roleID, chatID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *MySQLStore) SaveModelConfig(ctx context.Context, config domain.ModelConfig) (domain.ModelConfig, error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return domain.ModelConfig{}, err
	}
	config.UserID = userID
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.ModelConfig{}, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	res, err := tx.ExecContext(ctx, `INSERT INTO model_configs (user_id, name, provider, base_url, api_key, default_model, models) VALUES (?, ?, ?, ?, ?, ?, ?)`, config.UserID, config.Name, config.Provider, config.BaseURL, config.APIKey, config.DefaultModel, strings.Join(config.Models, "\n"))
	if err != nil {
		return domain.ModelConfig{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.ModelConfig{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.ModelConfig{}, err
	}
	return s.getModelConfigByID(ctx, id)
}

func (s *MySQLStore) GetModelConfig(ctx context.Context) (domain.ModelConfig, error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return domain.ModelConfig{}, err
	}
	var id int64
	err = s.db.QueryRowContext(ctx, `SELECT id FROM model_configs WHERE user_id = ? ORDER BY id DESC LIMIT 1`, userID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ModelConfig{}, ErrNotFound
	}
	if err != nil {
		return domain.ModelConfig{}, err
	}
	return s.getModelConfigByID(ctx, id)
}

func (s *MySQLStore) ListModelConfigs(ctx context.Context) ([]domain.ModelConfig, error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, user_id, name, provider, base_url, api_key, default_model, models, created_at, updated_at FROM model_configs WHERE user_id = ? ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	configs := []domain.ModelConfig{}
	for rows.Next() {
		config, err := scanModelConfig(rows)
		if err != nil {
			return nil, err
		}
		configs = append(configs, config)
	}
	return configs, rows.Err()
}

func (s *MySQLStore) GetModelConfigByID(ctx context.Context, configID int64) (domain.ModelConfig, error) {
	return s.getModelConfigByID(ctx, configID)
}

func (s *MySQLStore) DeleteModelConfig(ctx context.Context, configID int64) error {
	userID, err := requireUserID(ctx)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM model_configs WHERE id = ? AND user_id = ?`, configID, userID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *MySQLStore) CountRolesByModelConfig(ctx context.Context, configID int64) (int, error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return 0, err
	}
	var count int
	err = s.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
		 FROM roles r
		 JOIN chats c ON c.id = r.chat_id
		 WHERE r.model_config_id = ? AND c.user_id = ?`,
		configID,
		userID,
	).Scan(&count)
	return count, err
}

func (s *MySQLStore) getModelConfigByID(ctx context.Context, id int64) (domain.ModelConfig, error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return domain.ModelConfig{}, err
	}
	config, err := scanModelConfig(s.db.QueryRowContext(ctx, `SELECT id, user_id, name, provider, base_url, api_key, default_model, models, created_at, updated_at FROM model_configs WHERE id = ? AND user_id = ?`, id, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ModelConfig{}, ErrNotFound
	}
	if err != nil {
		return domain.ModelConfig{}, err
	}
	return config, nil
}

func (s *MySQLStore) CreateMessage(ctx context.Context, message domain.Message) (domain.Message, error) {
	res, err := s.db.ExecContext(
		ctx,
		`INSERT INTO messages (chat_id, sender_type, sender_name, role_id, content) VALUES (?, ?, ?, ?, ?)`,
		message.ChatID,
		string(message.SenderType),
		message.SenderName,
		message.RoleID,
		message.Content,
	)
	if err != nil {
		return domain.Message{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Message{}, err
	}
	return s.getMessage(ctx, id)
}

func (s *MySQLStore) getMessage(ctx context.Context, id int64) (domain.Message, error) {
	var msg domain.Message
	var senderType string
	err := s.db.QueryRowContext(ctx, `SELECT m.id, m.chat_id, m.sender_type, m.sender_name, COALESCE(r.avatar, ''), m.role_id, m.content, m.created_at FROM messages m LEFT JOIN roles r ON r.id = m.role_id WHERE m.id = ?`, id).
		Scan(&msg.ID, &msg.ChatID, &senderType, &msg.SenderName, &msg.SenderAvatar, &msg.RoleID, &msg.Content, &msg.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Message{}, ErrNotFound
	}
	if err != nil {
		return domain.Message{}, err
	}
	msg.SenderType = domain.SenderType(senderType)
	return msg, nil
}

func (s *MySQLStore) ListMessages(ctx context.Context, chatID int64) ([]domain.Message, error) {
	return s.listMessages(ctx, `SELECT m.id, m.chat_id, m.sender_type, m.sender_name, COALESCE(r.avatar, ''), m.role_id, m.content, m.created_at FROM messages m LEFT JOIN roles r ON r.id = m.role_id WHERE m.chat_id = ? ORDER BY m.created_at ASC, m.id ASC`, chatID)
}

func (s *MySQLStore) ListMessagesAfter(ctx context.Context, chatID, afterID int64) ([]domain.Message, error) {
	return s.listMessages(ctx, `SELECT m.id, m.chat_id, m.sender_type, m.sender_name, COALESCE(r.avatar, ''), m.role_id, m.content, m.created_at FROM messages m LEFT JOIN roles r ON r.id = m.role_id WHERE m.chat_id = ? AND m.id > ? ORDER BY m.id ASC`, chatID, afterID)
}

func (s *MySQLStore) listMessages(ctx context.Context, query string, args ...any) ([]domain.Message, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []domain.Message
	for rows.Next() {
		var msg domain.Message
		var senderType string
		if err := rows.Scan(&msg.ID, &msg.ChatID, &senderType, &msg.SenderName, &msg.SenderAvatar, &msg.RoleID, &msg.Content, &msg.CreatedAt); err != nil {
			return nil, err
		}
		msg.SenderType = domain.SenderType(senderType)
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

func (s *MySQLStore) CreateTokenUsage(ctx context.Context, usage domain.TokenUsage) (domain.TokenUsage, error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return domain.TokenUsage{}, err
	}
	res, err := s.db.ExecContext(
		ctx,
		`INSERT INTO token_usages (user_id, chat_id, message_id, role_id, model_config_id, model, prompt_tokens, completion_tokens, total_tokens)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID,
		usage.ChatID,
		usage.MessageID,
		nullInt64(usage.RoleID),
		nullInt64(usage.ModelConfigID),
		usage.Model,
		usage.PromptTokens,
		usage.CompletionTokens,
		usage.TotalTokens,
	)
	if err != nil {
		return domain.TokenUsage{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.TokenUsage{}, err
	}
	usage.ID = id
	usage.UserID = userID
	return usage, nil
}

func (s *MySQLStore) TokenUsageStats(ctx context.Context, now time.Time) (domain.TokenUsageStats, error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return domain.TokenUsageStats{}, err
	}
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	recentStart := todayStart.AddDate(0, 0, -6)
	today, err := s.tokenUsageSummary(ctx, userID, todayStart, "今天")
	if err != nil {
		return domain.TokenUsageStats{}, err
	}
	recent, err := s.tokenUsageSummary(ctx, userID, recentStart, "最近 7 天")
	if err != nil {
		return domain.TokenUsageStats{}, err
	}
	byModel, err := s.tokenUsageByModel(ctx, userID, recentStart)
	if err != nil {
		return domain.TokenUsageStats{}, err
	}
	return domain.TokenUsageStats{Today: today, Recent7: recent, ByModel: byModel}, nil
}

func (s *MySQLStore) CreateToolExecution(ctx context.Context, execution domain.ToolExecution) (domain.ToolExecution, error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return domain.ToolExecution{}, err
	}
	res, err := s.db.ExecContext(
		ctx,
		`INSERT INTO tool_executions (user_id, chat_id, message_id, tool_name, input, status, result, error)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		userID,
		execution.ChatID,
		execution.MessageID,
		execution.ToolName,
		execution.Input,
		string(execution.Status),
		execution.Result,
		execution.Error,
	)
	if err != nil {
		return domain.ToolExecution{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.ToolExecution{}, err
	}
	return s.getToolExecution(ctx, id)
}

func (s *MySQLStore) ListToolExecutions(ctx context.Context, chatID int64) ([]domain.ToolExecution, error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, user_id, chat_id, message_id, tool_name, input, status, result, error, created_at
		 FROM tool_executions
		 WHERE user_id = ? AND chat_id = ?
		 ORDER BY created_at DESC, id DESC
		 LIMIT 20`,
		userID,
		chatID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	executions := []domain.ToolExecution{}
	for rows.Next() {
		execution, err := scanToolExecution(rows)
		if err != nil {
			return nil, err
		}
		executions = append(executions, execution)
	}
	return executions, rows.Err()
}

func (s *MySQLStore) getToolExecution(ctx context.Context, id int64) (domain.ToolExecution, error) {
	userID, err := requireUserID(ctx)
	if err != nil {
		return domain.ToolExecution{}, err
	}
	execution, err := scanToolExecution(s.db.QueryRowContext(
		ctx,
		`SELECT id, user_id, chat_id, message_id, tool_name, input, status, result, error, created_at
		 FROM tool_executions
		 WHERE id = ? AND user_id = ?`,
		id,
		userID,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ToolExecution{}, ErrNotFound
	}
	return execution, err
}

func (s *MySQLStore) tokenUsageSummary(ctx context.Context, userID int64, since time.Time, label string) (domain.TokenUsageSummary, error) {
	summary := domain.TokenUsageSummary{Label: label}
	err := s.db.QueryRowContext(
		ctx,
		`SELECT COALESCE(SUM(prompt_tokens), 0), COALESCE(SUM(completion_tokens), 0), COALESCE(SUM(total_tokens), 0)
		 FROM token_usages
		 WHERE user_id = ? AND created_at >= ?`,
		userID,
		since,
	).Scan(&summary.PromptTokens, &summary.CompletionTokens, &summary.TotalTokens)
	return summary, err
}

func (s *MySQLStore) tokenUsageByModel(ctx context.Context, userID int64, since time.Time) ([]domain.TokenUsageSummary, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT model, COALESCE(SUM(prompt_tokens), 0), COALESCE(SUM(completion_tokens), 0), COALESCE(SUM(total_tokens), 0)
		 FROM token_usages
		 WHERE user_id = ? AND created_at >= ?
		 GROUP BY model
		 ORDER BY SUM(total_tokens) DESC, model ASC`,
		userID,
		since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var summaries []domain.TokenUsageSummary
	for rows.Next() {
		var summary domain.TokenUsageSummary
		if err := rows.Scan(&summary.Label, &summary.PromptTokens, &summary.CompletionTokens, &summary.TotalTokens); err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	return summaries, rows.Err()
}

func MySQLDSN(user, password, host, port, database string) string {
	return (MySQLConfig{
		User:     user,
		Password: password,
		Host:     host,
		Port:     port,
		Database: database,
	}).DSN()
}

func (cfg MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true&charset=utf8mb4,utf8", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)
}

func (cfg MySQLConfig) AdminDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/?parseTime=true&multiStatements=true&charset=utf8mb4,utf8", cfg.User, cfg.Password, cfg.Host, cfg.Port)
}

func (cfg MySQLConfig) SafeAddr() string {
	return fmt.Sprintf("%s@tcp(%s:%s)/%s", cfg.User, cfg.Host, cfg.Port, cfg.Database)
}

func validateDatabaseName(name string) error {
	if name == "" {
		return errors.New("mysql database name is required")
	}
	ok, err := regexp.MatchString(`^[A-Za-z0-9_]+$`, name)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("invalid mysql database name %q: only letters, numbers, and underscore are allowed", name)
	}
	return nil
}

func requireUserID(ctx context.Context) (int64, error) {
	userID, ok := domain.UserIDFromContext(ctx)
	if !ok {
		return 0, errors.New("authenticated user is required")
	}
	return userID, nil
}

type modelConfigScanner interface {
	Scan(dest ...any) error
}

type chatFileScanner interface {
	Scan(dest ...any) error
}

type toolExecutionScanner interface {
	Scan(dest ...any) error
}

func scanModelConfig(scanner modelConfigScanner) (domain.ModelConfig, error) {
	var config domain.ModelConfig
	var models string
	if err := scanner.Scan(&config.ID, &config.UserID, &config.Name, &config.Provider, &config.BaseURL, &config.APIKey, &config.DefaultModel, &models, &config.CreatedAt, &config.UpdatedAt); err != nil {
		return domain.ModelConfig{}, err
	}
	config.Models = splitModels(models)
	return config, nil
}

func scanChatFile(scanner chatFileScanner) (domain.ChatFile, error) {
	var file domain.ChatFile
	if err := scanner.Scan(&file.ID, &file.UserID, &file.ChatID, &file.OriginalName, &file.StoragePath, &file.ContentType, &file.SizeBytes, &file.ExtractedText, &file.CreatedAt); err != nil {
		return domain.ChatFile{}, err
	}
	return file, nil
}

func scanToolExecution(scanner toolExecutionScanner) (domain.ToolExecution, error) {
	var execution domain.ToolExecution
	var status string
	if err := scanner.Scan(&execution.ID, &execution.UserID, &execution.ChatID, &execution.MessageID, &execution.ToolName, &execution.Input, &status, &execution.Result, &execution.Error, &execution.CreatedAt); err != nil {
		return domain.ToolExecution{}, err
	}
	execution.Status = domain.ToolExecutionStatus(status)
	return execution, nil
}

func nullInt64(value int64) any {
	if value <= 0 {
		return nil
	}
	return value
}

func splitModels(models string) []string {
	parts := strings.FieldsFunc(models, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ','
	})
	result := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		model := strings.TrimSpace(part)
		if model == "" || seen[model] {
			continue
		}
		seen[model] = true
		result = append(result, model)
	}
	return result
}
