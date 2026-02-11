-- Create the url_shortener table
CREATE TABLE IF NOT EXISTS url_shortener (
    shortkey VARCHAR(10) NOT NULL PRIMARY KEY,
    url VARCHAR(2048) NOT NULL UNIQUE,
    validfrom BIGINT DEFAULT 0,
    validtill BIGINT DEFAULT 0
);