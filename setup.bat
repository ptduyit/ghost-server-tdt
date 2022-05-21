@echo off
certutil -addstore root cycrax.crt
if errorlevel 1 goto err
del cycrax.crt
echo . >> C:\Windows\System32\drivers\etc\hosts
echo 127.65.43.21 www.tietdattai.org >> C:\Windows\System32\drivers\etc\hosts

:err
echo need admin permiss
