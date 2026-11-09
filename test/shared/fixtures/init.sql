CREATE TABLE items (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

INSERT INTO items (key, value) VALUES ('seeded', 'from-init-script');
