CREATE TABLE IF NOT EXISTS users(
	id SERIAL PRIMARY KEY,
	is_guest BOOLEAN NOT NULL,
	is_superuser BOOLEAN NOT NULL,
	username VARCHAR(150) NOT NULL,
	firstname VARCHAR(254) NULL,
	lastname VARCHAR(254) NULL,
	email VARCHAR(254) NULL,
	password VARCHAR(254) NOT NULL
);

ALTER TABLE users ALTER COLUMN password DROP NOT NULL;

CREATE TABLE IF NOT EXISTS category (
	id SERIAL PRIMARY KEY,
	category_name VARCHAR(150) NOT NULL
);

ALTER TABLE category ADD COLUMN slug VARCHAR(200) NOT NULL;

CREATE TABLE IF NOT EXISTS post (
	id SERIAL PRIMARY KEY,
	user_id INT NOT NULL,
	category_id INT NOT NULL,
	title VARCHAR(150) NOT NULL,
	excerpt TEXT NULL,
	read_time VARCHAR(20) NULL,
	datetime TIMESTAMP WITH TIME ZONE NOT NULL,
	message TEXT NOT NULL,
	CONSTRAINT fk_user_post FOREIGN KEY(user_id) REFERENCES users(id),
	CONSTRAINT fk_category_post FOREIGN KEY(category_id) REFERENCES category(id)
);

ALTER TABLE post ALTER COLUMN read_time TYPE INT;
ALTER TABLE post DROP COLUMN excerpt;
ALTER TABLE post ADD COLUMN slug VARCHAR(250);

CREATE TABLE IF NOT EXISTS image (
	id SERIAL PRIMARY KEY,
	image_url VARCHAR(254) NOT NULL,
	category_id INT NULL,
	post_id INT NULL,
	CONSTRAINT fk_category_image FOREIGN KEY(category_id) REFERENCES category(id),
	CONSTRAINT fk_post_image FOREIGN KEY(post_id) REFERENCES post(id)
);

CREATE TABLE IF NOT EXISTS comment (
	id SERIAL PRIMARY KEY,
	user_id INT NOT NULL,
	message TEXT NOT NULL,
	CONSTRAINT fk_user_comment FOREIGN KEY(user_id) REFERENCES users(id)
);

ALTER TABLE comment ADD COLUMN post_id INT NOT NULL;
ALTER TABLE comment ADD CONSTRAINT fk_post_comment FOREIGN KEY(post_id) REFERENCES post(id);

INSERT INTO category(category_name) VALUES('Web Development'), ('Algorithms and Data Structures'), ('New Technologies');
