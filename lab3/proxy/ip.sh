sudo docker exec proxy ip r d default via 10.0.1.1
sudo docker exec proxy ip r a default via 10.0.1.254