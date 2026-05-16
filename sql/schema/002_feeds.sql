-- +goose Up
CREATE TABLE feeds(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    name TEXT NOT NULL,
    url TEXT NOT NULL UNIQUE,
    user_id UUID NOT NULL,
    CONSTRAINT fk_user
        FOREIGN KEY(user_id) 
        REFERENCES users(id)  
        ON DELETE CASCADE 
);

-- +goose Down
DROP TABLE feeds;