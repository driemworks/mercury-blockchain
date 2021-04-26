rm *.pem
openssl req -x509 -newkey rsa:4096 -days 365 -keyout ca-key.pem -out ca-cert.pem
openssl x509 -in ca-cert.pem -noout -text
openssl req -newkey rsa:4096 -keyout server-key.pem -out server-req.pem
openssl x509 -req -in server-req.pem -days 60 -CA ca-cert.pem -CAkey ca-key.pem -CAcreateserial -out server-cert.pem -extfile server-ext.cnf