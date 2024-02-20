#!/bin/bash


######## Run this bash script with to install Mongodb on your system ##########

# HOWTO:
	# System recommendation : Ubuntu 16.04	
	# Download the script
	# Open in bash and make it executable with command: chmod +x mongodb_install.sh
	# Run with command: sudo ./mongodb_install.sh

# OTHER USEFUL SCRIPTS:
	# For making a MongoDB replica set with three nodes: https://gist.github.com/Maria-UET/af4332f6dd9e57f2d0f6141dbb8dd447
	# For initating the MongoDB replica set after configuration: https://gist.github.com/Maria-UET/af4332f6dd9e57f2d0f6141dbb8dd447




# Add an appropriate username for your MongoDB user
USR="admin"
# Add an appropriate password for your MongoDB user. Password should be ideally read from a config.ini file, keeping passwords in bash scripts is not secure.
PASS="password123"
DB="admin"
ROLE="root"
BIND_IP=0.0.0.0


# Importing the Public Key
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv 68818C72E52529D4


echo ""
echo "installing mongodb dependencies"
sudo apt install mongodb-clients -y
sudo apt-get update && sudo apt-get upgrade -y


echo ""
echo "Installing mongodb dependencies libcurl3 openssl"
sudo apt-get install libcurl3 openssl


echo ""
echo "Creating source list file mongodb"
# Uncomment the line below for ubuntu 18.04
# sudo echo "deb http://repo.mongodb.org/apt/ubuntu bionic/mongodb-org/4.0 multiverse" | sudo tee /etc/apt/sources.list.d/mongodb-org-4.0.list 
sudo bash -c 'echo "deb http://repo.mongodb.org/apt/ubuntu xenial/mongodb-org/4.0 multiverse" > /etc/apt/sources.list.d/mongodb-org-4.0.list' # for Ubuntu 16.04


sudo apt-get update


echo ""
echo "Installing mongodb-org"
sudo apt-get install -y mongodb-org


echo ""
echo "Making a directory for db data"
sudo mkdir -p /data/db 

# If you are not the root user, change owner of the /data/db directory -> uncomment the block below 
# echo ""
# echo "Changing permisions for the db directory "
# sudo chown -R "maria" /data/db 

 
# Start MongoDB and add it as a service to be started at boot time:
sudo systemctl start mongod
sudo systemctl enable mongod
sudo systemctl restart mongod

echo ""
echo " ############################## Mongodb has been installed  ###############################"
echo " ##      Check status of mongod server by running this command: sudo netstat -plntu      ##"
echo " ##########################################################################################"

export LC_ALL=C


echo ""
echo "Creating mongodb user"
mongo admin --eval "db.createUser({'user':'$USR', 'pwd':'$PASS', 'roles':[{'role':'$ROLE', 'db':'$DB'}]})"


sudo systemctl stop mongod


echo ""
echo "Configuring the mongod.conf file to update bindip and enable authentication"
sudo sed -i[bindIp] "s/bindIp: /bindIp: $BIND_IP #/g" /etc/mongod.conf 
# Do not enable ip_bind_all without enabling authorization. otherwise, the db will be exposed.
sudo echo  "#authorization config
security:
   authorization: enabled" >> /etc/mongod.conf


echo ""
echo "Appending --auth to mongod.service to enable authentication"
sudo sed -i '/ExecStart/ s/$/ --auth/' /lib/systemd/system/mongod.service

sudo systemctl enable mongod.service


echo ""
echo "Initiating daemon-reload"
sudo systemctl daemon-reload


sudo service mongod stop


echo ""
echo "Starting the mongod server with the following parameters:"
echo "sudo mongod --bind_ip_all --fork --logpath /var/log/mongodb.log"
sudo mongod --bind_ip $BIND_IP --fork --logpath /var/log/mongodb.log


echo "Starting the mongo shell with following parameters:"
echo "mongo -u $USR -p $PASS"
mongo -u $USR -p $PASS 



echo "You can connect to this db from remote client using: mongo --username $USR --password $PASS ipaddress:27017/db_name --authenticationDatabase $DB"