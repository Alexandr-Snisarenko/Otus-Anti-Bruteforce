-- +goose Up
CREATE TABLE SUBNETS (
  CIDR TEXT NOT NULL,
  LIST_TYPE TEXT NOT NULL CHECK (list_type IN ('blacklist', 'whitelist')),
  DC TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (CIDR, LIST_TYPE)
);

CREATE INDEX idx_subnets_list_type ON subnets(LIST_TYPE);

-- +goose Down

DROP TABLE SUBNETS;