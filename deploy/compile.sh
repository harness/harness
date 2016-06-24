#!/bin/bash


# 获取dockerfiles/Dockerfile_compile_env 最后一行作为编译代码环境的镜像(比如go环境，python环境)
# 最后一行样例(#demoregistry.dataman-inc.com/library/centos7-go1.5.4:v0.1.061500)
code_compile_image=`tail -n 1 $SERVICE/dockerfiles/Dockerfile_compile_env`
code_compile_image=${code_compile_image#*#}
# 生成 compile.sh
cat > /data/build/compile.sh << EOF
#!/bin/bash
export GOPATH="/usr/local/go"
mkdir -p /usr/local/go/src/github.com/Dataman-Cloud
rm -rf /usr/local/go/src/github.com/Dataman-Cloud/$SERVICE
cp -r $SERVICE /usr/local/go/src/github.com/Dataman-Cloud/
cd /usr/local/go/src/github.com/Dataman-Cloud/$SERVICE
make build
# 将编译完成的二进制文件放到/data/build/$SERVICE/目录, 作为运行时的镜像dockerfile ADD 使用
cp $SERVICE /data/build/$SERVICE/
EOF

chmod +x /data/build/compile.sh

# 编译二进制
# docker run --rm -e SERVICE="$SERVICE" -v /tmp/codebuild:/data/build -w="/data/build" $code_compile_image /bin/bash -c "bash -x compile.sh"
