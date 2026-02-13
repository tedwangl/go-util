#!/bin/bash
# ProxySQL + MySQL 主从环境快速搭建脚本

set -e

echo "=== ProxySQL + MySQL 主从环境搭建 ==="

# 1. 启动容器
echo "1. 启动 MySQL 主从和 ProxySQL..."
docker-compose -f proxysql.example.yaml up -d

# 等待 MySQL 启动
echo "2. 等待 MySQL 启动（30秒）..."
sleep 30

# 3. 配置主从复制
echo "3. 配置主从复制..."

# 在主库创建复制用户
docker exec mysql-master mysql -uroot -proot123 -e "
CREATE USER IF NOT EXISTS 'repl'@'%' IDENTIFIED WITH mysql_native_password BY 'repl123';
GRANT REPLICATION SLAVE ON *.* TO 'repl'@'%';
FLUSH PRIVILEGES;
"

# 获取主库状态
MASTER_STATUS=$(docker exec mysql-master mysql -uroot -proot123 -e "SHOW MASTER STATUS\G")
MASTER_LOG_FILE=$(echo "$MASTER_STATUS" | grep "File:" | awk '{print $2}')
MASTER_LOG_POS=$(echo "$MASTER_STATUS" | grep "Position:" | awk '{print $2}')

echo "主库状态: File=$MASTER_LOG_FILE, Position=$MASTER_LOG_POS"

# 配置从库 1
echo "配置从库 1..."
docker exec mysql-slave1 mysql -uroot -proot123 -e "
STOP SLAVE;
CHANGE MASTER TO
  MASTER_HOST='mysql-master',
  MASTER_USER='repl',
  MASTER_PASSWORD='repl123',
  MASTER_LOG_FILE='$MASTER_LOG_FILE',
  MASTER_LOG_POS=$MASTER_LOG_POS;
START SLAVE;
"

# 配置从库 2
echo "配置从库 2..."
docker exec mysql-slave2 mysql -uroot -proot123 -e "
STOP SLAVE;
CHANGE MASTER TO
  MASTER_HOST='mysql-master',
  MASTER_USER='repl',
  MASTER_PASSWORD='repl123',
  MASTER_LOG_FILE='$MASTER_LOG_FILE',
  MASTER_LOG_POS=$MASTER_LOG_POS;
START SLAVE;
"

# 4. 创建监控用户
echo "4. 在主库创建 ProxySQL 监控用户..."
docker exec mysql-master mysql -uroot -proot123 -e "
CREATE USER IF NOT EXISTS 'monitor'@'%' IDENTIFIED WITH mysql_native_password BY 'monitor';
GRANT REPLICATION CLIENT ON *.* TO 'monitor'@'%';
FLUSH PRIVILEGES;
"

# 5. 验证配置
echo ""
echo "=== 验证配置 ==="

echo "从库 1 状态:"
docker exec mysql-slave1 mysql -uroot -proot123 -e "SHOW SLAVE STATUS\G" | grep -E "Slave_IO_Running|Slave_SQL_Running"

echo ""
echo "从库 2 状态:"
docker exec mysql-slave2 mysql -uroot -proot123 -e "SHOW SLAVE STATUS\G" | grep -E "Slave_IO_Running|Slave_SQL_Running"

echo ""
echo "=== 环境搭建完成 ==="
echo ""
echo "连接信息："
echo "  应用连接: mysql://root:root123@127.0.0.1:6033/testdb"
echo "  ProxySQL 管理: mysql://admin:admin@127.0.0.1:6032"
echo ""
echo "测试命令："
echo "  mysql -h127.0.0.1 -P6033 -uroot -proot123 testdb"
echo ""
echo "查看 ProxySQL 状态："
echo "  mysql -h127.0.0.1 -P6032 -uadmin -padmin -e 'SELECT * FROM stats_mysql_connection_pool;'"
echo "  mysql -h127.0.0.1 -P6032 -uadmin -padmin -e 'SELECT * FROM stats_mysql_query_rules;'"
