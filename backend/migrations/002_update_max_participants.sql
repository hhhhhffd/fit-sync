-- Migration: Update max_participants to allow NULL
-- Date: 2025-11-30
-- Purpose: Change max_participants from DEFAULT 5 to NULL (unlimited by default)

-- For existing databases, update the column
-- SQLite doesn't support ALTER COLUMN, so we need to recreate the table

-- Note: This migration is only needed if you have an existing database
-- New databases will use 001_initial_schema.sql which already has NULL

-- To apply this migration manually:
-- 1. Backup your database first!
-- 2. Run these commands in sqlite3:

-- Check current schema
-- PRAGMA table_info(challenges);

-- If max_participants has DEFAULT 5, run:
-- UPDATE challenges SET max_participants = NULL WHERE max_participants = 5;

-- Note: You cannot change the column default in SQLite without recreating the table
-- But existing challenges with max_participants=5 will continue to work
-- New challenges created after updating the code will use NULL (unlimited) by default

