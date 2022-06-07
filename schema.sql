CREATE TABLE `authors` (
	`user_id` varchar(500) NOT NULL,
	`full_name` varchar(500) NOT NULL,
	`email` varchar(320) NOT NULL,
	`bio` varchar(1000) NOT NULL,
	PRIMARY KEY (`user_id`)
) ENGINE InnoDB,
  CHARSET utf8mb4,
  COLLATE utf8mb4_0900_ai_ci;

CREATE TABLE `posts` (
	`slug` varchar(255) NOT NULL,
	`title` varchar(1000) NOT NULL,
	`excerpt` varchar(1000) NOT NULL,
	`content` text NOT NULL DEFAULT (_utf8mb4 ''),
	`modified_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP(),
	PRIMARY KEY (`slug`)
) ENGINE InnoDB,
  CHARSET utf8mb4,
  COLLATE utf8mb4_0900_ai_ci;

CREATE TABLE `posts_authors` (
	`post_slug` varchar(255) NOT NULL,
	`author_user_id` varchar(500) NOT NULL,
	`is_original` tinyint(1) NOT NULL,
	PRIMARY KEY (`post_slug`, `author_user_id`),
	UNIQUE KEY `posts_authors_UN` (`post_slug`, `is_original`)
) ENGINE InnoDB,
  CHARSET utf8mb4,
  COLLATE utf8mb4_0900_ai_ci;

CREATE TABLE `posts_cover_url` (
	`post_slug` varchar(255) NOT NULL,
	`cover_url` varchar(1000) NOT NULL,
	PRIMARY KEY (`post_slug`)
) ENGINE InnoDB,
  CHARSET utf8mb4,
  COLLATE utf8mb4_0900_ai_ci;

CREATE TABLE `posts_publication` (
	`post_slug` varchar(255) NOT NULL,
	`published_at` datetime NOT NULL,
	PRIMARY KEY (`post_slug`)
) ENGINE InnoDB,
  CHARSET utf8mb4,
  COLLATE utf8mb4_0900_ai_ci;