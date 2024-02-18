

Инсталляция
===========
0. Клонируем репозиторий git к себе
-----------------------------------
Раз вы читаете readme.md, вероятно вы уже это сделали
Если нет - 
```
git clone https://github.com/Mur466/distribcalc.git
```

1. Установка Postgresql
-----------------------
Скачиваем дистрибутив Postgres 16.2
https://www.enterprisedb.com/downloads/postgres-postgresql-downloads

Запускаем инсталлятор
В процесссе установки указываем пароль для системного пользователя postgres
Для учебного инстанса можно указать такой пароль же как имя - postres
Стандартный порт - 5432

Проверка, что всё установлено командой 
```
pg_config --version
```
Если команда не запускается, проверьте что в вашем PATH есть папка "C:\Program Files\PostgreSQL\16\bin" 

Если postgres установили не локально, а на другой хост/порт (или хотите иметь базу на другом хосте) то нужно в дальнейшем в командах подменять 
localhost на имя вашего хоста, 5432 на номер нужного порта

2. Создание базы данных и объектов в ней
----------------------------------------
Созаем базу distribcalc комадой 
```
createdb -h localhost -p 5432 -U postgres distribcalc
```
После запуска вответ на запрос вводим вышеуказанный пароль 

Создаем таблицы запуском скрипта \distribcalc\database\dbcreate.cmd 
Запускайте команду ниже из папки \distribcalc\database или укажите полный путь к файлу dbcreate.cmd 
```
psql -f dbmigrate.sql -U postgres -h localhost -p 5432 -d distribcalc
```

3. Запуск
--------------------------------------
Запуск сервера
```
cd cd cmd\server
go run main.go 
```

С параметрами по-умолчанию веб-морда сервера доступна по адресу http://localhost:8080/

Запуск агентов:
```
cd cd cmd\agent
go run main.go 
```

Отправка задания через Curl (формат для Windows, нужно все вложенные кавычки экранировать обратным слешем)
```
curl http://localhost:8080/calculate-expression --include --header "Content-Type: application/json" --request "POST" --data "{\"Expr\": \"(1+2)/3*4\", \"ext_id\": \"SomeUniqId12468\"}"
```
go get github.com/gin-gonic/gin
go get -u go.uber.org/zap
go get github.com/jackc/pgx/v5
go get github.com/jackc/pgx/v5/pgxpool
go get github.com/johncgriffin/overflow
go 
```

Postgres 16.2
https://www.enterprisedb.com/downloads/postgres-postgresql-downloads


9223372036854775802+9223372036854775802 = overflow