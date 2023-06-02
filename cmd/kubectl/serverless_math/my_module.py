# my_module.py
def calc(event: dict, context: dict)->dict:
    operator = context.get('operator')
    x = context.get('x')
    y = context.get('y')

    if operator and x is not None and y is not None:
        expression = f"{x} {operator} {y}"
        try:
            result = eval(expression)
            return {'result': result}
        except Exception as e:
            return {'error': str(e)}

    return {'error': 'Invalid input'}
