SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: fiber_storage; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.fiber_storage (
    k character varying(64) DEFAULT ''::character varying NOT NULL,
    v bytea NOT NULL,
    e bigint DEFAULT '0'::bigint NOT NULL
);


--
-- Name: item; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.item (
    id bigint NOT NULL,
    list_id bigint NOT NULL,
    "position" integer NOT NULL,
    name character varying(255) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: item_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.item_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: item_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.item_id_seq OWNED BY public.item.id;


--
-- Name: list; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.list (
    id bigint NOT NULL,
    name character varying(255) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: list_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.list_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: list_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.list_id_seq OWNED BY public.list.id;


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_migrations (
    version character varying(128) NOT NULL
);


--
-- Name: item id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.item ALTER COLUMN id SET DEFAULT nextval('public.item_id_seq'::regclass);


--
-- Name: list id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.list ALTER COLUMN id SET DEFAULT nextval('public.list_id_seq'::regclass);


--
-- Name: fiber_storage fiber_storage_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.fiber_storage
    ADD CONSTRAINT fiber_storage_pkey PRIMARY KEY (k);


--
-- Name: item item_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.item
    ADD CONSTRAINT item_pkey PRIMARY KEY (id);


--
-- Name: list list_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.list
    ADD CONSTRAINT list_name_key UNIQUE (name);


--
-- Name: list list_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.list
    ADD CONSTRAINT list_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: e; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX e ON public.fiber_storage USING btree (e);


--
-- Name: items_list_id_position_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX items_list_id_position_idx ON public.item USING btree (list_id, "position");


--
-- Name: item item_list_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.item
    ADD CONSTRAINT item_list_id_fkey FOREIGN KEY (list_id) REFERENCES public.list(id);


--
-- PostgreSQL database dump complete
--


--
-- Dbmate schema migrations
--

INSERT INTO public.schema_migrations (version) VALUES
    ('20230810152727'),
    ('20230810153011');
