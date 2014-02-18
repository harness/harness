DELETE FROM builds;
DELETE FROM commits;
DELETE FROM repos;
DELETE FROM members;
DELETE FROM teams;
DELETE FROM users;
DELETE FROM settings;

-- insert users (default password is "password")
INSERT INTO users values (1, 'brad.rydzewski@gmail.com'      , '$2a$10$b8d63QsTL38vx7lj0HEHfOdbu1PCAg6Gfca74UavkXooIBx9YxopS', 'nPmsbl6YNLUIUo0I7gkMcQ' ,'Brad Rydzewski', '8c58a0be77ee441bb8f8595b7f1b4e87', '2013-09-16 00:00:00', '2013-09-16 00:00:00', 1, '', '', '', '', '');
INSERT INTO users values (2, 'thomas.d.burke@gmail.com'      , '$2a$10$b8d63QsTL38vx7lj0HEHfOdbu1PCAg6Gfca74UavkXooIBx9YxopS', 'sal5Tzy6S10yZCaE0jl6QA', 'Thomas Burke',   'c62f7126273f7fa786274274a5dec8ce', '2013-09-16 00:00:00', '2013-09-16 00:00:00', 1, '', '', '', '', '');
INSERT INTO users values (3, 'carlos.morales.duran@gmail.com', '$2a$10$b8d63QsTL38vx7lj0HEHfOdbu1PCAg6Gfca74UavkXooIBx9YxopS', 'bq87o8AmDUOahKApEy2tVQ', 'Carlos Morales', 'c2180a539620d90d68eaeb848364f1c2', '2013-09-16 00:00:00', '2013-09-17 00:00:00', 1, '', '', '', '', '');

-- insert teams
insert into teams values (1, 'drone',  'Drone' , 'brad@drone.io'   , '0057e90a8036c29b1ddb22d0fd08b72c', '2013-09-16 00:00:00', '2013-09-16 00:00:00');
insert into teams values (2, 'google', 'Google', 'dev@google.com'  , '24ba30616d2a20673f54c2aee36d159e', '2013-09-16 00:00:00', '2013-09-16 00:00:00');
insert into teams values (3, 'gradle', 'Gradle', 'dev@gradle.com'  , '5cc3b557e3a3978d52036da9a5be2a08', '2013-09-16 00:00:00', '2013-09-16 00:00:00');
insert into teams values (4, 'dart',   'Dart'  , 'dev@dartlang.org', 'f41fe13f979f2f93cc8b971e1875bdf8', '2013-09-16 00:00:00', '2013-09-16 00:00:00');

-- insert team members
insert into members values (1, 1, 1, 'Owner');
insert into members values (2, 1, 2, 'Admin');
insert into members values (3, 1, 3, 'Write');

-- insert repository
insert into repos values (1, 'github.com/drone/jkl',           'github.com', 'drone',         'jkl',   0, 0, 0, 0, 900, 'git', 'git://github.com/drone/jkl.git',          '', '', '', '', '', '2013-09-16 00:00:00', '2013-09-16 00:00:00', 1, 1);
insert into repos values (2, 'github.com/drone/drone',         'github.com', 'drone',         'drone', 1, 0, 0, 0, 900, 'git', 'git@github.com:drone/drone.git',          '', '', '', '', '', '2013-09-16 00:00:00', '2013-09-16 00:00:00', 1, 1);
insert into repos values (3, 'github.com/bradrydzewski/drone', 'github.com', 'bradrydzewski', 'drone', 1, 0, 0, 0, 900, 'git', 'git@github.com:bradrydzewski/drone.git',  '', '', '', '', '', '2013-09-16 00:00:00', '2013-09-16 00:00:00', 1, 1);
insert into repos values (4, 'github.com/bradrydzewski/blog',  'github.com', 'bradrydzewski', 'blog',  0, 0, 0, 0, 900, 'git', 'git://github.com/bradrydzewski/blog.git', '', '', '', '', '', '2013-09-16 00:00:00', '2013-09-16 00:00:00', 1, 0);

-- insert commits

insert into commits values (1, 1, 'Success', '2013-09-16 00:00:00','2013-09-16 00:00:00', 60, 'ef2221722e6f07a6eaf8af8907b45324428a891d', 'master', '','brad.rydzewski@gmail.com', '8c58a0be77ee441bb8f8595b7f1b4e87', '2013-09-16 00:00:00', 'Fixed mock db class for entity', '2013-09-16 00:00:00', '2013-09-16 00:00:00');
insert into commits values (2, 1, 'Success', '2013-09-16 00:00:00','2013-09-16 00:00:00', 60, '867477aa487d01df28522cee84cd06f5aa154e53', 'master', '','brad.rydzewski@gmail.com', '8c58a0be77ee441bb8f8595b7f1b4e87', '2013-09-16 00:00:00', 'Fixed mock db class for entity', '2013-09-16 00:00:00', '2013-09-16 00:00:00');
insert into commits values (3, 1, 'Success', '2013-09-16 00:00:00','2013-09-16 00:00:00', 60, 'e43427ab462417cb3d53b8702c298c1675deb926', 'master', '','brad.rydzewski@gmail.com', '8c58a0be77ee441bb8f8595b7f1b4e87', '2013-09-16 00:00:00', 'Save deleted entity data to database', '2013-09-16 00:00:00', '2013-09-16 00:00:00');
insert into commits values (4, 1, 'Success', '2013-09-16 00:00:00','2013-09-16 00:00:00', 60, 'a43427ab462417cb3d53b8702c298c1675deb926', 'dev',    '','brad.rydzewski@gmail.com', '8c58a0be77ee441bb8f8595b7f1b4e87', '2013-09-16 00:00:00', 'Save deleted entity data to database', '2013-09-16 00:00:00', '2013-09-16 00:00:00');

-- insert builds

insert into builds values (1, 1, 'node_0.10', 'Success', '2013-09-16 00:00:00','2013-09-16 00:00:00', 60, '2013-09-16 00:00:00','2013-09-16 00:00:00', '');
insert into builds values (2, 1, 'node_0.90', 'Success', '2013-09-16 00:00:00','2013-09-16 00:00:00', 60, '2013-09-16 00:00:00','2013-09-16 00:00:00', '');
insert into builds values (3, 1, 'node_0.80', 'Success', '2013-09-16 00:00:00','2013-09-16 00:00:00', 60, '2013-09-16 00:00:00','2013-09-16 00:00:00', '');
insert into builds values (4, 2, 'node_0.10', 'Success', '2013-09-16 00:00:00','2013-09-16 00:00:00', 60, '2013-09-16 00:00:00','2013-09-16 00:00:00', '');
insert into builds values (5, 2, 'node_0.90', 'Success', '2013-09-16 00:00:00','2013-09-16 00:00:00', 60, '2013-09-16 00:00:00','2013-09-16 00:00:00', '');
insert into builds values (6, 2, 'node_0.80', 'Success', '2013-09-16 00:00:00','2013-09-16 00:00:00', 60, '2013-09-16 00:00:00','2013-09-16 00:00:00', '');
insert into builds values (7, 3, 'node_0.10', 'Success', '2013-09-16 00:00:00','2013-09-16 00:00:00', 60, '2013-09-16 00:00:00','2013-09-16 00:00:00', '');
insert into builds values (8, 3, 'node_0.90', 'Success', '2013-09-16 00:00:00','2013-09-16 00:00:00', 60, '2013-09-16 00:00:00','2013-09-16 00:00:00', '');
insert into builds values (9, 3, 'node_0.80', 'Success', '2013-09-16 00:00:00','2013-09-16 00:00:00', 60, '2013-09-16 00:00:00','2013-09-16 00:00:00', '');

-- insert default, dummy settings

insert into settings values (1,'','','github.com','https://api.github.com','','','','','','','','localhost:8080','http',0);

-- add public & private keys to all repositories

update repos set public_key = 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAAgQCybgl9+Y0VY0mKng3AB3CwCMAOVvg+Xh4X/4lP7SR815GaeEJQusaA0p33HkZfS/2XREWYMtiopHP0bZuBIht76JdhrJlHh1AcLoPQvWJROFvRGol6igVEVZzs9sUdZaPrexFz1CS/j6BJFzPsHnL4gXT3s4PYYST9++pThI90Aw==';

update repos set private_key = '-----BEGIN RSA PRIVATE KEY-----
MIICWwIBAAKBgQCybgl9+Y0VY0mKng3AB3CwCMAOVvg+Xh4X/4lP7SR815GaeEJQ
usaA0p33HkZfS/2XREWYMtiopHP0bZuBIht76JdhrJlHh1AcLoPQvWJROFvRGol6
igVEVZzs9sUdZaPrexFz1CS/j6BJFzPsHnL4gXT3s4PYYST9++pThI90AwIDAQAB
AoGAaxvs7MdaLsWcRu7cGDMfLT0DdVg1ytKaxBMsrWMQrTSGfjDEtkt4j6pfExIE
cn5ea2ibUmLrdkjKJqeJWrpLvlOZGhahBcL/SueFOfr6Lm+m8LvlTrX6JhyLXpx5
NbeEFr0mN16PC6JqkN0xRCN9BfV9m6gnpuP/ojD3RKYMZtkCQQDFbSX/ddEfp9ME
vRNAYif+bFxI6PEgMmwrCIjJGHOsq7zba3Z7KWjW034x2rJ3Cbhs8xtyTcA5qy9F
OzL3pFs3AkEA514SUXowIiqjh6ypnSvUBaQZsWjexDxTXN09DTYPt+Ck1qdzTHWP
9nerg2G3B6bTOWZBftHMaZ/plZ/eyV0LlQJACU1rTO4wPF2cA80k6xO07rgMYSMY
uXumvSBZ0Z/lU22EKJKXspXw6q5sc8zqO9GpbvjFgk1HkXAPeiOf8ys7YQJAD1CI
wd/mo7xSyr5BE+g8xorQMJASfsbHddQnIGK9s5wpDRRUa3E0sEnHjpC/PsBqJth/
6VcVwsAVBBRq+MUx6QJAS9KKxKcMf8JpnDheV7jh+WJKckabA1L2bq8sN6kXfPn0
o7deiE1FKJizXKJ6gd6anfuG3m7VAs7wJhzc685yMg==
-----END RSA PRIVATE KEY-----';

-- add standard output to all builds

update builds set stdout = '$ mvn test
-------------------------------------------------------
 T E S T S
-------------------------------------------------------
Running brooklyn.qa.longevity.MonitorUtilsTest
Configuring TestNG with: TestNG652Configurator
[GC 69952K->6701K(253440K), 0.0505760 secs]
2013-08-21 21:12:58,327 INFO  TESTNG RUNNING: Suite: "Command line test" containing "7" Tests (config: null)
2013-08-21 21:12:58,342 INFO  BrooklynLeakListener.onStart attempting to terminate all extant ManagementContexts: name=Command line test; includedGroups=[]; excludedGroups=[Integration, Acceptance, Live, WIP]; suiteName=brooklyn.qa.longevity.MonitorUtilsTest; outDir=/scratch/jenkins/workspace/brooklyncentral/brooklyn/usage/qa/target/surefire-reports/brooklyn.qa.longevity.MonitorUtilsTest
2013-08-21 21:12:58,473 INFO  TESTNG INVOKING: "Command line test" - brooklyn.qa.longevity.MonitorUtilsTest.testFindOwnPid()
2013-08-21 21:12:58,939 INFO  executing cmd: ps -p 7484
2013-08-21 21:12:59,030 INFO  TESTNG PASSED: "Command line test" - brooklyn.qa.longevity.MonitorUtilsTest.testFindOwnPid() finished in 595 ms
2013-08-21 21:12:59,033 INFO  TESTNG INVOKING: "Command line test" - brooklyn.qa.longevity.MonitorUtilsTest.testGetRunningPids()
2013-08-21 21:12:59,035 INFO  executing cmd: ps ax
2013-08-21 21:12:59,137 INFO  TESTNG PASSED: "Command line test" - brooklyn.qa.longevity.MonitorUtilsTest.testGetRunningPids() finished in 104 ms
2013-08-21 21:12:59,139 INFO  TESTNG INVOKING: "Command line test" - brooklyn.qa.longevity.MonitorUtilsTest.testGroovyExecuteAndWaitForConsumingOutputStream()
2013-08-21 21:12:59,295 INFO  TESTNG PASSED: "Command line test" - brooklyn.qa.longevity.MonitorUtilsTest.testGroovyExecuteAndWaitForConsumingOutputStream() finished in 155 ms
2013-08-21 21:12:59,298 INFO  TESTNG INVOKING: "Command line test" - brooklyn.qa.longevity.MonitorUtilsTest.testIsPidRunning()
2013-08-21 21:12:59,300 INFO  executing cmd: ps ax
2013-08-21 21:12:59,384 INFO  executing cmd: ps -p 7484
2013-08-21 21:12:59,391 INFO  executing cmd: ps -p 10000
2013-08-21 21:12:59,443 INFO  pid 10000 not running: 
2013-08-21 21:12:59,446 INFO  executing cmd: ps -p 1234567
2013-08-21 21:12:59,455 INFO  pid 1234567 not running: 
2013-08-21 21:12:59,456 INFO  TESTNG PASSED: "Command line test" - brooklyn.qa.longevity.MonitorUtilsTest.testIsPidRunning() finished in 158 ms
2013-08-21 21:12:59,481 INFO  TESTNG INVOKING: "Command line test" - brooklyn.qa.longevity.MonitorUtilsTest.testIsUrlUp()
[GC 76653K->7013K(253440K), 0.0729880 secs]
2013-08-21 21:13:00,726 INFO  Error reading URL http://localhost/thispathdoesnotexist: org.apache.http.conn.HttpHostConnectException: Connection to http://localhost refused
2013-08-21 21:13:00,727 INFO  TESTNG PASSED: "Command line test" - brooklyn.qa.longevity.MonitorUtilsTest.testIsUrlUp() finished in 1246 ms
2013-08-21 21:13:00,760 INFO  TESTNG INVOKING: "Command line test" - brooklyn.qa.longevity.MonitorUtilsTest.testMemoryUsage()
2013-08-21 21:13:00,762 INFO  executing cmd: jmap -histo 7484
2013-08-21 21:13:02,275 INFO  executing cmd: jmap -histo 7484
2013-08-21 21:13:03,690 INFO  executing cmd: jmap -histo 7484
2013-08-21 21:13:04,725 INFO  TESTNG PASSED: "Command line test" - brooklyn.qa.longevity.MonitorUtilsTest.testMemoryUsage() finished in 3965 ms
2013-08-21 21:13:04,752 INFO  TESTNG INVOKING: "Command line test" - brooklyn.qa.longevity.MonitorUtilsTest.testSearchLog()
2013-08-21 21:13:04,816 INFO  executing cmd: grep -E line1 /tmp/monitorUtilsTest.testSearchLog2369184699231420767.txt
2013-08-21 21:13:04,848 INFO  executing cmd: grep -E line1|line2 /tmp/monitorUtilsTest.testSearchLog2369184699231420767.txt
2013-08-21 21:13:04,854 INFO  executing cmd: grep -E textnotthere /tmp/monitorUtilsTest.testSearchLog2369184699231420767.txt
2013-08-21 21:13:04,858 INFO  executing cmd: grep -E line /tmp/monitorUtilsTest.testSearchLog2369184699231420767.txt
2013-08-21 21:13:04,897 INFO  TESTNG PASSED: "Command line test" - brooklyn.qa.longevity.MonitorUtilsTest.testSearchLog() finished in 145 ms
2013-08-21 21:13:04,917 INFO  TESTNG 
===============================================
    Command line test
    Tests run: 7, Failures: 0, Skips: 0
===============================================
2013-08-21 21:13:04,944 INFO  BrooklynLeakListener.onFinish attempting to terminate all extant ManagementContexts: name=Command line test; includedGroups=[]; excludedGroups=[Integration, Acceptance, Live, WIP]; suiteName=brooklyn.qa.longevity.MonitorUtilsTest; outDir=/scratch/jenkins/workspace/brooklyncentral/brooklyn/usage/qa/target/surefire-reports/brooklyn.qa.longevity.MonitorUtilsTest
Tests run: 7, Failures: 0, Errors: 0, Skipped: 0, Time elapsed: 10.849 sec

Results :

Tests run: 7, Failures: 0, Errors: 0, Skipped: 0';
