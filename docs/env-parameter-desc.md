##drone环境变量说明
注释括号里面代表原来相对应配置文件的字段


```
SERVER_ADDR=0.0.0.0:9898
REMOTE_DRIVER=sryun
REMOTE_CONFIG=https://omdev.riderzen.com:10080?open=true&skip_verify=true
RC_SRY_REG_INSECURE=true
RC_SRY_REG_HOST=registry:5000
PUBLIC_MODE=true
DATABASE_DRIVER="mysql"
DATABASE_CONFIG="root:111111@tcp(mysql:3306)/drone?parseTime=true"
AGENT_URI=registry:5000/library/drone-exec:latest
PLUGIN_FILTER=registry:5000/library/* plugins/* registry.shurenyun.com/* registry.shurenyun.com/* devregistry.dataman-inc.com/library/*
PLUGIN_PREFIX="library/"
DOCKER_STORAGE=overlay 
DOCKER_EXTRA_HOSTS="registry:REGISTRY harbor:HARBOR"
```
