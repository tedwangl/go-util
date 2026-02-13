"""
示例 DAG：简单的数据处理流程
"""
from datetime import datetime, timedelta
from airflow import DAG
from airflow.operators.bash import BashOperator
from airflow.operators.python import PythonOperator

# 默认参数
default_args = {
    'owner': 'airflow',
    'depends_on_past': False,
    'email_on_failure': False,
    'email_on_retry': False,
    'retries': 1,
    'retry_delay': timedelta(minutes=5),
}

# Python 任务函数
def extract_data(**context):
    """提取数据"""
    print("正在提取数据...")
    # 模拟数据提取
    data = {'records': 100, 'timestamp': datetime.now().isoformat()}
    return data

def transform_data(**context):
    """转换数据"""
    ti = context['ti']
    data = ti.xcom_pull(task_ids='extract')
    print(f"正在转换数据: {data}")
    # 模拟数据转换
    transformed = {**data, 'processed': True}
    return transformed

def load_data(**context):
    """加载数据"""
    ti = context['ti']
    data = ti.xcom_pull(task_ids='transform')
    print(f"正在加载数据: {data}")
    print("数据加载完成！")

# 定义 DAG
with DAG(
    'example_etl_pipeline',
    default_args=default_args,
    description='简单的 ETL 数据管道示例',
    schedule_interval=timedelta(days=1),
    start_date=datetime(2024, 1, 1),
    catchup=False,
    tags=['example', 'etl'],
) as dag:

    # 任务 1: 开始
    start = BashOperator(
        task_id='start',
        bash_command='echo "开始 ETL 流程"',
    )

    # 任务 2: 提取数据
    extract = PythonOperator(
        task_id='extract',
        python_callable=extract_data,
    )

    # 任务 3: 转换数据
    transform = PythonOperator(
        task_id='transform',
        python_callable=transform_data,
    )

    # 任务 4: 加载数据
    load = PythonOperator(
        task_id='load',
        python_callable=load_data,
    )

    # 任务 5: 结束
    end = BashOperator(
        task_id='end',
        bash_command='echo "ETL 流程完成"',
    )

    # 定义任务依赖关系
    start >> extract >> transform >> load >> end
