CREATE TABLE customer (
    id INT(6)  AUTO_INCREMENT PRIMARY KEY,
    firstname VARCHAR(30) NOT NULL,
    lastname VARCHAR(30) NOT NULL,
    email VARCHAR(50),
    regdate TIMESTAMP
);
INSERT INTO `customer` (`id`, `firstname`, `lastname`, `email`, `regdate`) VALUES (1, 'Roger', 'Federer', 'roger.federer@yomail.com', '2019-01-21 20:21:49');
INSERT INTO `customer` (`id`, `firstname`, `lastname`, `email`, `regdate`) VALUES (2, 'Rafael', 'Nadal', 'rafael.nadal@yomail.com', '2019-01-22 20:21:49');
INSERT INTO `customer` (`id`, `firstname`, `lastname`, `email`, `regdate`) VALUES (3, 'John', 'Mcenroe', 'john.mcenroe@yomail.com', '2019-01-23 20:21:49');
INSERT INTO `customer` (`id`, `firstname`, `lastname`, `email`, `regdate`) VALUES (4, 'Ivan', 'Lendl', 'ivan.lendl@yomail.com', '2019-01-23 23:21:49');
INSERT INTO `customer` (`id`, `firstname`, `lastname`, `email`, `regdate`) VALUES (5, 'Jimmy', 'Connors', 'jimmy.connors@yomail.com', '2019-01-23 22:21:49');


CREATE TABLE `resource_role`(
    `id` INT UNSIGNED AUTO_INCREMENT,
    `user_id` INT NOT NULL,
    `resource_id` INT NOT NULL,
    `role_id` INT NOT NULL,
    `create_time` BIGINT(20) NOT NULL COMMENT 'create_time',
    `update_time` BIGINT(20) NOT NULL COMMENT 'update_time',
    `delete_time` BIGINT(20) DEFAULT 0 COMMENT 'delete_time',
    PRIMARY KEY ( `id` ),
    UNIQUE KEY `user_resource` (user_id, resource_id),
    KEY `update_time` (update_time)
)ENGINE=InnoDB DEFAULT CHARSET=utf8;



create table `resource` (
    `id` INT UNSIGNED AUTO_INCREMENT,
    `name` VARCHAR(100) NOT NULL COMMENT 'resource name',
    `description` VARCHAR(100) NOT NULL COMMENT 'resource description',
    `create_time` BIGINT(20) NOT NULL COMMENT 'create_time',
    `update_time` BIGINT(20) NOT NULL COMMENT 'update_time',
    `delete_time` BIGINT(20) DEFAULT 0 COMMENT 'delete_time',
    PRIMARY KEY ( `id` ),
    KEY `update_time` (update_time)
)ENGINE=InnoDB DEFAULT CHARSET=utf8;



insert into resource(name, description, create_time, update_time, delete_time) values('name1', 'description1', 1628952454, 1628952454, 0);