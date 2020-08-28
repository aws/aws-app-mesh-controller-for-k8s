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

@app.route('/tlsroute')
def default():
    print(request.headers)
    response = requests.get(f'http://backend-tls.tls-e2e.svc.cluster.local:8080/tlsroute')
    return response.text

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=config.PORT, debug=config.DEBUG_MODE)
