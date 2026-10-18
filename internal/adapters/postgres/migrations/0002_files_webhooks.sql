CREATE TABLE files (
  id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  org_id       bigint NOT NULL REFERENCES orgs(id),
  chat_id      bigint NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
  uploader_id  bigint NOT NULL REFERENCES users(id),
  name         text NOT NULL,
  size         bigint NOT NULL,
  mime         text NOT NULL,
  storage_key  text NOT NULL,
  created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX files_chat_idx ON files (chat_id, id DESC);

-- One webhook per bot. Bots without a row use getUpdates long-polling.
CREATE TABLE bot_webhooks (
  bot_id      bigint PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  url         text NOT NULL,
  secret      text NOT NULL DEFAULT '',
  updated_at  timestamptz NOT NULL DEFAULT now()
);
