-- Production Database Initialization for WaveFlix Hub
-- This script sets up the database schema, indexes, and initial data for production

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Create production-optimized schemas
CREATE SCHEMA IF NOT EXISTS waveflix;
CREATE SCHEMA IF NOT EXISTS analytics;
CREATE SCHEMA IF NOT EXISTS cache;

-- Set default search path
ALTER DATABASE waveflix SET search_path TO waveflix, public;

-- Users table with enhanced security
CREATE TABLE IF NOT EXISTS waveflix.users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    display_name VARCHAR(100),
    avatar_url TEXT,
    is_verified BOOLEAN DEFAULT FALSE,
    is_active BOOLEAN DEFAULT TRUE,
    last_login TIMESTAMPTZ,
    login_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- User preferences
CREATE TABLE IF NOT EXISTS waveflix.user_preferences (
    user_id UUID PRIMARY KEY REFERENCES waveflix.users(id) ON DELETE CASCADE,
    theme VARCHAR(20) DEFAULT 'dark',
    language VARCHAR(10) DEFAULT 'en',
    adult_content BOOLEAN DEFAULT FALSE,
    email_notifications BOOLEAN DEFAULT TRUE,
    push_notifications BOOLEAN DEFAULT TRUE,
    auto_play BOOLEAN DEFAULT TRUE,
    quality_preference VARCHAR(20) DEFAULT 'auto',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Movies/Shows data (cached from TMDB)
CREATE TABLE IF NOT EXISTS waveflix.media_items (
    id INTEGER PRIMARY KEY, -- TMDB ID
    media_type VARCHAR(20) NOT NULL CHECK (media_type IN ('movie', 'tv')),
    title VARCHAR(500) NOT NULL,
    original_title VARCHAR(500),
    overview TEXT,
    release_date DATE,
    poster_path VARCHAR(200),
    backdrop_path VARCHAR(200),
    vote_average DECIMAL(3,1),
    vote_count INTEGER,
    popularity DECIMAL(8,3),
    adult BOOLEAN DEFAULT FALSE,
    original_language VARCHAR(10),
    genre_ids INTEGER[],
    runtime INTEGER,
    status VARCHAR(50),
    tagline TEXT,
    imdb_id VARCHAR(20),
    budget BIGINT,
    revenue BIGINT,
    homepage TEXT,
    cached_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- User favorites/watchlist
CREATE TABLE IF NOT EXISTS waveflix.user_favorites (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES waveflix.users(id) ON DELETE CASCADE,
    media_id INTEGER NOT NULL,
    media_type VARCHAR(20) NOT NULL,
    list_type VARCHAR(20) NOT NULL CHECK (list_type IN ('favorite', 'watchlist', 'watched')),
    rating INTEGER CHECK (rating >= 1 AND rating <= 10),
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, media_id, media_type, list_type)
);

-- User reviews
CREATE TABLE IF NOT EXISTS waveflix.user_reviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES waveflix.users(id) ON DELETE CASCADE,
    media_id INTEGER NOT NULL,
    media_type VARCHAR(20) NOT NULL,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 10),
    review_text TEXT,
    is_spoiler BOOLEAN DEFAULT FALSE,
    helpful_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, media_id, media_type)
);

-- Analytics tables for production insights
CREATE TABLE IF NOT EXISTS analytics.page_views (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES waveflix.users(id),
    session_id VARCHAR(100),
    path VARCHAR(500) NOT NULL,
    referrer TEXT,
    user_agent TEXT,
    ip_address INET,
    country VARCHAR(2),
    city VARCHAR(100),
    duration_ms INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS analytics.api_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    endpoint VARCHAR(200) NOT NULL,
    method VARCHAR(10) NOT NULL,
    status_code INTEGER NOT NULL,
    response_time_ms INTEGER NOT NULL,
    user_id UUID REFERENCES waveflix.users(id),
    ip_address INET,
    user_agent TEXT,
    request_size INTEGER,
    response_size INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS analytics.search_queries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES waveflix.users(id),
    query VARCHAR(500) NOT NULL,
    results_count INTEGER,
    clicked_result_id INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Cache tables for performance
CREATE TABLE IF NOT EXISTS cache.tmdb_cache (
    cache_key VARCHAR(500) PRIMARY KEY,
    cache_data JSONB NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS cache.image_cache (
    url VARCHAR(1000) PRIMARY KEY,
    optimized_url VARCHAR(1000),
    file_size INTEGER,
    format VARCHAR(10),
    width INTEGER,
    height INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_email ON waveflix.users(email) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_username ON waveflix.users(username) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_active ON waveflix.users(is_active, created_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_last_login ON waveflix.users(last_login);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_media_items_type ON waveflix.media_items(media_type);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_media_items_popularity ON waveflix.media_items(popularity DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_media_items_vote_average ON waveflix.media_items(vote_average DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_media_items_release_date ON waveflix.media_items(release_date DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_media_items_genre_ids ON waveflix.media_items USING GIN(genre_ids);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_media_items_title_search ON waveflix.media_items USING GIN(to_tsvector('english', title || ' ' || COALESCE(overview, '')));

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_favorites_user_id ON waveflix.user_favorites(user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_favorites_media ON waveflix.user_favorites(media_id, media_type);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_favorites_list_type ON waveflix.user_favorites(list_type, created_at DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_reviews_user_id ON waveflix.user_reviews(user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_reviews_media ON waveflix.user_reviews(media_id, media_type);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_reviews_rating ON waveflix.user_reviews(rating, created_at DESC);

-- Analytics indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_page_views_user_id ON analytics.page_views(user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_page_views_created_at ON analytics.page_views(created_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_page_views_path ON analytics.page_views(path);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_requests_endpoint ON analytics.api_requests(endpoint);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_requests_created_at ON analytics.api_requests(created_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_api_requests_status ON analytics.api_requests(status_code);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_search_queries_user_id ON analytics.search_queries(user_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_search_queries_created_at ON analytics.search_queries(created_at);

-- Cache indexes
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tmdb_cache_expires_at ON cache.tmdb_cache(expires_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_image_cache_created_at ON cache.image_cache(created_at);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply updated_at triggers
DROP TRIGGER IF EXISTS update_users_updated_at ON waveflix.users;
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON waveflix.users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_user_preferences_updated_at ON waveflix.user_preferences;
CREATE TRIGGER update_user_preferences_updated_at BEFORE UPDATE ON waveflix.user_preferences FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_media_items_updated_at ON waveflix.media_items;
CREATE TRIGGER update_media_items_updated_at BEFORE UPDATE ON waveflix.media_items FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_user_favorites_updated_at ON waveflix.user_favorites;
CREATE TRIGGER update_user_favorites_updated_at BEFORE UPDATE ON waveflix.user_favorites FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_user_reviews_updated_at ON waveflix.user_reviews;
CREATE TRIGGER update_user_reviews_updated_at BEFORE UPDATE ON waveflix.user_reviews FOR EACH ROW EXECUTE FUNCTION update_updated_at_updated_at();

-- Create database functions for common operations
CREATE OR REPLACE FUNCTION waveflix.get_trending_media(media_type_param VARCHAR DEFAULT NULL, limit_param INTEGER DEFAULT 20)
RETURNS TABLE (
    id INTEGER,
    media_type VARCHAR,
    title VARCHAR,
    poster_path VARCHAR,
    vote_average DECIMAL,
    popularity DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT m.id, m.media_type, m.title, m.poster_path, m.vote_average, m.popularity
    FROM waveflix.media_items m
    WHERE (media_type_param IS NULL OR m.media_type = media_type_param)
    AND m.cached_at > NOW() - INTERVAL '24 hours'
    ORDER BY m.popularity DESC
    LIMIT limit_param;
END;
$$ LANGUAGE plpgsql;

-- Function to clean up old cache entries
CREATE OR REPLACE FUNCTION cache.cleanup_expired_cache()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM cache.tmdb_cache WHERE expires_at < NOW();
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    -- Also clean up old analytics data (keep last 90 days)
    DELETE FROM analytics.page_views WHERE created_at < NOW() - INTERVAL '90 days';
    DELETE FROM analytics.api_requests WHERE created_at < NOW() - INTERVAL '90 days';
    DELETE FROM analytics.search_queries WHERE created_at < NOW() - INTERVAL '90 days';
    
    -- Clean up old image cache (keep last 30 days)
    DELETE FROM cache.image_cache WHERE created_at < NOW() - INTERVAL '30 days';
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Create roles for production security
DO $$
BEGIN
    -- Application role with limited permissions
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'waveflix_app') THEN
        CREATE ROLE waveflix_app WITH LOGIN PASSWORD 'REPLACE_WITH_STRONG_PASSWORD';
    END IF;
    
    -- Read-only role for analytics/reporting
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'waveflix_readonly') THEN
        CREATE ROLE waveflix_readonly WITH LOGIN PASSWORD 'REPLACE_WITH_READONLY_PASSWORD';
    END IF;
END
$$;

-- Grant appropriate permissions
GRANT USAGE ON SCHEMA waveflix TO waveflix_app;
GRANT USAGE ON SCHEMA analytics TO waveflix_app;
GRANT USAGE ON SCHEMA cache TO waveflix_app;

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA waveflix TO waveflix_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA analytics TO waveflix_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA cache TO waveflix_app;

GRANT USAGE ON ALL SEQUENCES IN SCHEMA waveflix TO waveflix_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA waveflix TO waveflix_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA cache TO waveflix_app;

-- Read-only permissions
GRANT USAGE ON SCHEMA waveflix TO waveflix_readonly;
GRANT USAGE ON SCHEMA analytics TO waveflix_readonly;
GRANT USAGE ON SCHEMA cache TO waveflix_readonly;

GRANT SELECT ON ALL TABLES IN SCHEMA waveflix TO waveflix_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA analytics TO waveflix_readonly;
GRANT SELECT ON ALL TABLES IN SCHEMA cache TO waveflix_readonly;

-- Insert initial data for testing
INSERT INTO waveflix.users (id, username, email, password_hash, display_name, is_verified) 
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'admin',
    'admin@waveflix.local',
    '$2a$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewKyNiQkMF6k8sPS', -- password: admin123
    'Administrator',
    true
) ON CONFLICT (username) DO NOTHING;

INSERT INTO waveflix.user_preferences (user_id)
VALUES ('00000000-0000-0000-0000-000000000001')
ON CONFLICT (user_id) DO NOTHING;

-- Create a function to populate sample data (for development/testing)
CREATE OR REPLACE FUNCTION waveflix.populate_sample_data()
RETURNS VOID AS $$
BEGIN
    -- Insert sample movies (popular titles for demo)
    INSERT INTO waveflix.media_items (id, media_type, title, overview, release_date, poster_path, vote_average, popularity) VALUES
    (550, 'movie', 'Fight Club', 'A ticking-time-bomb insomniac and a slippery soap salesman channel primal male aggression into a shocking new form of therapy.', '1999-10-15', '/pB8BM7pdSp6B6Ih7QZ4DrQ3PmJK.jpg', 8.4, 63.416),
    (13, 'movie', 'Forrest Gump', 'A man with a low IQ has accomplished great things in his life and been present during significant historic events.', '1994-06-23', '/arw2vcBveWOVZr6pxd9XTd1TdQa.jpg', 8.5, 48.307),
    (157336, 'movie', 'Interstellar', 'The adventures of a group of explorers who make use of a newly discovered wormhole to surpass the limitations on human space travel.', '2014-11-07', '/gEU2QniE6E77NI6lCU6MxlNBvIx.jpg', 8.4, 32.213),
    (27205, 'movie', 'Inception', 'Cobb, a skilled thief who commits corporate espionage by infiltrating the subconscious of his targets is offered a chance to regain his old life.', '2010-07-16', '/9gk7adHYeDvHkCSEqAvQNLV5Uge.jpg', 8.4, 29.108)
    ON CONFLICT (id) DO UPDATE SET
        title = EXCLUDED.title,
        overview = EXCLUDED.overview,
        cached_at = NOW();
END;
$$ LANGUAGE plpgsql;

-- Set up automatic cache cleanup (requires pg_cron extension if available)
-- This should be run periodically via cron or application scheduler
-- SELECT cron.schedule('cache-cleanup', '0 2 * * *', 'SELECT cache.cleanup_expired_cache();');

COMMIT;