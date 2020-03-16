#!/bin/bash

#pulling mysql from docker and executing the image
echo initialization of the database... 
#sudo su
#sudo docker pull mysql/mysql-server
#echo docker image is pulled and stored...
#sudo docker run --name=mysql1 -d mysql/mysql-server:tag
#echo docker mysql image is running...

#downloading locally
sudo yum update
sudo yum install wget -y
wget http://repo.mysql.com/mysql-community-release-el7-5.noarch.rpm
echo mysql package is downloaded...
sudo rpm -ivh mysql-community-release-el7-5.noarch.rpm
sudo yum update
echo updating yum...
sudo yum install mysql-server -y
echo installing mysql-server...
sudo systemctl start mysqld
echo starting mysql-server...

#creating sql user and logging in
echo Please provide the credentials for the database.
echo Username :
read -r username
echo Password :
read -r password
mysql -u root -p
\n
GRANT ALL PRIVILEGES ON *.* TO "$username"@'localhost' IDENTIFIED BY "$password";
\q
mysql -u "$username" -p
echo "$password"
echo mysql user is created with username:$username

#creating the db
CREATE DATABASE kdb;
USE kdb;
CREATE TABLE user (id int(10) not null primary key auto_increment, username varchar(30) not null unique, token varchar(30) unique, access_level varchar(10), password varchar(100), first_name varchar(30), last_name varchar(30), email varchar(50));
CREATE TABLE cluster (id int(10) not null primary key auto_increment, cluster_name varchar(30) unique, kafka_version varchar(10), active_controllers int(10));
CREATE TABLE broker (id int(10) not null primary key auto_increment, host varchar(100) unique, port int(10), created_at datetime, cluster_id int(10) not null);
echo Tables created :
DESCRIBE user;
DESCRIBE cluster;
DESCRIBE broker;
\q

