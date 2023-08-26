-- migrate:up
CREATE TABLE item
(
	id         BIGSERIAL PRIMARY KEY,
	list_id    BIGINT       NOT NULL REFERENCES list (id),
	position   INT          NOT NULL,
	name       VARCHAR(255) NOT NULL,
	created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX items_list_id_position_idx ON item (list_id, position);

-- migrate:down
DROP TABLE item;
