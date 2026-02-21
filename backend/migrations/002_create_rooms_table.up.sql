-- migrations/002_create_rooms_table.sql
-- Creates the rooms table for chat rooms

CREATE TABLE IF NOT EXISTS rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    invite_token VARCHAR(64) NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_rooms_invite_token ON rooms(invite_token);
CREATE INDEX IF NOT EXISTS idx_rooms_owner_id ON rooms(owner_id);
