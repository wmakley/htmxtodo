-- Add migration script here
CREATE TABLE public.list
(
	id         BIGSERIAL PRIMARY KEY,
	name       VARCHAR(255) NOT NULL,
	created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

	UNIQUE (name)
);
