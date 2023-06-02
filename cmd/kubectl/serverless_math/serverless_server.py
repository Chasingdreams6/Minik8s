# serverless_server.py
from flask import Flask, request
import importlib
import schedule
import threading
import time
import requests
import os
import logging

app = Flask(__name__)
request_count = 0
last_request_time = time.time()

FAILED_TIME = 30
Request_20_second = 4

Function_name = ""


# 创建日志记录器
logger = logging.getLogger(__name__)
logger.setLevel(logging.INFO)

# 获取当前目录的绝对路径
current_dir = os.path.abspath(os.path.dirname(__file__))

# 日志文件路径
log_file_path = os.path.join(current_dir, "serverless_server.log")
file_handler = logging.FileHandler(log_file_path)
file_handler.setLevel(logging.INFO)

# 创建格式化器，设置日志格式
formatter = logging.Formatter('%(asctime)s - %(levelname)s - %(message)s')
file_handler.setFormatter(formatter)

# 将文件处理器添加到日志记录器
logger.addHandler(file_handler)

def reset_timer():
    global last_request_time
    last_request_time = time.time()


def check_timer():
    global last_request_time
    logger.info(f"Checking timer - current time: {time.time()}, last request time: {last_request_time}")
    if time.time() - last_request_time >= FAILED_TIME:
        logger.info("No requests received for 1 minute. Exiting...")
        # 进行异常退出的操作，例如抛出异常或者调用系统退出函数
        os._exit(666)  # 退出整个程序


count_lock = threading.Lock()


def reset_count():
    global request_count
    with count_lock:
        request_count = 0


def increment_count():
    global request_count
    with count_lock:
        request_count += 1



@app.route('/function/<string:module_name>/<string:function_name>', methods=['POST'])
def execute_function(module_name: str, function_name: str):
    global Function_name
    Function_name = function_name
    global request_count
    reset_timer()
    increment_count()

    if request_count == Request_20_second:
        j = {"message": "Exceeded request limit", "function_name": Function_name}
        logger.info(j)
        try:
            response = requests.post("http://192.168.1.7:8080/scale", json={"message": "Exceeded request limit", "function_name": Function_name}, timeout=3)
            response.raise_for_status()  # 检查请求是否成功
            logger.info("Notification sent successfully")
        except Exception as e:
            logger.info("Failed to send notification. %s", str(e))


    module = importlib.import_module(module_name)
    event = {"method": "http"}

    if request.headers.get('Content-Type') == 'application/json':
        context = request.get_json()
    else:
        context = request.form.to_dict()

    try:
        result = getattr(module, function_name)(event, context)
        return result, 200
    except Exception as e:
        logger.info("An error occurred during function execution:", e)
        return "Error during function execution", 500


# 创建一个新线程并启动该线程
def run_thread():
    while True:
        schedule.run_pending()

if __name__ == '__main__':
    job1 = schedule.every(2).seconds.do(check_timer)

    job2 = schedule.every(20).seconds.do(reset_count)

    t = threading.Thread(target=run_thread)
    t.start()

    app.run(host='0.0.0.0', port=8888, processes=True)
