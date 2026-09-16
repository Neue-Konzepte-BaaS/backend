\restrict dbmate

-- Dumped from database version 16.13 (Ubuntu 16.13-0ubuntu0.24.04.1)
-- Dumped by pg_dump version 16.13 (Ubuntu 16.13-0ubuntu0.24.04.1)

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

--
-- Name: btree_gist; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS btree_gist WITH SCHEMA public;


--
-- Name: EXTENSION btree_gist; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION btree_gist IS 'support for indexing common datatypes in GiST';


--
-- Name: postgis; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS postgis WITH SCHEMA public;


--
-- Name: EXTENSION postgis; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION postgis IS 'PostGIS geometry and geography spatial types and functions';


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: account; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.account (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    email text NOT NULL,
    password_hash text NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: admin; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.admin (
    account_id uuid NOT NULL,
    role integer NOT NULL
);


--
-- Name: announcement; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.announcement (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    farmer uuid NOT NULL,
    subject text NOT NULL,
    body text NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: crop; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.crop (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    duration_months integer NOT NULL,
    CONSTRAINT crop_duration_months_check CHECK ((duration_months > 0))
);


--
-- Name: customer; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.customer (
    account_id uuid NOT NULL,
    postal_code integer NOT NULL
);


--
-- Name: farmer; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.farmer (
    account_id uuid NOT NULL,
    farm_name text NOT NULL,
    postal_code integer NOT NULL
);


--
-- Name: field; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.field (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    farmer uuid NOT NULL,
    coordinates public.geometry(Polygon,4326) NOT NULL
);


--
-- Name: plot; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.plot (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    name text NOT NULL,
    field uuid NOT NULL,
    coordinates public.geometry(Polygon,4326) NOT NULL
);


--
-- Name: plot_crop; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.plot_crop (
    plot uuid NOT NULL,
    crop uuid NOT NULL
);


--
-- Name: postal_code; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.postal_code (
    id integer NOT NULL,
    zipcode text NOT NULL,
    name text NOT NULL,
    coordinates public.geometry(Point,4326) NOT NULL
);


--
-- Name: postal_code_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.postal_code_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: postal_code_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.postal_code_id_seq OWNED BY public.postal_code.id;


--
-- Name: rental; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.rental (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    plot uuid NOT NULL,
    customer uuid NOT NULL,
    period tstzrange NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    crop uuid NOT NULL
);


--
-- Name: schema_migrations; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.schema_migrations (
    version character varying NOT NULL
);


--
-- Name: postal_code id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.postal_code ALTER COLUMN id SET DEFAULT nextval('public.postal_code_id_seq'::regclass);


--
-- Name: account account_email_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account
    ADD CONSTRAINT account_email_key UNIQUE (email);


--
-- Name: account account_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.account
    ADD CONSTRAINT account_pkey PRIMARY KEY (id);


--
-- Name: admin admin_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admin
    ADD CONSTRAINT admin_pkey PRIMARY KEY (account_id);


--
-- Name: announcement announcement_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.announcement
    ADD CONSTRAINT announcement_pkey PRIMARY KEY (id);


--
-- Name: crop crop_name_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.crop
    ADD CONSTRAINT crop_name_key UNIQUE (name);


--
-- Name: crop crop_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.crop
    ADD CONSTRAINT crop_pkey PRIMARY KEY (id);


--
-- Name: customer customer_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.customer
    ADD CONSTRAINT customer_pkey PRIMARY KEY (account_id);


--
-- Name: farmer farmer_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.farmer
    ADD CONSTRAINT farmer_pkey PRIMARY KEY (account_id);


--
-- Name: field field_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.field
    ADD CONSTRAINT field_pkey PRIMARY KEY (id);


--
-- Name: plot_crop plot_crop_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plot_crop
    ADD CONSTRAINT plot_crop_pkey PRIMARY KEY (plot, crop);


--
-- Name: plot plot_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plot
    ADD CONSTRAINT plot_pkey PRIMARY KEY (id);


--
-- Name: postal_code postal_code_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.postal_code
    ADD CONSTRAINT postal_code_pkey PRIMARY KEY (id);


--
-- Name: rental rental_no_overlap; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rental
    ADD CONSTRAINT rental_no_overlap EXCLUDE USING gist (plot WITH =, period WITH &&);


--
-- Name: rental rental_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rental
    ADD CONSTRAINT rental_pkey PRIMARY KEY (id);


--
-- Name: schema_migrations schema_migrations_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.schema_migrations
    ADD CONSTRAINT schema_migrations_pkey PRIMARY KEY (version);


--
-- Name: idx_account_email; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_account_email ON public.account USING btree (email);


--
-- Name: idx_announcement_farmer_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_announcement_farmer_created ON public.announcement USING btree (farmer, created_at DESC);


--
-- Name: idx_field_farmer; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_field_farmer ON public.field USING btree (farmer);


--
-- Name: idx_plot_coordinates; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_plot_coordinates ON public.plot USING gist (coordinates);


--
-- Name: idx_plot_field; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_plot_field ON public.plot USING btree (field);


--
-- Name: idx_postal_code_coordinates; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_postal_code_coordinates ON public.postal_code USING gist (coordinates);


--
-- Name: idx_postal_code_name; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_postal_code_name ON public.postal_code USING btree (name);


--
-- Name: idx_postal_code_zipcode; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_postal_code_zipcode ON public.postal_code USING btree (zipcode);


--
-- Name: idx_rental_customer; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_rental_customer ON public.rental USING btree (customer);


--
-- Name: announcement fk_announcement_farmer; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.announcement
    ADD CONSTRAINT fk_announcement_farmer FOREIGN KEY (farmer) REFERENCES public.farmer(account_id) ON DELETE CASCADE;


--
-- Name: admin fk_farmer_account; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admin
    ADD CONSTRAINT fk_farmer_account FOREIGN KEY (account_id) REFERENCES public.account(id) ON DELETE CASCADE;


--
-- Name: customer fk_farmer_account; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.customer
    ADD CONSTRAINT fk_farmer_account FOREIGN KEY (account_id) REFERENCES public.account(id) ON DELETE CASCADE;


--
-- Name: farmer fk_farmer_account; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.farmer
    ADD CONSTRAINT fk_farmer_account FOREIGN KEY (account_id) REFERENCES public.account(id) ON DELETE CASCADE;


--
-- Name: field fk_field_farmer; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.field
    ADD CONSTRAINT fk_field_farmer FOREIGN KEY (farmer) REFERENCES public.farmer(account_id);


--
-- Name: plot_crop fk_plot_crop_crop; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plot_crop
    ADD CONSTRAINT fk_plot_crop_crop FOREIGN KEY (crop) REFERENCES public.crop(id);


--
-- Name: plot_crop fk_plot_crop_plot; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plot_crop
    ADD CONSTRAINT fk_plot_crop_plot FOREIGN KEY (plot) REFERENCES public.plot(id) ON DELETE CASCADE;


--
-- Name: plot fk_plot_field; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.plot
    ADD CONSTRAINT fk_plot_field FOREIGN KEY (field) REFERENCES public.field(id);


--
-- Name: rental fk_rental_crop; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rental
    ADD CONSTRAINT fk_rental_crop FOREIGN KEY (crop) REFERENCES public.crop(id);


--
-- Name: rental fk_rental_customer; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rental
    ADD CONSTRAINT fk_rental_customer FOREIGN KEY (customer) REFERENCES public.customer(account_id);


--
-- Name: rental fk_rental_plot; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rental
    ADD CONSTRAINT fk_rental_plot FOREIGN KEY (plot) REFERENCES public.plot(id);


--
-- PostgreSQL database dump complete
--

\unrestrict dbmate


--
-- Dbmate schema migrations
--

INSERT INTO public.schema_migrations (version) VALUES
    ('20260909071744'),
    ('20260911090000'),
    ('20260913081624'),
    ('20260913114701'),
    ('20260913144639'),
    ('20260913155400'),
    ('20260913165138'),
    ('20260913170000'),
    ('20260914103000'),
    ('20260914130000'),
    ('20260915135248');
