CREATE TABLE IF NOT EXISTS chats (
    id BIGINT PRIMARY KEY,
    active_setlist_id INTEGER
);

CREATE TABLE IF NOT EXISTS songs (
    id SERIAL PRIMARY KEY,
    chat_id BIGINT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    tempo INTEGER,
    key TEXT,
    responsible TEXT,
    last_accessed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(chat_id, name)
);

CREATE INDEX IF NOT EXISTS idx_songs_chat_id ON songs(chat_id);
CREATE INDEX IF NOT EXISTS idx_songs_last_accessed ON songs(chat_id, last_accessed_at DESC);

CREATE TABLE IF NOT EXISTS song_notes (
    id SERIAL PRIMARY KEY,
    song_id INTEGER NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_song_notes_song_id ON song_notes(song_id);

CREATE TABLE IF NOT EXISTS song_pins (
    id SERIAL PRIMARY KEY,
    song_id INTEGER NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    label TEXT NOT NULL,
    message_id BIGINT NOT NULL,
    chat_id BIGINT NOT NULL,
    pinned_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_song_pins_song_id ON song_pins(song_id);

CREATE TABLE IF NOT EXISTS song_subscribers (
    song_id INTEGER NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    username TEXT NOT NULL,
    PRIMARY KEY (song_id, user_id)
);

CREATE TABLE IF NOT EXISTS song_history (
    id SERIAL PRIMARY KEY,
    song_id INTEGER NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    field TEXT NOT NULL,
    old_value TEXT,
    new_value TEXT,
    changed_by TEXT NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_song_history_song_id ON song_history(song_id, changed_at DESC);

CREATE TABLE IF NOT EXISTS setlists (
    id SERIAL PRIMARY KEY,
    chat_id BIGINT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(chat_id, name)
);

CREATE INDEX IF NOT EXISTS idx_setlists_chat_id ON setlists(chat_id);

CREATE TABLE IF NOT EXISTS setlist_songs (
    setlist_id INTEGER NOT NULL REFERENCES setlists(id) ON DELETE CASCADE,
    song_id INTEGER NOT NULL REFERENCES songs(id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    PRIMARY KEY (setlist_id, position)
);

CREATE TABLE IF NOT EXISTS user_chat_settings (
    user_id BIGINT NOT NULL,
    chat_id BIGINT NOT NULL,
    username TEXT NOT NULL DEFAULT '',
    subscribe_all BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (user_id, chat_id)
);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_chats_active_setlist'
    ) THEN
        ALTER TABLE chats
            ADD CONSTRAINT fk_chats_active_setlist
            FOREIGN KEY (active_setlist_id) REFERENCES setlists(id) ON DELETE SET NULL;
    END IF;
END
$$;
