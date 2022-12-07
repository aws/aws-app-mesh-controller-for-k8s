import csv
import json
import os
import subprocess

import matplotlib.pyplot as plt
import numpy as np

from scripts.constants import S3_BUCKET

DIR_PATH = os.path.dirname(os.path.realpath(__file__))
DATA_PATH = os.path.join(DIR_PATH, 'data')


def get_s3_data():
    res = subprocess.run(["aws sts get-caller-identity"], shell=True, stdout=subprocess.PIPE,
                         universal_newlines=True)
    out = res.stdout
    print(out)

    command = "aws s3 sync s3://{} {}".format(S3_BUCKET, DATA_PATH)
    print("command: {}".format(command))
    res = subprocess.run([command], shell=True, stdout=subprocess.PIPE, universal_newlines=True)
    out = res.stdout
    print(out)


def plot_graph(actual_QPS_list, node_0_mem_list):
    print(actual_QPS_list)
    print(node_0_mem_list)
    node_0_mem_list = [float(x) for x in node_0_mem_list]
    Y = [x for _, x in sorted(zip(actual_QPS_list, node_0_mem_list))]
    X = sorted(actual_QPS_list)
    print(X)
    print(Y)
    xpoints = np.array(X)
    ypoints = np.array(Y)

    plt.figure(figsize=(10, 5))
    plt.bar(xpoints, ypoints, width=20)
    plt.ylabel('Node-0 (MiB)')
    plt.xlabel('Actual QPS')
    plt.show()


def read_data():
    res = [x for x in os.listdir(DIR_PATH) if os.path.isdir(x)]
    res_list = []
    for exp_f in res:
        result = [os.path.join(dp, f) for dp, dn, filenames in os.walk(os.path.join(DIR_PATH, exp_f)) for f in filenames
                  if "fortio.json" in f or "envoy_memory_MB_by_replica_set.csv" in f]
        res_list.append(result)

    actual_qps_list = []
    node_0_mem_list = []
    for res in res_list:
        for f in res:
            if "fortio.json" in f:
                with open(f) as json_f:
                    j = json.load(json_f)
                    actual_qps_list.append(j["ActualQPS"])
            else:
                with open(f) as csv_f:
                    c = csv.reader(csv_f, delimiter=',', skipinitialspace=True)
                    node_0_mem = []
                    for line in c:
                        if "node-0" in line[0]:
                            node_0_mem.append(line[2])
                    node_0_mem_list.append(max(node_0_mem))

    plot_graph(actual_qps_list, node_0_mem_list)


if __name__ == '__main__':
    get_s3_data()
    read_data()
