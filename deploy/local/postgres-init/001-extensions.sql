-- Extensions required by some DIGIT service migrations (e.g. Registry uses uuid_generate_v4()).
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

