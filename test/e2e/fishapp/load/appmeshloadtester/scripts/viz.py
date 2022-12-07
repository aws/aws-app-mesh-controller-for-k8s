import io
import os
import sys

import altair as alt
import pandas as pd

from constants import *
from scripts.load_driver import get_s3_client

if __name__ == '__main__':
    test_name = sys.argv[1]
    driver_ts = sys.argv[2]

    s3_client = get_s3_client(region='us-west-2')
    response = s3_client.list_objects_v2(Bucket=S3_BUCKET,
                                         Prefix="/".join([test_name.rstrip("/"), driver_ts.rstrip("/")]), MaxKeys=100)

    if (response['ResponseMetadata']['HTTPStatusCode'] != 200 or 'Contents' not in response):
        print("Bucket or folder path incorrect. Please check inputs")
        exit(1)

    file_names = []
    full_df = pd.DataFrame(
        columns=["metric.kubernetes_pod_name", "timestamp", "metric_name", "metric_value", "normalized_ts"])
    for res in response['Contents']:
        if ("csv" in res['Key']):
            file_names.append(res['Key'].split("/")[-1])
        if ("rate" in res['Key']):
            # Pick only requests related metrics
            obj = s3_client.get_object(Bucket=S3_BUCKET, Key=res['Key'])
            df = pd.read_csv(io.BytesIO(obj['Body'].read()))
            metric_name = df.columns[-2]
            df.insert(loc=2, column="metric_name", value=metric_name)
            df.rename(columns={metric_name: "metric_value"}, inplace=True)
            if (not df.empty):
                print(res['Key'])
                full_df = pd.concat([full_df, df], ignore_index=True)

    full_df.rename(columns={"metric.kubernetes_pod_name": "kubernetes_pod_name"}, inplace=True)
    if ('metric.<rollup_column>' in full_df.columns):
        full_df.drop(columns='metric.<rollup_column>', inplace=True)

    # Altair
    # input_dropdown = alt.binding_select(options=full_df.metric_name.unique(), name='Metric')
    selection2 = alt.selection_multi(fields=['kubernetes_pod_name'], bind='legend')

    chart1 = alt.Chart(full_df.loc[full_df.metric_name == "envoy_ingress_rate_by_replica_set"],
                       title="envoy_ingress_rate_by_replica_set").mark_line().encode(
        x='normalized_ts',
        y='metric_value',
        color=alt.Color('kubernetes_pod_name'),
        tooltip=['kubernetes_pod_name', 'normalized_ts', 'metric_value'],
        opacity=alt.condition(selection2, alt.value(1), alt.value(0.2))
    ).add_selection(
        selection2
    )

    chart2 = alt.Chart(full_df.loc[full_df.metric_name == "envoy_2xx_requests_rate_by_replica_set"],
                       title="envoy_2xx_requests_rate_by_replica_set").mark_line().encode(
        x='normalized_ts',
        y='metric_value',
        color=alt.Color('kubernetes_pod_name'),
        tooltip=['kubernetes_pod_name', 'normalized_ts', 'metric_value'],
        opacity=alt.condition(selection2, alt.value(1), alt.value(0.2))
    ).add_selection(
        selection2
    )

    chart3 = alt.Chart(full_df.loc[full_df.metric_name == "envoy_5xx_requests_rate_by_replica_set"],
                       title="envoy_5xx_requests_rate_by_replica_set").mark_line().encode(
        x='normalized_ts',
        y='metric_value',
        color=alt.Color('kubernetes_pod_name'),
        tooltip=['kubernetes_pod_name', 'normalized_ts', 'metric_value'],
        opacity=alt.condition(selection2, alt.value(1), alt.value(0.2))
    ).add_selection(
        selection2
    )

    final = chart1 & (chart2 | chart3)
    final.save("viz.html")
    final.show()
