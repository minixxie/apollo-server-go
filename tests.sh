#!/bin/bash

echo "This should be 200..."
curl -v "http://127.0.0.1:8070/services/config?appId=myAppID&ip=192.168.41.4"

echo "This should be 404..."
curl -v http://127.0.0.1:8070/configs/notExist/myCluster/myNamespace

echo "This should be 404..."
curl -v http://127.0.0.1:8070/configs/myAppID/notExist/myNamespace

echo "This should be 404..."
curl -v http://127.0.0.1:8070/configs/myAppID/myCluster/notExist

echo "This should be 200..."
curl -v http://127.0.0.1:8070/configs/myAppID/myCluster/myNamespace

