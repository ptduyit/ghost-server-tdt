## create selfsign cert
*openssl req -x509 -newkey rsa:4096 -nodes -keyout cycrax.key -out cycrax.crt -days 1337 -subj "/CN=Cy Crax /O=CyCrax Inc /OU=cycrax.io /C=VN"*


*openssl genrsa -out service.key 4096*


*openssl req -new -key service.key -out service.csr -subj /CN=tool.kidti.com -addext subjectAltName=DNS:tool.kidti.com*


*openssl x509 -req -days 1337 -in service.csr -CA cycrax.crt -CAkey cycrax.key -CAcreateserial -out service.crt*


*openssl pkcs12 -export -out cycrax.p12 -inkey cycrax.key -in cycrax.crt*


## Install Cert
1. Double click file cycrax.crt
2. Install Certificate...
3. Local Machine -> Next
4. Browse -> Trusted Root Certification Authorities -> Ok -> Next
5. Finish



go build -ldflags -H=windowsgui -o fakeserver.exe main.go