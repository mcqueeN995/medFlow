CREATE TABLE poi (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name varchar(255) NOT NULL,
    type poi_type NOT NULL,
    latitude numeric(10, 8) NOT NULL,
    longitude numeric(11, 8) NOT NULL,
    address text,
    description text,
    photo_url varchar(500),
    rating numeric(3, 2) DEFAULT 0,
    tags text[],
    created_at timestamp DEFAULT now(),
    updated_at timestamp DEFAULT now()
);

CREATE INDEX idx_poi_type ON poi(type);
CREATE INDEX idx_poi_lat_lon ON poi(latitude, longitude);

CREATE TABLE poi_campus_links (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    poi_id uuid NOT NULL,
    campus_id uuid NOT NULL,
    distance_meters int,
    walking_time_seconds int,
    CONSTRAINT fk_poi_campus_links_poi FOREIGN KEY (poi_id) REFERENCES poi(id),
    CONSTRAINT fk_poi_campus_links_campus FOREIGN KEY (campus_id) REFERENCES poi(id),
    CONSTRAINT uq_poi_campus_links UNIQUE (poi_id, campus_id)
);

CREATE INDEX idx_poi_campus_links_poi_id ON poi_campus_links(poi_id);
CREATE INDEX idx_poi_campus_links_campus_id ON poi_campus_links(campus_id);
