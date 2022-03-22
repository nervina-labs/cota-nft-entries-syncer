CREATE TABLE IF NOT EXISTS issuer_infos (
    id bigint NOT NULL AUTO_INCREMENT,
    block_number bigint unsigned NOT NULL,
    lock_hash char(64) NOT NULL,
    lock_hash_crc int unsigned NOT NULL,
    version varchar(40) NOT NULL,
    `name` varchar(255) NOT NULL,
    avatar varchar(500) NOT NULL,
    description varchar(1000) NOT NULL,
    localization json NOT NULL,
    created_at datetime(6) NOT NULL,
    updated_at datetime(6) NOT NULL,
    PRIMARY KEY (id),
    KEY index_issuer_infos_on_block_number (block_number),
    CONSTRAINT uc_issuer_infos_on_cota_id UNIQUE (cota_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE IF NOT EXISTS class_infos (
    id bigint NOT NULL AUTO_INCREMENT,
    block_number bigint unsigned NOT NULL,
    cota_id char(40) NOT NULL,
    version varchar(40) NOT NULL,
    `name` varchar(255) NOT NULL,
    symbol varchar(255) NOT NULL,
    description varchar(1000) NOT NULL,
    image varchar(500) NOT NULL,
    audio varchar(500) NOT NULL,
    video varchar(500) NOT NULL,
    model varchar(500) NOT NULL,
    `schema` json NOT NULL,
    properties json NOT NULL,
    localization json NOT NULL,
    created_at datetime(6) NOT NULL,
    updated_at datetime(6) NOT NULL,
    PRIMARY KEY (id),
    KEY index_class_infos_on_block_number (block_number),
    CONSTRAINT uc_class_infos_on_cota_id UNIQUE (cota_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
