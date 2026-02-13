# ProxySQL DNS 配置方案

## 方案 B：DNS + read_only 自动检测（推荐）

### 架构

```
ProxySQL
  ↓ (定期检查 read_only)
  ├─→ mysql-node1.internal (read_only=0) → hostgroup=0 (写)
  ├─→ mysql-node2.internal (read_only=1) → hostgroup=1 (读)
  └─→ mysql-node3.internal (read_only=1) → hostgroup=1 (读)

主从切换后：
  ├─→ mysql-node1.internal (read_only=1) → hostgroup=1 (读)
  ├─→ mysql-node2.internal (read_only=0) → hostgroup=0 (写) ← 新主库
  └─→ mysql-node3.internal (read_only=1) → hostgroup=1 (读)
```

### 1. DNS 配置

#### 内网 DNS（推荐）

```
# /etc/hosts 或内网 DNS 服务器
mysql-node1.internal    192.168.1.10
mysql-node2.internal    192.168.1.11
mysql-node3.internal    192.168.1.12
```

#### Kubernetes

```yaml
# MySQL StatefulSet 自动提供 DNS
apiVersion: v1
kind: Service
metadata:
  name: mysql
spec:
  clusterIP: None  # Headless Service
  selector:
    app: mysql

# DNS 自动生成：
# mysql-0.mysql.default.svc.cluster.local
# mysql-1.mysql.default.svc.cluster.local
# mysql-2.mysql.default.svc.cluster.local
```

#### Docker Compose

```yaml
services:
  mysql-node1:
    image: mysql:8.0
    container_name: mysql-node1
    hostname: mysql-node1.internal
    networks:
      - db-network

  mysql-node2:
    image: mysql:8.0
    container_name: mysql-node2
    hostname: mysql-node2.internal
    networks:
      - db-network

  mysql-node3:
    image: mysql:8.0
    container_name: mysql-node3
    hostname: mysql-node3.internal
    networks:
      - db-network

networks:
  db-network:
    driver: bridge
```

### 2. ProxySQL 配置

```cnf
# proxysql.cnf

# 所有节点初始都在 hostgroup=0
mysql_servers = (
    { address="mysql-node1.internal", port=3306, hostgroup=0 },
    { address="mysql-node2.internal", port=3306, hostgroup=0 },
    { address="mysql-node3.internal", port=3306, hostgroup=0 },
)

# ProxySQL 自动检测 read_only 并分组
mysql_replication_hostgroups = (
    {
        writer_hostgroup=0
        reader_hostgroup=1
        check_type="read_only"
    }
)

# 监控配置
mysql_variables = {
    monitor_username="monitor"
    monitor_password="monitor"
    monitor_read_only_interval=1500   # 每 1.5 秒检查一次
    monitor_read_only_timeout=500
}
```

### 3. MySQL 配置

#### 主库
```sql
-- 主库：read_only=0
SET GLOBAL read_only=0;
SET GLOBAL super_read_only=0;

-- 创建监控用户
CREATE USER 'monitor'@'%' IDENTIFIED BY 'monitor';
GRANT REPLICATION CLIENT ON *.* TO 'monitor'@'%';
```

#### 从库
```sql
-- 从库：read_only=1
SET GLOBAL read_only=1;
SET GLOBAL super_read_only=1;
```

### 4. 主从切换流程

#### 手动切换

```bash
# 1. 旧主库降级为从库
mysql-node1> SET GLOBAL read_only=1;

# 2. 新主库提升
mysql-node2> STOP SLAVE;
mysql-node2> RESET SLAVE ALL;
mysql-node2> SET GLOBAL read_only=0;

# 3. ProxySQL 自动检测（1-2 秒内）
# 无需重启 ProxySQL 或修改配置
```

#### 使用 MHA/Orchestrator（自动）

```bash
# MHA 自动执行：
# 1. 检测主库故障
# 2. 选择最新的从库
# 3. 提升为主库（SET GLOBAL read_only=0）
# 4. 其他从库指向新主库
# 5. ProxySQL 自动检测并切换路由
```

### 5. 验证

```bash
# 查看 ProxySQL 服务器分组
mysql -h127.0.0.1 -P6032 -uadmin -padmin -e "
SELECT hostgroup_id, hostname, port, status, 
       Queries, Bytes_data_sent, Bytes_data_recv
FROM stats_mysql_connection_pool
ORDER BY hostgroup_id;
"

# 预期输出：
# hostgroup_id | hostname              | status | Queries
# 0            | mysql-node2.internal  | ONLINE | 1000    (写)
# 1            | mysql-node1.internal  | ONLINE | 5000    (读)
# 1            | mysql-node3.internal  | ONLINE | 5000    (读)

# 查看 read_only 检测日志
mysql -h127.0.0.1 -P6032 -uadmin -padmin -e "
SELECT * FROM monitor.mysql_server_read_only_log
ORDER BY time_start_us DESC LIMIT 10;
"
```

### 6. 优势

1. **自动切换**：主从切换后 ProxySQL 自动重新分组
2. **无需修改配置**：DNS 域名不变，ProxySQL 配置不变
3. **应用无感知**：应用只连 ProxySQL，完全透明
4. **故障恢复快**：1-2 秒内完成切换
5. **支持多环境**：dev/test/prod 用不同 DNS

### 7. 注意事项

#### DNS 缓存
```bash
# ProxySQL 会缓存 DNS 解析结果
# 如果 IP 变更，需要重启 ProxySQL 或等待 TTL 过期

# 建议：DNS TTL 设置较短（60 秒）
```

#### 监控用户权限
```sql
-- 监控用户需要 REPLICATION CLIENT 权限
GRANT REPLICATION CLIENT ON *.* TO 'monitor'@'%';
```

#### 网络延迟
```cnf
# 如果网络延迟较大，增加检查间隔
monitor_read_only_interval=3000  # 3 秒
```

### 8. 故障排查

#### ProxySQL 未检测到主从切换

```bash
# 1. 检查监控用户权限
mysql -hmysql-node1.internal -umonitor -pmonitor -e "SHOW VARIABLES LIKE 'read_only';"

# 2. 检查 ProxySQL 监控日志
mysql -h127.0.0.1 -P6032 -uadmin -padmin -e "
SELECT * FROM monitor.mysql_server_read_only_log
WHERE hostname='mysql-node1.internal'
ORDER BY time_start_us DESC LIMIT 5;
"

# 3. 手动触发检查
mysql -h127.0.0.1 -P6032 -uadmin -padmin -e "
UPDATE global_variables SET variable_value='1000' 
WHERE variable_name='monitor_read_only_interval';
LOAD MYSQL VARIABLES TO RUNTIME;
"
```

#### DNS 解析失败

```bash
# 测试 DNS 解析
nslookup mysql-node1.internal

# 测试 ProxySQL 容器内 DNS
docker exec proxysql nslookup mysql-node1.internal
```

## 总结

**方案 B（DNS + read_only）是生产环境最佳实践**：
- 配置简单
- 自动切换
- 应用无感知
- 易于维护
