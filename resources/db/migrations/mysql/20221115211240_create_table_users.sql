-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    id bigint NOT NULL AUTO_INCREMENT,
    created_at timestamp default CURRENT_TIMESTAMP,
    updated_at timestamp,
    deleted_at timestamp,
    first_name varchar(80) NOT NULL,
    last_name varchar(80) NOT NULL,
    username varchar(120) NOT NULL,
    password varchar(120) NOT NULL,
    email varchar(120) NOT NULL,
    active BOOLEAN NOT NULL default FALSE,
    activation_confirmed_at timestamp,
    CONSTRAINT PK_users PRIMARY KEY (id),
    CONSTRAINT UC_users_username UNIQUE INDEX (username)
);
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TABLE users_roles (
    user_id bigint NOT NULL,
    role_id int NOT NULL,
    created_at timestamp default CURRENT_TIMESTAMP,
    CONSTRAINT PK_users_roles PRIMARY KEY (user_id, role_id),
    CONSTRAINT UC_users_roles_user_id_role_id UNIQUE (user_id, role_id),
    CONSTRAINT FK_users_roles_user_id FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT FK_users_roles_role_id FOREIGN KEY (role_id) REFERENCES roles(id)
);
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TABLE activation_tokens (
    user_id bigint NOT NULL,
    token_hash varchar(80) NOT NULL,
    expires_at timestamp,
    created_at timestamp default CURRENT_TIMESTAMP,
    CONSTRAINT PK_activation_tokens PRIMARY KEY (user_id, token_hash)
);
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TABLE password_reset_tokens (
    user_id bigint NOT NULL,
    token_hash varchar(80) NOT NULL,
    expires_at timestamp NOT NULL,
    created_at timestamp default CURRENT_TIMESTAMP,
    CONSTRAINT PK_password_reset_tokens PRIMARY KEY (user_id, token_hash)
);
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE users;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE users_roles;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE activation_tokens;
-- +goose StatementEnd
-- +goose StatementBegin
DROP TABLE password_reset_tokens;
-- +goose StatementEnd