FROM registry.cn-zhangjiakou.aliyuncs.com/vinehoo_v3/alpine:3.14
MAINTAINER "xin.he"
# 设置固定的项目路径
ENV WORKDIR /var/www/main
# 添加应用可执行文件，并设置执行权限
ADD ./main   $WORKDIR/main
RUN chmod +x $WORKDIR/main
WORKDIR $WORKDIR
CMD ["./main"]