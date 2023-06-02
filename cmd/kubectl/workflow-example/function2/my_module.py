# Function2: multi
def function2(event: dict, context: dict)->dict:
    x = context.get('x')
    y = context.get('y')

    if x is not None and y is not None:
        try:
            result = x * y
            return {'result': result}
        except Exception as e:
            return {'error': str(e)}

    return {'error': 'Invalid input'}
