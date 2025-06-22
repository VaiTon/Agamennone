#!/bin/env python3

import sys
import requests

TEAM_TOKEN = "840ade564d40b2f82f11ef0d0087fda3"

# read flags from stdin
flags = [f.strip() for f in sys.stdin.readlines()]
res = requests.put(
    "http://10.10.0.1:8080/flags",
    headers={"X-Team-Token": TEAM_TOKEN},
    json=flags,
).json()


RESULTS = {
    "ACCEPTED": "OK",
    "DENIED": "ERROR",
    "RESUBMIT": "",  # empty string means the server will retry to submit the flag
    "ERROR": "ERROR",
}

for obj in res:
    flag = obj["flag"]
    msg = obj["msg"]
    status = RESULTS.get(obj["status"], "")

    print(flag, status, msg)
