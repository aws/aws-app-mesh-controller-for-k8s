from os import environ as env
import multiprocessing

PORT = int(env.get("PORT", 8080))
DEBUG_MODE = int(env.get("DEBUG_MODE", 0))
XRAY_APP_NAME = env.get('XRAY_APP_NAME', 'frontend')
BACKEND_TIMEOUT_HOST = env.get('BACKEND_TIMEOUT_HOST', 'backend.timeout-e2e.svc.cluster.local')
BACKEND_TLS_HOST = env.get('BACKEND_TLS_HOST', 'backend-tls.tls-e2e.svc.cluster.local')

# Gunicorn config
bind = ":" + str(PORT)
workers = multiprocessing.cpu_count() * 2 + 1
threads = 2 * multiprocessing.cpu_count()
