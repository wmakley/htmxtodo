-- migrate:up
CREATE TABLE items (
    id SERIAL PRIMARY KEY,
    list_id BIGINT NOT NULL REFERENCES lists(id),
    position INT NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX items_list_id_position_idx ON items(list_id, position);

-- migrate:down
DROP TABLE items;
