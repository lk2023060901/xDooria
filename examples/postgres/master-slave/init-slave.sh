#!/bin/bash
set -e

# 等待主库就绪
until pg_isready -h $POSTGRES_MASTER_HOST -p $POSTGRES_MASTER_PORT; do
  echo "Waiting for master to be ready..."
  sleep 2
done

# 清空数据目录
rm -rf "$PGDATA"/*

# 从主库创建基础备份
PGPASSWORD=$POSTGRES_REPLICATION_PASSWORD pg_basebackup \
  -h $POSTGRES_MASTER_HOST \
  -p $POSTGRES_MASTER_PORT \
  -U $POSTGRES_REPLICATION_USER \
  -D "$PGDATA" \
  -Fp -Xs -P -R

# 创建 standby.signal 文件（标记为备库）
touch "$PGDATA/standby.signal"

# 配置连接到主库的信息
cat >> "$PGDATA/postgresql.auto.conf" <<EOF
primary_conninfo = 'host=$POSTGRES_MASTER_HOST port=$POSTGRES_MASTER_PORT user=$POSTGRES_REPLICATION_USER password=$POSTGRES_REPLICATION_PASSWORD'
EOF
