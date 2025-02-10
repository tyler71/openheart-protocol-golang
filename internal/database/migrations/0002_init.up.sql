START TRANSACTION;
CREATE TABLE site (
                       id INT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
                       url VARCHAR(255) UNIQUE NOT NULL,
                       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                       updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
CREATE TABLE emoji (
                        id INT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
                        site_id INT UNSIGNED NOT NULL,
                        emoji VARCHAR(255) NOT NULL,
                        count INT UNSIGNED DEFAULT 0,
                        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                        FOREIGN KEY (site_id) REFERENCES site(id),
                        INDEX emoji_idx (emoji)
);
COMMIT;