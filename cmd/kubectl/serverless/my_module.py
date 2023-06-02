# my_module.py
def my_function(event: dict, context: dict)->dict:
    return {"result": "hello {}{}!".format(event, context)}