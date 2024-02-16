Сервер (storage)
======

Таблицы
-------

- дерево выражения (astnode)
    - astnode_id
    - task_id
    - parent_astnode_id
    - operand1
    - operand2
    - operator
    - operator_delay
    - status (parsing, error, waiting, ready, in progress, done)
    - date_ins
    - date_start
    - date_done
    - agent_id
    - result

- Задания(task)
    - task_id
    - extid
    - expression
    - result
    - task_status (parsing, error, ready, in progress, done)
    - message
 
- Агенты (agent)
    - agent_id
    - date_ping
    - total_procs
    - idle_procs


- Параметры config - параметры времени исполнения арифметических операций и таймаут агента
    - name
    - value

API
---
- /calculate-expression {extid, expression} (POST)
  клиент передает выражение на расчет
  создает в БД task со статусом parsing
  сервер принимает и парсит выражение:
    - парсит выражение в дерево AST и пишет в базу каждый узел со статусом parsing
    - если все успешно распарсилось, ставит у task статус ready
    - если ошибка парсинга, status task = error и в message текст ошибки, у проблемного узла статус error 
 
  Если ОК, возвращает 200 и task_id
  Если невалидно - 400
  Если что-то не так (ЧТО??? нет серверов? ну можем же их подождать. Или нет?) то возвращаем 500
  

- /list-expressions (GET)
  выводим список выражений (все поля)
  *что-то говорили про пагинацию. будет время, добавить ее (параметр offset и настройку page_line_limit)*

- /get-expression-result (GET)
    возвращаем статус и результат выражения, если он готов

- /config-get
  Страница со списком операций в виде пар: имя операции + время его выполнения (доступное для редактирования поле).
  Еще один редактируемый параметр - время  agent_lost_timeout для джоба контроля агентов

- /config-set
  получаем список параметров в виде json и сохраняем в БД таблицу config
  перечитываем config в глобальную мапу 

- /list-agents
    выводит список агентов со всеми полями

- /give-me-astnode
    агент получает AST-узел  
    1. в таблице astnode ищем самое старое задание в статусе ready
    2. возвращаем его в ответе
    3. фиксируем в узле статус in_porgress, date_start и agent_id
    4. если связанный task еще не in_progress, переводим его в in_progress и фиксируем date_start


- /take-oper-result (agent_id, astnode_id, result)
    агент возвращает результат узла
    1. Находим узел
    2. Проверяем, что совпадает статус in_porgress и agent_id. Если нет, возвращаем ошибку
    3. Если совпало, фиксиуруем статус done, rusult, date_done
    4. пытаемся вычислить все выражение:
        - если узел более верхнего уровня, то пищем в задание результат узла и статус done и done_date
        - иначе узел более верхнего уровня должен быть waiting 
        - проверяем оба подчиненных узла. Если оба ready - то прописываем их результаты в operator1 и operator2, переводим статус в ready 
  
- /heartbeat (agent_id, status, total_procs, idle_procs)
    агент сообщает о своем состоянии, фиксируем в таблице агентов date_ping, total_procs, idle_procs

Описание
-----------
1. Стартует
2. Читает конфигурацию (параметры БД и лога)
3. Инициализирует лог
4. Подключается к БД
5. Читает таблицу config и сохраняет в глобальную мапу
6. Запускает горутину контроля


Горутина контроля
-----------------
- регулярно обходит список агентов, проверят date_ping, при превышении  agent_lost_timeout  удаляет агента и списка. Если у него есть активные task'и, 
то переводит их и связанные узлы в состояние new, чтобы их могли взять другие агенты
- перечитывает конфигурацию (таблицу config), на случай если в другом инстансе storage ее поменяли В БД


Агент
=====

Описание
--------
Порт очевидно нужно брать из командной строки, чтобы можно было стартовать несколько агентов сразу
1. Стартует
2. Читает свою конфигурацию (параметры логирования кол-во горутин, адрес и порт сервера storage, частоту опроса сервера storage)
3. Формирует agent_id (хеш от адреса и порта?)
3. Запускает тикер, который дергает /heartbeat
4. Запускает тикер, который оправшивает storage в надежде получить задание

Тикер heartbeat
---------------
1. вызывает storage/heartbeat, передает (agent_id, total_procs, idle_procs)

Тикер poll_task
---------------
1. Проверяет, что кол-во свбодных горутин больше 0
2. Уменьшает счетчик свободных горутин
3. Запускает горутину
    1. Она обращается к storage/give-me-astnode
    2. Вычисляет результат
    3. Делает задержку
    4. Вызывает storage/take-oper-result
4. Увеличивает счетчик свободных горутин




Заметки
=======
Фронтэнд на ГО https://habr.com/ru/articles/475390/
Шаблонизатор https://golangify.com/template-actions-and-functions
супердока по gin  https://www.squash.io/optimizing-gin-in-golang-project-structuring-error-handling-and-testing/


логирование
zap для gin https://pkg.go.dev/github.com/gin-contrib/zap#section-readme

Вычислитель арифметических выражений на ГО https://rosettacode.org/wiki/Arithmetic_Evaluator/Go

Работа с постгресс на ГО https://metanit.com/go/tutorial/10.3.php
https://golangdocs.com/golang-postgresql-example
https://hevodata.com/learn/golang-postgres/
pgx https://henvic.dev/posts/go-postgres/
postgress json fields https://www.alexedwards.net/blog/using-postgresql-jsonb




Работа с конфигурацией
https://zetcode.com/golang/flag/
https://dev.to/ilyakaznacheev/a-clean-way-to-pass-configs-in-a-go-application-1g64
https://habr.com/ru/articles/479882/

REST API 
https://go.dev/doc/tutorial/web-service-gin

Пакеты и модули, импорт и структура проекта
https://www.alexedwards.net/blog/an-introduction-to-packages-imports-and-modules

Docker
https://dev.to/divrhino/build-a-rest-api-from-scratch-with-go-and-docker-3o54

разбейте на мелкие части, например вот такой подход (можно много других придумать):
1. как написать простенький CRUD - чтобы был http сервер, html шаблоны и юзер мог делать пост запрос через форму
2. подключите к нему постгрес - чтобы сохранять задания от юзеров и хранить статус этих заданий
3. можете разобраться как работать с Redis и что такое pub/sub
4. попробуйте написать два простеньких сервиса - один будет публиковать что-то в редис в один канал, а экземпляры другого сервиса - забирать из редиса, выполнять и слать в другой канал
5. теперь пусть первый сервис не только сохраняет в постгрес, но и публикует в редис, а оттуда задание забирает первый попавшийся свободный исполнитель
6. ... много нюансов и допилинга...
7. ...первый сервис должен тоже слушать какой-то канал в редис и забирать готовые задания, плюс обновлять их статус в постгре
8. ...
99. profit 😁

можно и без Redis - просто вычислители будут общаться по TCP, например... или, ещё проще, будут общаться с вашим http сервисом через отдельный роут GET и POST запросами

ну и много много гугла, https://www.phind.com/ какой-нить или чатгпт и норм)
первое время не парьтесь с архитектурой - пусть это даже будут просто два отдельных файла, лишь бы работало...

