#!/bin/bash
set -e

# 创建复制用户
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE USER $POSTGRES_REPLICATION_USER WITH REPLICATION ENCRYPTED PASSWORD '$POSTGRES_REPLICATION_PASSWORD';
    SELECT pg_create_physical_replication_slot('replication_slot_1');
    SELECT pg_create_physical_replication_slot('replication_slot_2');
EOSQL

# 配置 pg_hba.conf 允许复制连接
echo "host replication $POSTGRES_REPLICATION_USER all md5" >> "$PGDATA/pg_hba.conf"

# 重新加载配置
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    SELECT pg_reload_conf();
EOSQL
