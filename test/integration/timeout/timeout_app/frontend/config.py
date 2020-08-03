from os import environ as env
import multiprocessing

PORT = int(env.get("PORT", 8080))
DEBUG_MODE = int(env.get("DEBUG_MODE", 0))
XRAY_APP_NAME = env.get('XRAY_APP_NAME', 'frontend')
BACKEND_HOST = env.get('BACKEND_HOST', 'backend.timeout-e2e.svc.cluster.local:8080')

# Gunicorn config
bind = ":" + str(PORT)
workers = multiprocessing.cpu_count() * 2 + 1
threads = 2 * multiprocessing.cpu_count()
