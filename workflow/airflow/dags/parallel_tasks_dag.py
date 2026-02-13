"""
并行任务示例：展示如何并行执行多个任务
"""
from datetime import datetime, timedelta
from airflow import DAG
from airflow.operators.python import PythonOperator
import time

default_args = {
    'owner': 'airflow',
    'retries': 1,
    'retry_delay': timedelta(minutes=1),
}

def process_task_a():
    """处理任务 A"""
    print("开始处理任务 A")
    time.sleep(2)
    print("任务 A 完成")
    return "A"

def process_task_b():
    """处理任务 B"""
    print("开始处理任务 B")
    time.sleep(2)
    print("任务 B 完成")
    return "B"

def process_task_c():
    """处理任务 C"""
    print("开始处理任务 C")
    time.sleep(2)
    print("任务 C 完成")
    return "C"

def aggregate_results(**context):
    """汇总结果"""
    ti = context['ti']
    result_a = ti.xcom_pull(task_ids='task_a')
    result_b = ti.xcom_pull(task_ids='task_b')
    result_c = ti.xcom_pull(task_ids='task_c')
    print(f"汇总结果: {result_a}, {result_b}, {result_c}")

with DAG(
    'parallel_tasks_example',
    default_args=default_args,
    description='并行任务执行示例',
    schedule_interval='@daily',
    start_date=datetime(2024, 1, 1),
    catchup=False,
    tags=['example', 'parallel'],
) as dag:

    start = PythonOperator(
        task_id='start',
        python_callable=lambda: print("开始并行处理"),
    )

    # 三个并行任务
    task_a = PythonOperator(
        task_id='task_a',
        python_callable=process_task_a,
    )

    task_b = PythonOperator(
        task_id='task_b',
        python_callable=process_task_b,
    )

    task_c = PythonOperator(
        task_id='task_c',
        python_callable=process_task_c,
    )

    # 汇总任务
    aggregate = PythonOperator(
        task_id='aggregate',
        python_callable=aggregate_results,
    )

    # 依赖关系：start -> [a, b, c] -> aggregate
    start >> [task_a, task_b, task_c] >> aggregate
