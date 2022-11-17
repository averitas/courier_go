# courier_go

## preparations

### Start mq:

```
 docker run -d --hostname my-rabbit --name some-rabbit -p 8000:15672 -p 8099:5672 rabbitmq:3-management
```

### Start DB

```
docker run --name test-mysql -p 3306:3306 -p 33060:33060 -e MYSQL_ROOT_HOST=172.17.0.1 -e MYSQL_HOST=172.17.0.1 -e MYSQL_ROOT_PASSWORD=my-secret-pw -e MYSQL_USER=user -e MYSQL_DATABASE=test -e MYSQL_PASSWORD=my-secret-pw -d mysql
```

### Create table sql

```
CREATE TABLE `order_models` (
    `order_id` varchar(32),
    `created_at` datetime(3) NULL,
    `updated_at` datetime(3) NULL,
    `deleted_at` datetime(3) NULL,
    `id` varchar(191) unique,
    `name` longtext,
    `prep_time` bigint,
    `order_status` bigint,
    `order_type` varchar(32),
    PRIMARY KEY (`order_id`), INDEX `idx_order_models_id` (`id`))
```

## Build and start server and worker

run ``make`` under root folder.

### Start server
If we have two couriers running, we can start server like this.
```
./apiserver -couriers="http://localhost:8081/ http://localhost:8082/" -addr=:8080
```

### Start worker
```
./worker.exe -addr :8081
``` 

### Start tester to call api
test will send 2 orders per seconds, as the homework required.
```
./tester -type=match # test of Matched dispatch API
./tester -type=fifo # test of First-in-first-out​ dispatch API
```

### Query the average pickup delay
after running tester, we could run sql in db to query average
```
select AVG(tt.pickup_delay) as avg_pickup_delay from
(select order_id, DATE_SUB(timediff(updated_at, created_at), INTERVAL prep_time second) as pickup_delay from order_models) AS tt
```

Or we can use API to query delay

Here is the test result:

GET http://url/api/delay/fifo

result:
```
{
    "Code": 0,
    "Message": "Average dispatch delay is 107.5885"
}
```

GET http://url/api/delay/match

result:
{
    "Code": 0,
    "Message": "Average dispatch delay is 79.5463"
}

## Result

As previous result, the test shows that Matched dispatch strategies will have 79.5463 ms average delay.
First-in-first-out​ dispatch strategies will have 107.5885 ms average delay.
