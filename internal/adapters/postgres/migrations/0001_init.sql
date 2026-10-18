CREATE TABLE orgs (
  id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  name          text NOT NULL,
  email_domain  text NOT NULL DEFAULT '',
  created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
  id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  org_id         bigint NOT NULL REFERENCES orgs(id),
  kind           text NOT NULL CHECK (kind IN ('user','bot')),
  username       text NOT NULL,
  display_name   text NOT NULL,
  email          text NOT NULL DEFAULT '',
  department     text NOT NULL DEFAULT '',
  title          text NOT NULL DEFAULT '',
  role           text NOT NULL DEFAULT 'member' CHECK (role IN ('admin','member')),
  status         text NOT NULL DEFAULT 'active' CHECK (status IN ('active','suspended')),
  password_hash  text,
  bot_token_hash text,
  created_by     bigint REFERENCES users(id),
  created_at     timestamptz NOT NULL DEFAULT now(),
  UNIQUE (org_id, username)
);
CREATE UNIQUE INDEX users_bot_token_hash_idx ON users (bot_token_hash) WHERE bot_token_hash IS NOT NULL;
CREATE INDEX users_org_display_idx ON users (org_id, display_name);

CREATE TABLE sessions (
  token_hash    text PRIMARY KEY,
  user_id       bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at    timestamptz NOT NULL DEFAULT now(),
  expires_at    timestamptz NOT NULL,
  last_seen_at  timestamptz NOT NULL DEFAULT now(),
  user_agent    text NOT NULL DEFAULT ''
);
CREATE INDEX sessions_user_idx ON sessions (user_id);

CREATE TABLE chats (
  id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  org_id      bigint NOT NULL REFERENCES orgs(id),
  type        text NOT NULL CHECK (type IN ('direct','group','channel')),
  title       text NOT NULL DEFAULT '',
  direct_key  text,
  created_by  bigint NOT NULL REFERENCES users(id),
  last_seq    bigint NOT NULL DEFAULT 0,
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX chats_direct_key_idx ON chats (org_id, direct_key) WHERE direct_key IS NOT NULL;

CREATE TABLE chat_members (
  chat_id    bigint NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
  user_id    bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role       text NOT NULL DEFAULT 'member' CHECK (role IN ('owner','admin','member')),
  read_seq   bigint NOT NULL DEFAULT 0,
  joined_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (chat_id, user_id)
);
CREATE INDEX chat_members_user_idx ON chat_members (user_id);

CREATE TABLE messages (
  id             bigint GENERATED ALWAYS AS IDENTITY UNIQUE,
  chat_id        bigint NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
  seq            bigint NOT NULL,
  sender_id      bigint NOT NULL REFERENCES users(id),
  client_msg_id  text,
  kind           text NOT NULL CHECK (kind IN ('text','system','file')),
  body           text NOT NULL DEFAULT '',
  attachment     jsonb,
  created_at     timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (chat_id, seq)
);
CREATE UNIQUE INDEX messages_client_msg_idx ON messages (sender_id, client_msg_id) WHERE client_msg_id IS NOT NULL;

CREATE TABLE audit_log (
  id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  org_id      bigint NOT NULL REFERENCES orgs(id),
  actor_id    bigint NOT NULL REFERENCES users(id),
  action      text NOT NULL,
  target      text NOT NULL DEFAULT '',
  meta        jsonb,
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX audit_log_org_time_idx ON audit_log (org_id, created_at DESC);
