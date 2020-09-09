import os
import requests
import config
from flask import Flask, request
from aws_xray_sdk.core import xray_recorder, patch_all
from aws_xray_sdk.ext.flask.middleware import XRayMiddleware

app = Flask(__name__)

xray_recorder.configure(
    context_missing='LOG_ERROR',
    service=config.XRAY_APP_NAME,
)
patch_all()
XRayMiddleware(app, xray_recorder)

@app.route('/defaultroute')
def default():
    print(request.headers)
    backend_url = 'http://' + config.BACKEND_TIMEOUT_HOST + ':' + str(config.PORT) + '/defaultroute'
    response = requests.get(backend_url)
    return response.text

@app.route('/timeoutroute')
def timeout():
    print(request.headers)
    backend_url = 'http://' + config.BACKEND_TIMEOUT_HOST + ':' + str(config.PORT) + '/timeoutroute'
    response = requests.get(backend_url)
    return response.text

@app.route('/tlsroute')
def tlsroute():
    print(request.headers)
    backend_url = 'http://' + config.BACKEND_TLS_HOST + ':' + str(config.PORT) + '/tlsroute'
    response = requests.get(backend_url)
    return response.text

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=config.PORT, debug=config.DEBUG_MODE)
