CREATE TABLE IF NOT EXISTS matches (
  id SERIAL PRIMARY KEY,
  match_date DATE NOT NULL,
  opponent TEXT NOT NULL,
  venue TEXT NOT NULL CHECK (venue IN ('home','away','neutral')),
  goals_for INT NOT NULL CHECK (goals_for >= 0),
  goals_against INT NOT NULL CHECK (goals_against >= 0),
  season INT NOT NULL,
  UNIQUE (match_date, opponent)
);
CREATE INDEX IF NOT EXISTS idx_matches_season_venue ON matches (season, venue);
