FROM python:3-alpine
MAINTAINER LuHaoqi <LuHaoqi123@sjtu.edu.cn>

WORKDIR /usr/src/app

# 修改成国内源
RUN set -eux && sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories

# 更新操作系统和安装依赖
RUN apk update && apk upgrade && \
    apk add --no-cache bash && \
    apk add --no-cache bash curl && \
    pip install --upgrade pip -i https://pypi.tuna.tsinghua.edu.cn/simple --trusted-host https://pypi.tuna.tsinghua.edu.cn --default-timeout=60

# install python requirements
RUN pip3 install flask requests schedule -i https://pypi.tuna.tsinghua.edu.cn/simple --trusted-host https://pypi.tuna.tsinghua.edu.cn --default-timeout=60

# 如果用户需要安装额外的库，那么还需要提供requirements.txt
COPY requirements.txt ./
RUN pip3 install --no-cache-dir -r requirements.txt -i https://pypi.tuna.tsinghua.edu.cn/simple --trusted-host https://pypi.tuna.tsinghua.edu.cn --default-timeout=60

# 把两个文件添加到image中，这样就不需要挂载了
COPY ./serverless_server.py serverless_server.py
COPY ./my_module.py my_module.py


CMD python serverless_server.py