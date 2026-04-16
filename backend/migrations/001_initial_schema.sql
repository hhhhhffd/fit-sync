-- Users table with all fields
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT,
    login TEXT,
    password_hash TEXT,
    phone TEXT,
    telegram_id INTEGER,
    name TEXT,
    age INTEGER,
    height REAL,
    weight REAL,
    photo_url TEXT,
    description TEXT,
    total_wins INTEGER DEFAULT 0 CHECK(total_wins >= 0),
    current_streak INTEGER DEFAULT 0 CHECK(current_streak >= 0),
    best_streak INTEGER DEFAULT 0 CHECK(best_streak >= 0),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create unique indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE email IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_login ON users(login) WHERE login IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_telegram_id ON users(telegram_id) WHERE telegram_id IS NOT NULL;

-- Activities table
CREATE TABLE IF NOT EXISTS activities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    activity_type TEXT NOT NULL,
    duration INTEGER,
    distance REAL,
    calories INTEGER,
    notes TEXT,
    activity_date DATE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Challenges table
CREATE TABLE IF NOT EXISTS challenges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    creator_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL DEFAULT 'accumulative', -- accumulative, consistency
    goal_value INTEGER NOT NULL DEFAULT 0, -- numeric target for accumulative, days count for consistency
    max_participants INTEGER, -- NULL = unlimited participants
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    status TEXT DEFAULT 'pending', -- pending, active, completed
    winner_id INTEGER,
    invite_code TEXT UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (creator_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (winner_id) REFERENCES users(id) ON DELETE SET NULL
);

-- Challenge participants table
CREATE TABLE IF NOT EXISTS challenge_participants (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    challenge_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    total_points INTEGER DEFAULT 0,
    current_progress INTEGER DEFAULT 0, -- accumulative: sum of values, consistency: days count
    UNIQUE(challenge_id, user_id),
    FOREIGN KEY (challenge_id) REFERENCES challenges(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Challenge logs table for progress tracking
CREATE TABLE IF NOT EXISTS challenge_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    challenge_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    value INTEGER, -- for accumulative: added value, for consistency: 1 (check-in)
    photo_file_id TEXT, -- optional Telegram file_id for photo proof
    notes TEXT,
    logged_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (challenge_id) REFERENCES challenges(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Achievements table
CREATE TABLE IF NOT EXISTS achievements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    requirement_type TEXT NOT NULL, -- wins, streak, coins
    requirement_value INTEGER NOT NULL,
    icon TEXT
);

-- User achievements table
CREATE TABLE IF NOT EXISTS user_achievements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    achievement_id INTEGER NOT NULL,
    unlocked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, achievement_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (achievement_id) REFERENCES achievements(id) ON DELETE CASCADE
);

-- Friendships table
CREATE TABLE IF NOT EXISTS friendships (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    friend_id INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, friend_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (friend_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Insert default achievements
INSERT OR IGNORE INTO achievements (name, description, requirement_type, requirement_value, icon) VALUES
    ('First Blood', 'Win your first challenge', 'wins', 5, '🥉'),
    ('Rising Star', 'Win 10 challenges', 'wins', 10, '🥈'),
    ('Champion', 'Win 15 challenges', 'wins', 15, '🥇'),
    ('Legend', 'Win 20 challenges', 'wins', 20, '🏆'),
    ('Master', 'Win 30 challenges', 'wins', 30, '👑'),
    ('Grandmaster', 'Win 50 challenges', 'wins', 50, '💎'),
    ('Ultimate', 'Win 100 challenges', 'wins', 100, '🌟'),
    ('On Fire', 'Achieve 3 win streak', 'streak', 3, '🔥'),
    ('Unstoppable', 'Achieve 5 win streak', 'streak', 5, '⚡'),
    ('Godlike', 'Achieve 10 win streak', 'streak', 10, '💫');

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_activities_user_date ON activities(user_id, activity_date);
CREATE INDEX IF NOT EXISTS idx_challenges_status ON challenges(status);
CREATE INDEX IF NOT EXISTS idx_challenges_invite_code ON challenges(invite_code);
CREATE INDEX IF NOT EXISTS idx_challenge_participants_challenge ON challenge_participants(challenge_id);
CREATE INDEX IF NOT EXISTS idx_challenge_logs_challenge ON challenge_logs(challenge_id);
CREATE INDEX IF NOT EXISTS idx_challenge_logs_user ON challenge_logs(user_id, challenge_id);

-- Indexes for leaderboard queries
CREATE INDEX IF NOT EXISTS idx_users_total_wins ON users(total_wins DESC);
CREATE INDEX IF NOT EXISTS idx_users_best_streak ON users(best_streak DESC);
