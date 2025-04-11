#!/bin/env python3

import sys, requests

TEAM_TOKEN = "840ade564d40b2f82f11ef0d0087fda3"

# read flags from stdin
flags = [f.strip() for f in sys.stdin.readlines()]
r = requests.put("http://10.10.0.1:8080/flags", headers={"X-Team-Token": TEAM_TOKEN}, json=flags)
r = r.json()

for obj in r:
    status = "OK" if obj["status"] else "ERROR"
    flag = obj["flag"]
    msg = obj["msg"]

    print(flag, status, msg)
