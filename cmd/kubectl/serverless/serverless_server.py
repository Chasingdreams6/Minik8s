# serverless_server.py
from flask import Flask, request
import importlib

app = Flask(__name__)

@app.route('/function/<string:module_name>/<string:function_name>', methods=['POST'])
def execute_function(module_name: str, function_name: str):
    module = importlib.import_module(module_name) # 动态导入当前目录下的名字为modul_name的模块
    event = {"method": "http"}	# 设置触发器参数，我们当前默认都是http触发
    # print(request.headers.get('Content-Type'))
    if request.headers.get('Content-Type') == 'application/json':
        context = request.get_json()
    else:
        context = request.form.to_dict() # 把POST中的form的参数转化为字典形式作为函数的参数
    # print(context)
    # eval函数就是执行对应的指令，此处就是动态执行module模块下名为function_name的函数，并且把两个参数传入得到返回值放到result中
    result = getattr(module, function_name)(event, context)
    return result, 200	# 把result放在http response中返回

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8888, processes=True)