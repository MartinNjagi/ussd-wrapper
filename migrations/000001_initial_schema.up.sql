-- ====================
-- Drop tables if they exist (safe rollback)
-- ====================
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS ussd_sessions;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS system_config;
DROP TABLE IF EXISTS ussd_menus;
DROP TABLE IF EXISTS users;

-- ====================
-- Users table
-- ====================
CREATE TABLE users
(
    id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    phone_number VARCHAR(20) UNIQUE NOT NULL,
    pin          VARCHAR(255)       NOT NULL,
    first_name   VARCHAR(100),
    last_name    VARCHAR(100),
    balance      DECIMAL(20, 2)                        DEFAULT 0.00,
    status       ENUM ('active', 'inactive', 'banned') DEFAULT 'active',
    created_at   TIMESTAMP                             DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP                             DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- ====================
-- Transactions table
-- ====================
CREATE TABLE transactions
(
    id               BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    reference_id     VARCHAR(50) UNIQUE                         NOT NULL,
    transaction_type ENUM ('deposit', 'withdrawal', 'transfer') NOT NULL,
    sender_id        BIGINT UNSIGNED,
    recipient_id     BIGINT UNSIGNED,
    amount           DECIMAL(20, 2)                             NOT NULL,
    fee              DECIMAL(10, 2) DEFAULT 0.00,
    status           ENUM ('pending', 'completed', 'failed')    NOT NULL,
    description      TEXT,
    metadata         JSON,
    created_at       TIMESTAMP      DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP      DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (sender_id) REFERENCES users (id) ON DELETE SET NULL,
    FOREIGN KEY (recipient_id) REFERENCES users (id) ON DELETE SET NULL
);

-- ====================
-- USSD Sessions table
-- ====================
CREATE TABLE ussd_sessions
(
    id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    session_id   VARCHAR(100) UNIQUE NOT NULL,
    phone_number VARCHAR(20)         NOT NULL,
    current_menu VARCHAR(50),
    data         JSON,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    expires_at   TIMESTAMP
);

-- ====================
-- Audit Logs table
-- ====================
CREATE TABLE audit_logs
(
    id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id     BIGINT UNSIGNED,
    action      VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50),
    entity_id   VARCHAR(50),
    old_value   JSON,
    new_value   JSON,
    ip_address  VARCHAR(50),
    user_agent  TEXT,
    created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL
);

-- ====================
-- System Config table
-- ====================
CREATE TABLE system_config
(
    setting_key VARCHAR(100) PRIMARY KEY,
    value       TEXT,
    description TEXT,
    updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    updated_by  BIGINT UNSIGNED,
    FOREIGN KEY (updated_by) REFERENCES users (id) ON DELETE SET NULL
);

-- ====================
-- USSD Menus Configuration table
-- ====================
CREATE TABLE ussd_menus
(
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    menu_key      VARCHAR(100) UNIQUE NOT NULL,
    title         TEXT                NOT NULL,
    options       JSON,
    parent_menu   VARCHAR(100),
    requires_auth BOOLEAN   DEFAULT FALSE,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- ====================
-- Indexes
-- ====================
CREATE INDEX idx_transactions_reference_id ON transactions(reference_id);
CREATE INDEX idx_transactions_sender_id ON transactions(sender_id);
CREATE INDEX idx_transactions_recipient_id ON transactions(recipient_id);
CREATE INDEX idx_ussd_sessions_session_id ON ussd_sessions(session_id);
CREATE INDEX idx_ussd_sessions_phone_number ON ussd_sessions(phone_number);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_entity_type_id ON audit_logs(entity_type, entity_id);

-- ====================
-- Optional Seed Data
-- ====================
INSERT INTO users (phone_number, pin, first_name, last_name, balance)
VALUES ('254700000001', 'hashed_pin_example', 'Jane', 'Doe', 500.00);

INSERT INTO system_config (setting_key, value, description, updated_by)
VALUES ('maintenance_mode', 'off', 'Enable or disable system-wide maintenance mode', 1);
