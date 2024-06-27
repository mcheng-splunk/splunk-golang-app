#!/bin/sh  
while true  
do 
  curl -X GET http://localhost:8080/hello
  sleep 300
done
